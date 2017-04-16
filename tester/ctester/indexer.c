#include <expat.h>
#include <sqlite3.h>
#include <stdio.h>
#include <unistd.h>
#include <magic.h>
#include <getopt.h>

#include "xml.c"
#include "string.c"

#define sqlite3_bind_text_name(S, N, V) (sqlite3_bind_text((S), sqlite3_bind_parameter_index((S), (N)), (V), -1, NULL))
#define sqlite3_bind_int_name(S, N, V)  (sqlite3_bind_int ((S), sqlite3_bind_parameter_index((S), (N)), (V)))

magic_t magic;

void usage() {
    char *usage_str = 
        "Accepts a list of xml files and indexes their contents into a sqlite database\n"
        "\n"
        "Usage: indexer index <outdb>\n"
        "       indexer query [-a] <indb> [<clause>]\n"
        "\n"
        "This works with tester/run.sh -d db.sqlite -q '1=1' so you can select\n"
        "xml files for the test based on specific criteria\n"
        "\n"
        "Options:\n"
        "  -a  Include all statuses in clause, not just OK.\n"
    ;
    fprintf(stderr, "%s", usage_str);
}

typedef XML_Char expat_ch;

enum idx_err {
    OK = 0,
    ERR = 1,
    ERR_OPEN = 2,
    ERR_PARSE_FAIL = 10,
    ERR_PARSE_STOPPED = 11,
    ERR_DB_EXISTS = 12,
    ERR_DB_NOT_EXISTS = 13,
    ERR_DB_OPEN = 14,
    ERR_DB_SELECT = 15,
    ERR_USAGE = 64,
};

struct ctx_err {
    char *msg;
    int code;
    void (*free)(void *p);
};

void ctx_err_deinit(struct ctx_err *err) {
    if (err->msg != NULL && err->free != NULL) {
        err->free(err->msg);
    }
    err->msg = NULL;
    err->free = NULL;
    err->code = OK;
}

int ctx_errf(struct ctx_err *err, int code, char *fmt, ...) {
    ctx_err_deinit(err);
    va_list args;
    va_start(args, fmt);
    err->code = code;
    err->free = free;
    vasprintf(&err->msg, fmt, args);
    va_end(args);
    return code;
}

int ctx_err(struct ctx_err *err, int code, char *error) {
    ctx_err_deinit(err);
    err->code = code;
    err->msg = error;
    return code;
}

struct current {
    char *raw_encoding;
    char *encoding;
    char *version;
    size_t bytes;
    size_t elems;
    size_t nselems;
    size_t attrs;
    size_t nsattrs;
    size_t comments;
    size_t comment_bytes;
    size_t comment_max;
    size_t cdatas;
    size_t pis;
    size_t dtds_public;
    size_t dtds_system;
    size_t dtd_elems;
    size_t dtd_attlists;
    size_t dtd_entities;
    size_t entity_refs;
    size_t entity_refs_dec;
    size_t entity_refs_hex;
    size_t notations;
    int max_depth;
    int depth;
};

void current_deinit(struct current *current) {
    free(current->raw_encoding);
    free(current->encoding);
    free(current->version);
}

struct index_ctx {
    struct current *current;
    struct ctx_err err;
    sqlite3_stmt *pstmt;
};

void dump_current(struct current *current) {
    printf("raw_\t%s\n"             , current->raw_encoding);
    printf("encoding\t%s\n"         , current->encoding);
    printf("version\t%s\n"          , current->version);
    printf("bytes\t%zu\n"           , current->bytes);
    printf("elems\t%zu\n"           , current->elems);
    printf("nselems\t%zu\n"         , current->nselems);
    printf("attrs\t%zu\n"           , current->attrs);
    printf("nsattrs\t%zu\n"         , current->nsattrs);
    printf("pis\t%zu\n"             , current->comments);
    printf("comments\t%zu\n"        , current->comments);
    printf("comment_bytes\t%zu\n"   , current->comment_bytes);
    printf("comment_max\t%zu\n"     , current->comment_max);
    printf("cdatas\t%zu\n"          , current->cdatas);
    printf("dtds_public\t%zu\n"     , current->dtds_public);
    printf("dtds_system\t%zu\n"     , current->dtds_system);
    printf("dtd_elems\t%zu\n"       , current->dtd_elems);
    printf("dtd_attlists\t%zu\n"    , current->dtd_attlists);
    printf("dtd_entities\t%zu\n"    , current->dtd_entities);
    printf("notations\t%zu\n"       , current->notations);
    printf("entity_refs\t%zu\n"     , current->entity_refs);
    printf("entity_refs_dec\t%zu\n" , current->entity_refs_dec);
    printf("entity_refs_hex\t%zu\n" , current->entity_refs_hex);
    printf("max_depth\t%d\n"        , current->max_depth);
}

