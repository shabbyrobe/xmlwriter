#include <unistd.h>
#include <getopt.h>
#include <stdio.h>
#include <string.h>
#include <stdbool.h>
#include <assert.h>
#include <stdarg.h>
#include <ctype.h>

#include <expat.h>
#include <libxml/xmlwriter.h>

#include "string.c"
#include "xml.c"

typedef XML_Char expat_ch;
typedef xmlChar lxml_ch;

enum tb_err {
    TB_OK = 0,
    TB_ERR = 1,
    TB_ERR_UNHANDLED = 10,
    TB_ERR_PARSE_FAIL = 11,
    TB_ERR_PARSE_STOPPED = 12,
    TB_ERR_ARGS = 64, // I think this is EXIT_USAGE
};

void usage() {
    char *usage_str = 
        "Accepts xml from stdin and emits an xml tester script to stdout\n"
        "\n"
        "Usage: testbuilder [-sd]\n"
        "\n"
        "Options:\n"
        "  -d  Debug mode. Outputs the function and line that caused the\n"
        "      command to be written into the resulting test.\n"
        "  -s  Strip unnecessary whitespace. Experimental.\n"
        "\n"
        "Notes:\n"
        "  - If the parser encouters an error, there will still be invalid\n"
        "    xml flushed to stdout. For any exit status other than 0, assume\n"
        "    stdout can't be used\n"
    ;

    fprintf(stderr, "%s", usage_str);
}

struct xml_ctx {
    xmlTextWriterPtr writer;
    XML_Parser parser;
    bool debug;
    bool strip_ws;

    char *error;
    int error_code;
    void (*error_free)(void *p);

    // internal buffer {{{
    bool collect;
    char *content;
    size_t len;
    size_t sz;
    // }}}

    bool doc;
    bool in_dtd;
    bool in_attlist;

    const expat_ch **ns; size_t nslen; size_t nssz;
    const expat_ch **as; size_t aslen; size_t assz;
};

