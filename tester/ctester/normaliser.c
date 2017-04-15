/**
 * XML normaliser
 *
 * - Sorts attributes alphabetically
 * - Ensures numeric entities all have the same style of representation
 *   (decimal &#1234; or hex &#x89ab;)
 */

#include <unistd.h>
#include <stdio.h>
#include <string.h>
#include <stdbool.h>
#include <assert.h>
#include <ctype.h>

#include <expat.h>
#include <libxml/xmlwriter.h>
#include "xml.c"
#include "string.c"

typedef xmlChar lxml_ch;
typedef XML_Char expat_ch;

enum n_err {
    N_OK = 0,
    N_ERR = 1,
    N_ERR_ENCODING = 10,
    N_ERR_PARSE_FAIL = 11,
    N_ERR_PARSE_STOPPED = 12,
};

enum ent_num_mode {
    ENT_NUM_LEAVE = 0,
    ENT_NUM_DEC = 1,
    ENT_NUM_HEX = 2,
};

bool debug = false;

struct attr {
    const char *name;
    const char *value;
};

struct ctx {
    xmlTextWriterPtr writer;
    XML_Parser parser;

    enum ent_num_mode ent_num_mode;
    char *error;
    int error_code;
    void (*error_free)(void *p);

    bool self_close;
    struct attr *attrs;
    size_t attrs_cap;
};

static int ctx_errf(struct ctx *ctx, int code, char *fmt, ...) {
    if (ctx->error != NULL && ctx->error_free != NULL) {
        ctx->error_free(ctx->error);
    }
    va_list args;
    va_start(args, fmt);
    ctx->error_code = code;
    ctx->error_free = free;
    vasprintf(&ctx->error, fmt, args);
    va_end(args);
    return code;
}

int ctx_err(struct ctx *ctx, int code, char *error) {
    if (ctx->error != NULL && ctx->error_free != NULL) {
        ctx->error_free(ctx->error);
    }
    ctx->error_code = code;
    ctx->error = error;
    return code;
}

void ctx_deinit(struct ctx *ctx) {
    if (ctx->error != NULL && ctx->error_free != NULL) {
        ctx->error_free(ctx->error);
    }
}

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
    struct ctx *ctx = user_data;
    if (ctx->self_close) {
        xmlTextWriterEndElement(ctx->writer);
    } else {
        xmlTextWriterFullEndElement(ctx->writer);
    }
    ctx->self_close = false;
}

void xml_elem_start(void *user_data, const expat_ch *name, const expat_ch **atts) {
    struct ctx *ctx = user_data;
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
    struct ctx *ctx = user_data;
    ctx->self_close = false;
    lxml_ch *copy;

    // expat seems to deserialise entities in discrete chunks, i.e.  not as
    // part of the character data stream. so we can (hopefully) assume that the
    // default handler gets them complete as a single unit.
    if (ctx->ent_num_mode != ENT_NUM_LEAVE && len > 4 && s[0] == '&' && s[1] == '#' && s[len-1] == ';') {
        int base = 10;
        const char *start = s + 2;
        if (s[2] == 'x') {
            base = 16;
            start++;
        }
        if (base == 16 && ctx->ent_num_mode == ENT_NUM_HEX) {
            goto std;
        } else if (base == 10 && ctx->ent_num_mode == ENT_NUM_DEC) {
            goto std;
        }

        char *end;
        long ent = strtoul(start, &end, base);
        if (*end != *(s + len-1)) {
            // if it doesn't parse up to the semicolon, just handle it like normal.
            goto std;
        }

        // cheat! just allocate double the memory rather than waste time 
        // working out how to calculate hex to dec sizes
        copy = xmlMalloc(len * 2);
        sprintf((char *)copy, (ctx->ent_num_mode == ENT_NUM_DEC ? "&#%ld;" : "&#x%lx;"), ent);
        goto print;
    } 

std:;
    copy = xmlCharStrndup(s, len);
print:;
    xmlTextWriterWriteRaw(ctx->writer, copy);
    free(copy);
}

int xml_unknown_encoding(void *encoding_handler_data, const expat_ch *name, XML_Encoding *info) {
    (void)name;
    (void)info;
    struct ctx *ctx = encoding_handler_data;
    ctx_err(ctx, N_ERR_ENCODING, "unknown encoding");
    return XML_STATUS_ERROR;
}

int main() {
    int rc = 0;
    xmlTextWriterPtr writer = xmlNewTextWriterFilename("/dev/stdout", 0);

    XML_Parser parser = XML_ParserCreate(NULL);

    struct ctx ctx = {
        .writer = writer,
        .parser = parser,
        .attrs = calloc(sizeof(struct attr), 32),
        .attrs_cap = 32,
        .ent_num_mode = ENT_NUM_HEX,
    };
    XML_SetUserData(parser, &ctx);

    XML_SetElementHandler(parser, xml_elem_start, xml_elem_end);
    XML_SetDefaultHandler(parser, xml_default);
    XML_SetUnknownEncodingHandler(parser, xml_unknown_encoding, &ctx);

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
                enum XML_Error err = XML_GetErrorCode(parser);

                ctx_errf(&ctx, N_ERR_PARSE_FAIL, 
                    "expat error %s(%d) before completion %ld != %zu, byte %zu\n", 
                    expat_errors[err], err, idx, len, read);
                goto cleanup;
            }

            size_t idx_sz = (size_t) idx;
            if (done) {
                if (idx_sz != read) {
                    enum XML_Error err = XML_GetErrorCode(parser);
                    ctx_errf(&ctx, N_ERR_PARSE_STOPPED,
                        "expat stopped %s(%d) before completion %ld != %zu, byte %zu\n", 
                        expat_errors[err], err, idx, len, read);
                    goto cleanup;
                }
            }
        }
    }

    xmlTextWriterFlush(writer);

cleanup:
    ctx_deinit(&ctx);
    free(ctx.attrs);
    XML_ParserFree(parser);
    xmlFreeTextWriter(writer);

    return rc;
}