void xml_elem_start(void *user_data, const expat_ch *name, const expat_ch **atts) {
    struct index_ctx *ctx = user_data;
    
    if (strchr(name, ':') > name) {
        ctx->current->nselems++;
    } else {
        ctx->current->elems++;
    }

    for (size_t i = 0; atts[i]; i+=2) {
        if (strchr(name, ':') > name) {
            ctx->current->nsattrs++;
        } else {
            ctx->current->attrs++;
        }
    }

    ctx->current->depth++;
    if (ctx->current->depth > ctx->current->max_depth) {
        ctx->current->max_depth = ctx->current->depth;
    }
}

void xml_elem_end(void *user_data, const expat_ch *name) {
    (void)name;
    struct index_ctx *ctx = user_data;
    ctx->current->depth--;
}

void xml_pi(void *user_data, const expat_ch *target, const expat_ch *data) {
    (void)target; (void)data;
    struct index_ctx *ctx = user_data;
    ctx->current->pis++;
}

void xml_decl(void *user_data,
    const expat_ch *version,
    const expat_ch *encoding,
    int             standalone
) {
    (void)standalone;
    struct index_ctx *ctx = user_data;

    ctx->current->encoding = encoding == NULL ? NULL : strdup(encoding);
    ctx->current->version  = version  == NULL ? NULL : strdup(version);
}

void xml_cdata_start(void *user_data) {
    struct index_ctx *ctx = user_data;
    ctx->current->cdatas++;
}

void xml_comment(void *user_data, const expat_ch *data) {
    struct index_ctx *ctx = user_data;
    ctx->current->comments++;
    size_t len = strlen(data);
    ctx->current->comment_bytes += len;
    if (ctx->current->comment_max < len) {
        ctx->current->comment_max = len;
    }
}

void xml_default(void *user_data, const expat_ch *s, int len) {
    struct index_ctx *ctx = user_data;
    if (len > 3 && s[0] == '&' && s[len-1] == ';') {
        if (s[1] == '#' && s[2] == 'x') {
            ctx->current->entity_refs_hex++;
        } else if (s[1] == '#') {
            ctx->current->entity_refs_dec++;
        } else {
            ctx->current->entity_refs++;
        }
    }
}

void xml_doctype_start(
    void *user_data,
    const expat_ch *doctype_name,
    const expat_ch *sysid,
    const expat_ch *pubid,
    int has_internal_subset
) {
    (void)doctype_name; (void)sysid; (void)pubid; (void)has_internal_subset;
    struct index_ctx *ctx = user_data;
    if (sysid != NULL && pubid != NULL) {
        ctx->current->dtds_public++;
    } else {
        ctx->current->dtds_system++;
    }
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
    (void)name; (void)is_parameter_entity; (void)value; (void)value_length;
    (void)base; (void)system_id; (void)public_id; (void)notation;
    struct index_ctx *ctx = user_data;
    ctx->current->dtd_entities++;
}

void xml_element_decl(void *user_data, const expat_ch *name, XML_Content *model) {
    (void)name; (void)model;
    struct index_ctx *ctx = user_data;
    ctx->current->dtd_elems++;
}

void xml_attlist_decl(
    void            *user_data,
    const expat_ch  *elname,
    const expat_ch  *attname,
    const expat_ch  *att_type,
    const expat_ch  *dflt,
    int              isrequired
) {
    (void)elname; (void)attname; (void)dflt; (void)att_type; (void)isrequired;
    struct index_ctx *ctx = user_data;
    ctx->current->dtd_attlists++;
}

