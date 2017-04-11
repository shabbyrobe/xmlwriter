/**
 * Sorting attributes yields semantically identical XML that diffs
 * more cleanly.
 */

#include <unistd.h>
#include <stdio.h>
#include <string.h>
#include <stdbool.h>
#include <assert.h>

#include <expat.h>
#include <libxml/xmlwriter.h>

typedef xmlChar lxml_ch;
typedef XML_Char expat_ch;

char *error = NULL;
bool debug = false;

#ifndef HAVE_STRNDUP
char *strndup(const char *s, size_t n)
{
    char* new = malloc(n+1);
    if (new) {
        strncpy(new, s, n);
        new[n] = '\0';
    }
    return new;
}
#endif

struct attr {
    const char *name;
    const char *value;
};

struct xml_ctx {
    xmlTextWriterPtr writer;
    XML_Parser parser;

    bool self_close;
    struct attr *attrs;
    size_t attrs_cap;
};

bool str_has_prefix(const char *str, const char *pre)
{
    return strncmp(pre, str, strlen(pre)) == 0;
}

int attr_comp(const void *a, const void *b) {
    const struct attr *aa = a;
    const struct attr *bb = b;
    return strcmp(aa->name, bb->name);
}

void xml_elem_end(void *user_data, const expat_ch *name) {
    (void)name;
    struct xml_ctx *ctx = user_data;
    if (ctx->self_close) {
        xmlTextWriterEndElement(ctx->writer);
    } else {
        xmlTextWriterFullEndElement(ctx->writer);
    }
    ctx->self_close = false;
}

void xml_elem_start(void *user_data, const expat_ch *name, const expat_ch **atts) {
    struct xml_ctx *ctx = user_data;
    ctx->self_close = true;

    xmlTextWriterStartElement(ctx->writer, (lxml_ch*)name);
    if (atts == NULL || *atts == NULL) {
        return;
    }

    size_t attlen = 0;
    for (size_t i = 0; atts[i]; i+=2) {
        attlen++;
    }

    if (attlen > ctx->attrs_cap) {
        while (ctx->attrs_cap < attlen) {
            ctx->attrs_cap *= 2;
            ctx->attrs = realloc(ctx->attrs, sizeof(struct attr) * ctx->attrs_cap);
        }
    }

    for (size_t i = 0, j = 0; atts[i]; j+=1, i+=2) {
        ctx->attrs[j].name = atts[i];
        ctx->attrs[j].value = atts[i+1];
    }
    
    qsort(ctx->attrs, attlen, sizeof(struct attr), attr_comp);

    for (size_t i = 0; i < attlen; i++) {
        xmlTextWriterWriteAttribute(ctx->writer, 
            (lxml_ch*)ctx->attrs[i].name, (lxml_ch*)ctx->attrs[i].value);
    }
}

void xml_default(void *user_data, const expat_ch *s, int len) {
    struct xml_ctx *ctx = user_data;
    ctx->self_close = false;

    lxml_ch *copy = xmlCharStrndup(s, len);
    xmlTextWriterWriteRaw(ctx->writer, copy);
    free(copy);
}

int xml_unknown_encoding(void *encoding_handler_data, const expat_ch *name, XML_Encoding *info) {
    (void)encoding_handler_data;
    (void)name;
    (void)info;
    fprintf(stderr, "UNKNOWN ENCODING\n");
    return XML_STATUS_ERROR;
}

int main() {
    int rc = 0;
    xmlTextWriterPtr writer = xmlNewTextWriterFilename("/dev/stdout", 0);

    XML_Parser parser = XML_ParserCreate("UTF-8");

    struct xml_ctx ctx = {
        .writer = writer,
        .parser = parser,
        .attrs = calloc(sizeof(struct attr), 32),
        .attrs_cap = 32,
    };
    XML_SetUserData(parser, &ctx);

    XML_SetElementHandler(parser, xml_elem_start, xml_elem_end);
    XML_SetDefaultHandler(parser, xml_default);
    XML_SetUnknownEncodingHandler(parser, xml_unknown_encoding, NULL);

    {
        #define READ_SIZE 8192
        char buffer[READ_SIZE];
        bool done = false;
        size_t read = 0;
        while (!done) {
            size_t len = fread(&buffer, sizeof(char), READ_SIZE, stdin);
            if (len < READ_SIZE) {
                done = true;
            }
            read += len;
            enum XML_Status status = XML_Parse(parser, buffer, len, done);

            XML_Index idx = XML_GetCurrentByteIndex(parser);
            if (idx < 0 || status != XML_STATUS_OK) {
                fprintf(stderr, "Error parsing stopped before completion %ld != %zu, byte %zu\n", idx, len, read);
                rc = 1;
                goto cleanup;
            }

            size_t idx_sz = (size_t) idx;
            if (done) {
                /* XML_ParsingStatus ps = {}; */
                if (idx_sz != read) {
                    fprintf(stderr, "Parsing stopped before completion %zu != %zu, byte %zu\n", idx_sz, len, read);
                    rc = 1;
                    goto cleanup;
                }
            }
        }
    }

    xmlTextWriterFlush(writer);

cleanup:
    free(ctx.attrs);
    XML_ParserFree(parser);
    xmlFreeTextWriter(writer);

    return rc;
}