int xml_ctx_errf(struct xml_ctx *ctx, int code, char *fmt, ...) {
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

int xml_ctx_err(struct xml_ctx *ctx, int code, char *error) {
    if (ctx->error != NULL && ctx->error_free != NULL) {
        ctx->error_free(ctx->error);
    }
    ctx->error_code = code;
    ctx->error = error;
    return code;
}

void xml_ctx_deinit(struct xml_ctx *ctx) {
    if (ctx->error != NULL && ctx->error_free != NULL) {
        ctx->error_free(ctx->error);
    }
    free(ctx->content);
    ctx->sz = 0;
    free(ctx->ns);
    free(ctx->as);
}

void xml_ctx_clear(struct xml_ctx *ctx) {
    if (ctx->content != NULL) {
        *ctx->content = 0;
    }
    ctx->len = 0;
}

void xml_ctx_appendx(struct xml_ctx *ctx, const expat_ch *ch, int len) {
    if (ctx->collect) {
        size_t pos = ctx->len;
        if (ctx->content == NULL || (ctx->len + len + 1) >= ctx->sz) {
            ctx->sz += ctx->len + len + 1;
            size_t v = ctx->sz;
            v--; v |= v >> 1; v |= v >> 2; v |= v >> 4; v |= v >> 8; v |= v >> 16; v++;
            ctx->sz = v;
            ctx->content = realloc(ctx->content, ctx->sz);
        }
        ctx->len += len;
        memcpy(ctx->content + pos, ch, len);
        ctx->content[ctx->len] = 0;
    }
}

void xml_ctx_append(struct xml_ctx *ctx, const char *ch) {
    xml_ctx_appendx(ctx, ch, strlen(ch));
}

void command_start(struct xml_ctx *ctx, char *action, char *kind, const char *fn) {
    xmlTextWriterStartElement(ctx->writer, (lxml_ch*)"command");
    xmlTextWriterWriteAttribute(ctx->writer, (lxml_ch*)"action", (lxml_ch*)action);
    xmlTextWriterWriteAttribute(ctx->writer, (lxml_ch*)"kind", (lxml_ch*)kind);

    if (ctx->debug) {
        char buf[64];
        XML_Size ln = XML_GetCurrentLineNumber(ctx->parser);
        snprintf(buf, 64, "%lu", ln);
        xmlTextWriterWriteAttribute(ctx->writer, (lxml_ch*)"line", (lxml_ch*)buf);
        XML_Size pos = XML_GetCurrentByteIndex(ctx->parser);
        snprintf(buf, 64, "%lu", pos);
        xmlTextWriterWriteAttribute(ctx->writer, (lxml_ch*)"pos", (lxml_ch*)buf);
        xmlTextWriterWriteAttribute(ctx->writer, (lxml_ch*)"fn", (lxml_ch*)fn);
    }
}

void command_end(struct xml_ctx *ctx) {
    xmlTextWriterEndElement(ctx->writer);
}

void command_end_content(struct xml_ctx *ctx, const expat_ch *content) {
    if (content != NULL) {
        xmlTextWriterWriteString(ctx->writer, (lxml_ch*)content);
    }
    xmlTextWriterEndElement(ctx->writer);
}

void command_end_content_len(struct xml_ctx *ctx, const expat_ch *content, size_t len) {
    if (len > 0 && content != NULL) {
        lxml_ch *copy = xmlCharStrndup(content, len);
        xmlTextWriterWriteString(ctx->writer, copy);
        free(copy);
    }
    xmlTextWriterEndElement(ctx->writer);
}

void command_write(struct xml_ctx *ctx, char *action, char *kind, const char *fn) {
    command_start(ctx, action, kind, fn);
    command_end(ctx);
}

void command_attr(xmlTextWriterPtr writer, char *key, const expat_ch *value) {
    if (value != NULL) {
        // FIXME: are these casts actually OK as per the C standard?
        // expat_ch is 'char', but lxml_ch is 'unsigned char'
        xmlTextWriterWriteAttribute(writer, (lxml_ch*)key, (lxml_ch*)value);
    }
}

void command_attr_len(xmlTextWriterPtr writer, char *key, const expat_ch *value, size_t len) {
    if (len > 0 && value != NULL) {
        lxml_ch *copy = xmlCharStrndup(value, len);
        xmlTextWriterWriteAttribute(writer, (lxml_ch*)key, copy);
        free(copy);
    }
}

bool str_has_prefix(const char *str, const char *pre)
{
    return strncmp(pre, str, strlen(pre)) == 0;
}

void xml_elem_start(void *user_data, const expat_ch *name, const expat_ch **atts) {
    struct xml_ctx *ctx = user_data;

    command_start(ctx, "start", "elem", __FUNCTION__);
    
    char *sep = strchr(name, ':');
    const char *eprefix = NULL;
    if (sep != NULL) {
        *sep = 0;
        eprefix = name;
        name = sep + 1;
        command_attr(ctx->writer, "prefix", eprefix);
    }
    command_attr(ctx->writer, "name", name);
    
    // shortcut bailout if we have no attributes
    if (atts == NULL || *atts == NULL) {
        command_end(ctx);
        return;
    }
    
    size_t attlen = 0;
    for (size_t i = 0; atts[i]; i++) {
        attlen++;
    }

    if (ctx->ns == NULL || attlen + 1 >= ctx->nssz) {
        ctx->nssz = attlen + 1;
        ctx->ns = realloc(ctx->ns, sizeof(const expat_ch *) * ctx->nssz);
    }
    if (ctx->as == NULL || attlen + 1 >= ctx->assz) {
        ctx->assz = attlen + 1;
        ctx->as = realloc(ctx->as, sizeof(const expat_ch *) * ctx->assz);
    }
    ctx->aslen = 0;
    ctx->nslen = 0;

    for (size_t i = 0; atts[i]; i += 2) {
        const char *anm = atts[i];
        const char *avl = atts[i+1];

        if (str_has_prefix(anm, "xmlns:")) {
            anm+=6;
            if (eprefix != NULL && strcmp(anm, eprefix) == 0) {
                command_attr(ctx->writer, "uri", avl);
            } else {
                ctx->ns[ctx->nslen++] = anm;
                ctx->ns[ctx->nslen++] = avl;
            }
        } else {
            ctx->as[ctx->aslen++] = anm;
            ctx->as[ctx->aslen++] = avl;
        }
    }
    ctx->ns[ctx->nslen] = 0;
    ctx->as[ctx->aslen] = 0;

    // start element command end
    command_end(ctx);

    // attribute commands
    for (size_t i = 0; i < ctx->aslen; i += 2) {
        const char *anm = ctx->as[i];
        const char *avl = ctx->as[i+1];

        command_start(ctx, "write", "attr", __FUNCTION__);

        char *asep = strchr(anm, ':');
        if (asep != NULL) {
            char *aprefix = strndup(anm, asep - anm);
            anm = asep + 1;

            command_attr(ctx->writer, "prefix", aprefix);
            for (size_t j = 0; j < ctx->nslen; j += 2) {
                if (ctx->ns[j] != NULL && strcmp(ctx->ns[j], aprefix) == 0) {
                    command_attr(ctx->writer, "uri", ctx->ns[j+1]);
                    ctx->ns[j] = NULL;
                    break;
                }
            }
            free(aprefix);
        }

        command_attr(ctx->writer, "name", anm);
        command_end_content(ctx, avl);
    }

    // unclaimed prefixes
    for (size_t i = 0; i < ctx->nslen; i += 2) {
        if (ctx->ns[i] != NULL) {
            command_start(ctx, "write", "attr", __FUNCTION__);

            // this grim hack winds the pointer to ns[i] back the
            // length of xmlns, which we know is safe because we
            // wound it forward ourselves.
            command_attr(ctx->writer, "name", ctx->ns[i] - 6);

            command_end_content(ctx, ctx->ns[i+1]);
        }
    }
}

void xml_elem_end(void *user_data, const expat_ch *name) {
    struct xml_ctx *ctx = user_data;
    command_start(ctx, "end", "elem", __FUNCTION__);
    command_attr(ctx->writer, "name", name);
    command_end(ctx);
}

/* s is not 0 terminated */
void xml_character_data(void *user_data, const expat_ch *s, int len) {
    struct xml_ctx *ctx = user_data;
    if (ctx->collect) {
        xml_ctx_appendx(ctx, s, len);
    } else {
        if (ctx->in_attlist) {
            command_write(ctx, "end", "dtd-att-list", __FUNCTION__);
            ctx->in_attlist = false;
        }
        command_start(ctx, "write", "text", __FUNCTION__);
        command_end_content_len(ctx, s, len);
    }
}

/* target and data are 0 terminated */
void xml_pi(void *user_data, const expat_ch *target, const expat_ch *data) {
    struct xml_ctx *ctx = user_data;
    command_start(ctx, "write", "pi", __FUNCTION__);
    command_attr(ctx->writer, "target", target);
    command_end_content(ctx, data);
}

/* data is 0 terminated */
void xml_comment (void *user_data, const expat_ch *data) {
    struct xml_ctx *ctx = user_data;
    command_start(ctx, "write", "comment", __FUNCTION__);
    command_end_content(ctx, data);
}

void xml_cdata_start(void *user_data) {
    struct xml_ctx *ctx = user_data;
    ctx->collect = true;
    xml_ctx_clear(ctx);
}

void xml_cdata_end(void *user_data) {
    struct xml_ctx *ctx = user_data;
    command_start(ctx, "write", "cdata", __FUNCTION__);
    char *data = ctx->content;

    if (data == NULL) {
        data = "";
    }
    command_end_content(ctx, data);
    ctx->collect = false;
    xml_ctx_clear(ctx);
}

void xml_decl(void *user_data,
    const expat_ch *version,
    const expat_ch *encoding,
    int             standalone
) {
    struct xml_ctx *ctx = user_data;
    command_start(ctx, "start", "doc", __FUNCTION__);
    ctx->doc = true;
    if (version != NULL) {
        command_attr(ctx->writer, "version", version);
    }
    if (encoding != NULL) {
        command_attr(ctx->writer, "encoding", encoding);
    }
    if (standalone >= 0) {
        command_attr(ctx->writer, "standalone", standalone ? "yes" : "no");
    }
    command_end(ctx);
}

void xml_default(void *user_data, const expat_ch *s, int len) {
    struct xml_ctx *ctx = user_data;
    
    if (ctx->in_attlist) {
        command_write(ctx, "end", "dtd-att-list", __FUNCTION__);
        ctx->in_attlist = false;
    }

    // this is a nasty workaround for a bug in expat:
    //   https://sourceforge.net/p/expat/patches/88/
    //
    // "On an element declaration with a choice of three or more elements, i.e.
    // <!ELEMENT elm1 (elm2|elm3|elm4)> each pipe character (starting at the
    // second one) caused one default handler call before the element
    // declaration handler call, containing only the pipe character. So the
    // above example caused a default handler call with "|" as its data
    // followed by the expected element declaration handler call. Choices with
    // more elements caused more default handler calls. handleDefault was not
    // properly set to false in this case."
    //
    // it will also exclude any stray pipes which may be "legitimately"
    // present in the DTD from being included in the built test, but this
    // could be prevented by linting the incoming xml file before doing
    // comparison testing anyway - garbage in, garbage in between, garbage out.
    if (ctx->in_dtd) {
        if (strncmp(s, "|", len) == 0) {
            return;
        }
    }

    if (s != NULL) {
        // this is not well tested - consider it experimental.
        if (ctx->strip_ws) {
            bool is_ws = true;
            for (int i = 0; i < len; i++) {
                if (!isspace(s[i])) {
                    is_ws = false;
                    break;
                }
            }
            if (is_ws) {
                return;
            }
        }

        command_start(ctx, "write", "raw", __FUNCTION__);

        // next must always be true if we want this to work with
        // the ctester. this could be controlled by a flag.
        command_attr(ctx->writer, "next", "true");

        command_end_content_len(ctx, s, len);
    }
}

// this is static in libxml and not visible to us, so 
// of course that means copypasta!
void xmlDumpEntityContent(xmlBufferPtr buf, const xmlChar *content, int len) {
    if (buf->alloc == XML_BUFFER_ALLOC_IMMUTABLE) return;
    if (xmlStrchr(content, '%')) {
        const xmlChar * base, *cur;

        base = cur = content;
        for (int i = 0; i < len; i++) {
            if (*cur == '"') {
                if (base != cur)
                    xmlBufferAdd(buf, base, cur - base);
                xmlBufferAdd(buf, BAD_CAST "&quot;", 6);
                cur++;
                base = cur;
            } else if (*cur == '%') {
                if (base != cur)
                    xmlBufferAdd(buf, base, cur - base);
                    xmlBufferAdd(buf, BAD_CAST "&#37;", 5);
                    cur++;
                    base = cur;
            } else {
                cur++;
            }
        }
        if (base != cur)
            xmlBufferAdd(buf, base, cur - base);
    } else {
        xmlBufferWriteChar(buf, (char *)content);
    }
}

void xml_doctype_start(
    void *user_data,
    const expat_ch *doctypeName,
    const expat_ch *sysid,
    const expat_ch *pubid,
    int has_internal_subset
) {
    // TODO: how do we handle this?
    (void)has_internal_subset;

    struct xml_ctx *ctx = user_data;
    ctx->in_dtd = true;
    command_start(ctx, "start", "dtd", __FUNCTION__);
    command_attr(ctx->writer, "name", doctypeName);
    command_attr(ctx->writer, "system-id", sysid);
    command_attr(ctx->writer, "public-id", pubid);
    command_end(ctx);
}

void xml_doctype_end(void *user_data) {
    struct xml_ctx *ctx = user_data;
    ctx->in_dtd = false;
    if (ctx->in_attlist) {
        command_write(ctx, "end", "dtd-att-list", __FUNCTION__);
        ctx->in_attlist = false;
    }
    command_start(ctx, "end", "dtd", __FUNCTION__);
    command_end(ctx);
}

void xml_entity_decl (
    void *user_data,
    const expat_ch *name,
    int is_parameter_entity,
    const expat_ch *value,
    int value_length,
    const expat_ch *base,
    const expat_ch *system_id,
    const expat_ch *public_id,
    const expat_ch *notation
) {
    struct xml_ctx *ctx = user_data;
    if (ctx->in_attlist) {
        command_write(ctx, "end", "dtd-att-list", __FUNCTION__);
        ctx->in_attlist = false;
    }

    command_start(ctx, "write", "dtd-entity", __FUNCTION__);
    command_attr(ctx->writer, "name", name);
    command_attr(ctx->writer, "system-id", system_id);
    command_attr(ctx->writer, "public-id", public_id);
    command_attr(ctx->writer, "ndata-id", notation);
    if (is_parameter_entity) {
        command_attr(ctx->writer, "is-pe", "true");
    }

    if (base != NULL && strlen(base) > 0) {
        // FIXME: find out what this actually means, the docs don't say.
        xml_ctx_err(ctx, TB_ERR_UNHANDLED, "dtd-entity base found");
        XML_StopParser(ctx->parser, false);
        return;
    }

    bool raw = false;

    // HACK: special case for invisible unicode spaces in entity defs.
    // I suspect this will need to expand considerably as symbolic numeric
    // values are far likelier to appear in entities. This could be an
    // ongoing pain point for the tester - the following are semantically
    // identical and either are valid if they appear in the source: 
    //   <!ENTITY excl "&#33;"> == <!ENTITY excl "!">
    //
    // TODO: this is the job of a normaliser. scan all the xml files on
    // my hard drive looking for entities, then look at what the convention
    // is for encoding them. I suspect even in UTF-8 documents the convention
    // is to convert >127 chars to entities. Normaliser can have that as a rule.
    if (value_length == 2 && strcmp(value, "\xC2\xA0") == 0) {
        xmlTextWriterWriteString(ctx->writer, (lxml_ch*)"&#160;");
        value_length = 6;
        raw = true;
    }

    if (value_length > 0 && !raw) {
        // entities have special escape rules for the content - the %
        // sign must be escaped.
        xmlBufferPtr buf = xmlBufferCreate();
        xmlDumpEntityContent(buf, (lxml_ch*)value, value_length);
        xmlTextWriterWriteString(ctx->writer, xmlBufferContent(buf));
        xmlTextWriterEndElement(ctx->writer);

        xmlBufferFree(buf);
    } else {
        command_end(ctx);
    }
}


void xml_element_decl_child(struct xml_ctx *ctx, XML_Content *model) {
    char *sep = "";
    bool first = true;
    xml_ctx_append(ctx, "(");
    if (model->type == XML_CTYPE_MIXED) {
        xml_ctx_append(ctx, "#PCDATA");
        first = false;
    }
    switch (model->type) {
    case XML_CTYPE_MIXED  :
    case XML_CTYPE_CHOICE : sep = "|"; break;
    case XML_CTYPE_SEQ    : sep = ","; break;
    default:
        xml_ctx_errf(ctx, TB_ERR_UNHANDLED, "unexpected entity model type %d", model->type);
        XML_StopParser(ctx->parser, false);
        return;
    }
    for (size_t i = 0; i < model->numchildren; i++) {
        XML_Content *child = model->children + i;
        if (!first) {
            xml_ctx_append(ctx, sep);
        } else {
            first = false;
        }
        switch (child->type) {
        case XML_CTYPE_NAME:
            xml_ctx_append(ctx, child->name);
        break;
        case XML_CTYPE_MIXED:
        case XML_CTYPE_CHOICE:
        case XML_CTYPE_SEQ:
            xml_element_decl_child(ctx, child);
        break;
        default:
            // no dessert for you!
            xml_ctx_err(ctx, TB_ERR_UNHANDLED, "element decl bad child");
            XML_StopParser(ctx->parser, false);
            return;
        }
        switch (child->quant) {
        case XML_CQUANT_NONE : break;
        case XML_CQUANT_OPT  : xml_ctx_append(ctx, "?"); break;
        case XML_CQUANT_REP  : xml_ctx_append(ctx, "*"); break;
        case XML_CQUANT_PLUS : xml_ctx_append(ctx, "+"); break;
        default:
            xml_ctx_err(ctx, TB_ERR_UNHANDLED, "element decl bad quant");
            XML_StopParser(ctx->parser, false);
            return;
        }
    }
    xml_ctx_append(ctx, ")");
}

void xml_element_decl(void *user_data, const expat_ch *name, XML_Content *model) {
    struct xml_ctx *ctx = user_data;
    xmlTextWriterFlush(ctx->writer);

    if (ctx->in_attlist) {
        command_write(ctx, "end", "dtd-att-list", __FUNCTION__);
        ctx->in_attlist = false;
    }

    xml_ctx_clear(ctx);
    ctx->collect = true;
    command_start(ctx, "write", "dtd-elem", __FUNCTION__);
    command_attr(ctx->writer, "name", name);

    switch (model->type) {
    case XML_CTYPE_EMPTY:
        xml_ctx_append(ctx, "EMPTY");
    break;
    case XML_CTYPE_ANY:
        xml_ctx_append(ctx, "ANY");
    break;
    case XML_CTYPE_MIXED:
    case XML_CTYPE_SEQ:
    case XML_CTYPE_CHOICE:
        xml_element_decl_child(ctx, model);
        switch (model->quant) {
        case XML_CQUANT_NONE : break;
        case XML_CQUANT_OPT  : xml_ctx_append(ctx, "?"); break;
        case XML_CQUANT_REP  : xml_ctx_append(ctx, "*"); break;
        case XML_CQUANT_PLUS : xml_ctx_append(ctx, "+"); break;
        }
    break;

    default:
        xml_ctx_err(ctx, TB_ERR_UNHANDLED, "element decl bad child");
        XML_StopParser(ctx->parser, false);
        return;
    }

    if (ctx->content != NULL && ctx->len > 0) {
        command_end_content(ctx, ctx->content);
    } else {
        command_end(ctx);
    }
    ctx->collect = false;
    xml_ctx_clear(ctx);
}

void xml_attlist_decl(
    void            *user_data,
    const expat_ch  *elname,
    const expat_ch  *attname,
    const expat_ch  *att_type,
    const expat_ch  *dflt,
    int              isrequired
) {
    struct xml_ctx *ctx = user_data;
    if (!ctx->in_attlist) {
        command_start(ctx, "start", "dtd-att-list", __FUNCTION__);
        command_attr(ctx->writer, "name", elname);
        command_end(ctx);
        ctx->in_attlist = true;
    }
    command_start(ctx, "write", "dtd-attr", __FUNCTION__);
    command_attr(ctx->writer, "name", attname);
    command_attr(ctx->writer, "type", att_type);
    if (dflt != NULL) {
        command_attr(ctx->writer, "decl", dflt);
    }
    command_attr(ctx->writer, "required", isrequired ? "true" : "false");
    command_end(ctx);
}

void xml_notation(
    void *user_data,
    const expat_ch *name,
    const expat_ch *base,
    const expat_ch *system_id,
    const expat_ch *public_id
) {
    struct xml_ctx *ctx = user_data;
    if (ctx->in_attlist) {
        command_write(ctx, "end", "dtd-att-list", __FUNCTION__);
        ctx->in_attlist = false;
    }

    if (base != NULL && strlen(base) > 0) {
        // FIXME: find out what this actually means, the docs don't say.
        xml_ctx_err(ctx, TB_ERR_UNHANDLED, "notation base found");
        XML_StopParser(ctx->parser, false);
        return;
    }

    command_start(ctx, "write", "notation", __FUNCTION__);
    command_attr(ctx->writer, "name", name);
    command_attr(ctx->writer, "system-id", system_id);
    command_attr(ctx->writer, "public-id", public_id);
    command_end(ctx);
}    

#if 0
int xml_not_standalone (void *user_data) {
    struct xml_ctx *ctx = user_data;
    (void)ctx;

    fprintf(stderr, "NOT STANDALONE\n");
    return XML_STATUS_ERROR;
}
#endif

int xml_unknown_encoding(void *encoding_handler_data, const expat_ch *name, XML_Encoding *info) {
    (void)encoding_handler_data;
    (void)name;
    (void)info;
    fprintf(stderr, "UNKNOWN ENCODING\n");
    return XML_STATUS_ERROR;
}

size_t stdin_read(char **buffer) {
    const size_t READ_SIZE = 10;

    FILE *f = stdin;
    size_t cap = 0;
    size_t len = 0;
    size_t read = 0;

    do {
        cap += READ_SIZE;
        *buffer = realloc(*buffer, cap + 1);
        read = fread(*buffer + len, 1, READ_SIZE, f);
        len += read;
    } while (read == READ_SIZE);

    (*buffer)[len] = 0;

    return len + 1;
}

int main(int argc, char *argv[]) {
    int rc = 0;

    xmlTextWriterPtr writer = xmlNewTextWriterFilename("/dev/stdout", 0);
    xmlTextWriterSetIndent(writer, 1);

    XML_Parser parser = XML_ParserCreate(NULL);
    struct xml_ctx ctx = {
        .writer = writer,
        .parser = parser,
    };

    int ch;
    while ((ch = getopt(argc, argv, "sdh")) != -1) {
        switch (ch) {
        case 'd': ctx.debug = true; break;
        case 's': ctx.strip_ws = true; break;
        case 'h': usage(); goto cleanup; break;
        case '?': usage(); rc = TB_ERR_ARGS; goto cleanup; break;
        }
    }

    XML_SetUserData(parser, &ctx);

    XML_SetElementHandler(parser, xml_elem_start, xml_elem_end);
    XML_SetCharacterDataHandler(parser, xml_character_data);
    XML_SetProcessingInstructionHandler(parser, xml_pi);
    XML_SetCommentHandler(parser, xml_comment);
    XML_SetDefaultHandler(parser, xml_default);
    XML_SetCdataSectionHandler(parser, xml_cdata_start, xml_cdata_end);
    XML_SetDoctypeDeclHandler(parser, xml_doctype_start, xml_doctype_end);

    XML_SetEntityDeclHandler(parser, xml_entity_decl);
    XML_SetXmlDeclHandler(parser, xml_decl);
    XML_SetElementDeclHandler(parser, xml_element_decl);
    XML_SetAttlistDeclHandler(parser, xml_attlist_decl);
    XML_SetNotationDeclHandler(parser, xml_notation);

    XML_SetUnknownEncodingHandler(parser, xml_unknown_encoding, NULL);

    // marked as obsolete in expat.h
    /* XML_SetUnparsedEntityDeclHandler(parser, xml_unparsed_entity); */

    // we just want this to be dumped by the default handler.
    // see /opt/local/share/docbook2X/charmaps/roff.charmap.xml for a file which
    // triggered this
    /* XML_SetSkippedEntityHandler(parser, xml_skipped_entity); */

    // we just want to pass these through as they are, we don't care if
    // they're external
    /* XML_SetExternalEntityRefHandler(parser, xml_external_entity_ref); */

    // we don't care if it's standalone or not - we are just trying
    // to emit a representation of the file as is
    /* XML_SetNotStandaloneHandler(parser, xml_not_standalone); */

    xmlTextWriterStartDocument(writer, "1.0", "UTF-8", NULL);
    xmlTextWriterStartElement(writer, (lxml_ch*)"script");

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

                xml_ctx_errf(&ctx, TB_ERR_PARSE_FAIL, 
                    "expat error %s(%d) before completion %ld != %zu, byte %zu\n", 
                    expat_errors[err], err, idx, len, read);
                goto cleanup;
            }

            size_t idx_sz = (size_t) idx;
            if (done) {
                if (idx_sz != read) {
                    enum XML_Error err = XML_GetErrorCode(parser);
                    xml_ctx_errf(&ctx, TB_ERR_PARSE_STOPPED,
                        "expat stopped %s(%d) before completion %ld != %zu, byte %zu\n", 
                        expat_errors[err], err, idx, len, read);
                    goto cleanup;
                }
            }
        }
    }

    if (ctx.doc) {
        command_start(&ctx, "end", "doc", __FUNCTION__);
        command_end(&ctx);
    }

    xmlTextWriterEndElement(writer);
    xmlTextWriterEndDocument(writer);
    xmlTextWriterFlush(writer);

cleanup:
    if (ctx.error_code > 0) {
        rc = ctx.error_code;
        if (ctx.error != NULL) {
            fprintf(stderr, "%s\n", ctx.error);
        } else {
            fprintf(stderr, "parsing failed with unknown error %d\n", ctx.error_code);
        }
    }

    XML_ParserFree(parser);
    xml_ctx_deinit(&ctx);
    xmlFreeTextWriter(writer);
    xmlCleanupParser();

    return rc;
}