void xml_notation(
    void *user_data,
    const expat_ch *name,
    const expat_ch *base,
    const expat_ch *system_id,
    const expat_ch *public_id
) {
    (void)name; (void)base; (void)system_id; (void)public_id;
    struct index_ctx *ctx = user_data;
    ctx->current->notations++;
}    

enum idx_err xml_index(struct index_ctx *ctx, char *file) {
    FILE *fp = fopen(file, "r");
    if (fp == NULL) {
        return ctx_errf(&ctx->err, ERR_OPEN, "could not open file %s", file);
    }
    const char *mime = magic_file(magic, file);
    if (strcmp(mime, "text/plain") != 0 &&
        strcmp(mime, "application/xml") != 0) {
        fclose(fp);
        return ctx_errf(&ctx->err, ERR_OPEN, "unsupported mime type %s", mime);
    }

    struct current current = {0};
    ctx->current = &current;

    XML_Parser parser = XML_ParserCreate(NULL);
    XML_SetUserData(parser, ctx);

    XML_SetElementHandler(parser, xml_elem_start, xml_elem_end);
    XML_SetXmlDeclHandler(parser, xml_decl);
    XML_SetProcessingInstructionHandler(parser, xml_pi);
    XML_SetCommentHandler(parser, xml_comment);
    XML_SetStartCdataSectionHandler(parser, xml_cdata_start);
    XML_SetEntityDeclHandler(parser, xml_entity_decl);
    XML_SetElementDeclHandler(parser, xml_element_decl);
    XML_SetAttlistDeclHandler(parser, xml_attlist_decl);
    XML_SetNotationDeclHandler(parser, xml_notation);
    XML_SetStartDoctypeDeclHandler(parser, xml_doctype_start);
    XML_SetDefaultHandler(parser, xml_default);

    /* XML_SetSkippedEntityHandler(parser, xml_skipped_entity); */

    // if we use this handler, entities won't get passed to XML_SetDefaultHandler
    /* XML_SetCharacterDataHandler(parser, xml_character_data); */

    /* XML_SetUnknownEncodingHandler(parser, xml_unknown_encoding, NULL); */

    {
        #define READ_SIZE 8192
        char buffer[READ_SIZE];
        bool done = false;
        size_t read = 0;
        while (!done) {
            size_t len = fread(&buffer, sizeof(char), READ_SIZE, fp);
            if (read == 0) {
                // extract raw encoding the hard way to account for expat
                // not being able to parse all encodings - we still want
                // an index.
                ctx->current->raw_encoding = raw_encoding_extract(buffer, len);
            }
            read += len;
            if (len < READ_SIZE) {
                done = true;
                ctx->current->bytes = read;
            }
            enum XML_Status status = XML_Parse(parser, buffer, len, done);

            XML_Index idx = XML_GetCurrentByteIndex(parser);
            if (idx < 0 || status != XML_STATUS_OK) {
                enum XML_Error err = XML_GetErrorCode(parser);

                ctx_errf(&ctx->err, ERR_PARSE_FAIL, 
                    "expat error %s(%d) before completion %ld != %zu", 
                    expat_errors[err], err, idx, read);
                goto cleanup;
            }

            size_t idx_sz = (size_t) idx;
            if (done) {
                if (idx_sz != read) {
                    enum XML_Error err = XML_GetErrorCode(parser);
                    ctx_errf(&ctx->err, ERR_PARSE_STOPPED,
                        "expat stopped %s(%d) before completion %ld != %zu", 
                        expat_errors[err], err, idx, read);
                    goto cleanup;
                }
            }
        }
    }

cleanup:
    sqlite3_reset(ctx->pstmt);
    sqlite3_bind_text_name(ctx->pstmt, ":file", file);
    sqlite3_bind_int_name (ctx->pstmt, ":status", ctx->err.code);
    sqlite3_bind_text_name(ctx->pstmt, ":msg", ctx->err.msg);
    sqlite3_bind_int_name (ctx->pstmt, ":bytes", ctx->current->bytes);
    sqlite3_bind_text_name(ctx->pstmt, ":encoding", ctx->current->encoding);
    sqlite3_bind_text_name(ctx->pstmt, ":raw_encoding", ctx->current->raw_encoding);
    sqlite3_bind_text_name(ctx->pstmt, ":version", ctx->current->version);
    sqlite3_bind_int_name (ctx->pstmt, ":elems", ctx->current->elems);
    sqlite3_bind_int_name (ctx->pstmt, ":nselems", ctx->current->nselems);
    sqlite3_bind_int_name (ctx->pstmt, ":attrs", ctx->current->attrs);
    sqlite3_bind_int_name (ctx->pstmt, ":nsattrs", ctx->current->nsattrs);
    sqlite3_bind_int_name (ctx->pstmt, ":comments", ctx->current->comments);
    sqlite3_bind_int_name (ctx->pstmt, ":comment_bytes", ctx->current->comment_bytes);
    sqlite3_bind_int_name (ctx->pstmt, ":comment_max", ctx->current->comment_max);
    sqlite3_bind_int_name (ctx->pstmt, ":cdatas", ctx->current->cdatas);
    sqlite3_bind_int_name (ctx->pstmt, ":pis", ctx->current->pis);
    sqlite3_bind_int_name (ctx->pstmt, ":dtds_public", ctx->current->dtds_public);
    sqlite3_bind_int_name (ctx->pstmt, ":dtds_system", ctx->current->dtds_system);
    sqlite3_bind_int_name (ctx->pstmt, ":dtd_elems", ctx->current->dtd_elems);
    sqlite3_bind_int_name (ctx->pstmt, ":dtd_attlists", ctx->current->dtd_attlists);
    sqlite3_bind_int_name (ctx->pstmt, ":dtd_entities", ctx->current->dtd_entities);
    sqlite3_bind_int_name (ctx->pstmt, ":entity_refs", ctx->current->entity_refs);
    sqlite3_bind_int_name (ctx->pstmt, ":entity_refs_dec", ctx->current->entity_refs_dec);
    sqlite3_bind_int_name (ctx->pstmt, ":entity_refs_hex", ctx->current->entity_refs_hex);
    sqlite3_bind_int_name (ctx->pstmt, ":notations", ctx->current->notations);
    sqlite3_bind_int_name (ctx->pstmt, ":max_depth", ctx->current->max_depth);
    sqlite3_step(ctx->pstmt);

    XML_ParserFree(parser);
    fclose(fp);
    current_deinit(&current);

    return ctx->err.code;
}

int cmd_index(int cargc, char *cargv[]) {
    struct index_ctx ctx = (struct index_ctx){0};

    if (cargc < 2) {
        return ERR_USAGE;
    }
    char *output = cargv[1];
    if (access(output, F_OK) != -1) {
        return ERR_DB_EXISTS;
    }

    sqlite3 *db;
    sqlite3_stmt *pstmt;
    if ((sqlite3_open(output, &db)) != SQLITE_OK) {
        return ERR_DB_OPEN;
    }

    char *err;
    // TODO: capture total bytes, bytes read
    if ((sqlite3_exec(db, 
        "CREATE TABLE xml("
            "file STRING PRIMARY KEY, status INT, msg TEXT, bytes INTEGER, raw_encoding STRING, "
            "encoding STRING, version STRING, elems INTEGER, "
            "nselems INTEGER, attrs INTEGER, nsattrs INTEGER, comments INTEGER, "
            "comment_bytes INTEGER, comment_max INTEGER, cdatas INTEGER, pis INTEGER, "
            "dtds_public INTEGER, dtds_system INTEGER, dtd_elems INTEGER, dtd_attlists INTEGER, "
            "dtd_entities INTEGER, entity_refs INTEGER, entity_refs_dec INTEGER, entity_refs_hex INTEGER, "
            "notations INTEGER, max_depth INTEGER);", NULL, NULL, &err)) != SQLITE_OK) {
        return ERR_DB_OPEN;
    }
    if ((sqlite3_prepare_v2(db, 
        "INSERT INTO xml ("
        "    file, status, msg, bytes, raw_encoding, encoding, version, elems, nselems, attrs, "
        "    nsattrs, comments, comment_bytes, comment_max, cdatas, "
        "    pis, dtds_public, dtds_system, dtd_elems, dtd_attlists, "
        "    dtd_entities, entity_refs, entity_refs_dec, entity_refs_hex, notations, "
        "    max_depth)"
        "VALUES ("
        "    :file, :status, :msg, :bytes, :raw_encoding, :encoding, :version, :elems, :nselems, :attrs, "
        "    :nsattrs, :comments, :comment_bytes, :comment_max, :cdatas, "
        "    :pis, :dtds_public, :dtds_system, :dtd_elems, :dtd_attlists, "
        "    :dtd_entities, :entity_refs, :entity_refs_dec, :entity_refs_hex, :notations, "
        "    :max_depth)", -1, &pstmt, NULL)) != SQLITE_OK) {
        return ERR_DB_OPEN;
    }

    int rc = OK;
    size_t cap;
    char *line = NULL;
    ssize_t len;

    ctx.pstmt = pstmt;

    while ((len = getline(&line, &cap, stdin)) > 0) {
        if (line[len-1] == '\n') {
            line[len-1] = 0;
        }

        ctx_err_deinit(&ctx.err);
        xml_index(&ctx, line);
        enum idx_err code = ctx.err.code;

        if (code != OK) {
            fprintf(stderr, "error: %s %d %s\n", line, code, ctx.err.msg);
            switch (code) {
            case ERR_OPEN:
            case ERR_PARSE_FAIL:
            case ERR_PARSE_STOPPED:
                // count skipped
                continue;
            default:
                rc = code;
                goto cleanup;
            }
        }
    }

    goto cleanup;

cleanup:
    ctx_err_deinit(&ctx.err);
    if (pstmt != NULL) {
        sqlite3_finalize(pstmt);
    }
    if (db != NULL) {
        sqlite3_close(db);
    }
    free(line);
    return rc;
}

int cmd_query(int cargc, char *cargv[]) {
    int rc = OK;
    sqlite3 *db = NULL;
    sqlite3_stmt *stmt = NULL;
    bool all = false;

    char c;
    while ((c = getopt(cargc, cargv, "a")) != -1) {
        switch (c) {
        case 'a' : all = true; break;
        default  : return ERR_USAGE;
        }
    }

    cargc -= optind; cargv += optind;
    if (cargc < 1 || cargc > 2) {
        return ERR_USAGE;
    }
    char *input = cargv[0];
    if (access(input, F_OK) == -1) {
        return ERR_DB_NOT_EXISTS;
    }

    if ((sqlite3_open(input, &db)) != SQLITE_OK) {
        rc = ERR_DB_OPEN; goto cleanup;
    }

    char *clause;
    if (cargc == 2 && strlen(cargv[1]) > 0) {
        clause = cargv[1];
    } else {
        clause = "1=1";
    }

    // TODO: status=0 should be disabled by a flag
    struct buf buf = {0};
    buf_strappend(&buf, "SELECT file FROM xml WHERE ");
    if (!all) {
        buf_strappend(&buf, "status=0 AND ");
    }
    buf_strappend(&buf, "(");
    buf_strappend(&buf, clause);
    buf_strappend(&buf, ")");

    if ((sqlite3_prepare_v2(db, buf.bytes, -1, &stmt, NULL)) != SQLITE_OK) {
        rc = ERR_DB_SELECT; goto cleanup;
    }

    while ((rc = sqlite3_step(stmt) == SQLITE_ROW)) {
        printf("%s\n", sqlite3_column_text(stmt, 0));
    }

cleanup:
    sqlite3_finalize(stmt);
    sqlite3_close(db);
    buf_deinit(&buf);
    return rc;
}

int main(int argc, char *argv[]) {
    int rc = OK;

    // TODO: work out why mime magic always says text/plain
    magic = magic_open(MAGIC_MIME_TYPE); 
    magic_load(magic, NULL);
    magic_compile(magic, NULL);

    if (argc < 2) {
        rc = ERR_USAGE; goto cleanup;
    }

    char *command = argv[1];
    if (strcmp(command, "index")==0) {
        rc = cmd_index(argc - 1, argv + 1);
    } else if (strcmp(command, "query")==0) {
        rc = cmd_query(argc - 1, argv + 1);
    } else {
        rc = ERR_USAGE; goto cleanup;
    }

cleanup:
    magic_close(magic);

    if (rc == ERR_USAGE) {
        usage();
    }
    return rc;
}

