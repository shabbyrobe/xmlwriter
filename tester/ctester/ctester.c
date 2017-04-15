#include <unistd.h>
#include <stdio.h>
#include <string.h>
#include <stdbool.h>
#include <assert.h>

#include <libxml/xmlreader.h>
#include <libxml/xmlwriter.h>

#include "string.c"

enum ct_err {
    CT_OK = 0,
    CT_ERR = 1,
};

const xmlChar *kind_all_str           = (xmlChar*)"all";
const xmlChar *kind_attr_str          = (xmlChar*)"attr";
const xmlChar *kind_cdata_str         = (xmlChar*)"cdata";
const xmlChar *kind_cdata_content_str = (xmlChar*)"cdata-content";
const xmlChar *kind_comment_str       = (xmlChar*)"comment";
const xmlChar *kind_doc_str           = (xmlChar*)"doc";
const xmlChar *kind_dtd_str           = (xmlChar*)"dtd";
const xmlChar *kind_dtd_attr_str      = (xmlChar*)"dtd-attr";
const xmlChar *kind_dtd_attlist_str   = (xmlChar*)"dtd-att-list";
const xmlChar *kind_dtd_elem_str      = (xmlChar*)"dtd-elem";
const xmlChar *kind_dtd_entity_str    = (xmlChar*)"dtd-entity";
const xmlChar *kind_elem_str          = (xmlChar*)"elem";
const xmlChar *kind_notation_str      = (xmlChar*)"notation";
const xmlChar *kind_pi_str            = (xmlChar*)"pi";
const xmlChar *kind_raw_str           = (xmlChar*)"raw";
const xmlChar *kind_text_str          = (xmlChar*)"text";

const xmlChar *action_start_str       = (xmlChar*)"start";
const xmlChar *action_end_str         = (xmlChar*)"end";
const xmlChar *action_write_str       = (xmlChar*)"write";

const xmlChar *ws_mode_strip_str      = (xmlChar*)"strip";

enum ws_mode {
    ws_mode_none,
    ws_mode_strip,
};

enum kind {
    kind_all,
	kind_attr,
	kind_cdata,
	kind_cdata_content,
	kind_comment,
	kind_doc,
	kind_dtd,
    kind_dtd_attr,
	kind_dtd_attlist,
	kind_dtd_elem,
	kind_dtd_entity,
	kind_elem,
    kind_notation,
	kind_pi,
	kind_raw,
    kind_text,
};

char *kind_string(enum kind kind) {
    switch (kind) {
    case kind_all           : return "kind_all";
	case kind_attr          : return "kind_attr";
	case kind_cdata         : return "kind_cdata";
	case kind_cdata_content : return "kind_cdata_content";
	case kind_comment       : return "kind_comment";
	case kind_doc           : return "kind_doc";
	case kind_dtd           : return "kind_dtd";
    case kind_dtd_attr      : return "kind_dtd_attr";
	case kind_dtd_attlist   : return "kind_dtd_attlist";
	case kind_dtd_elem      : return "kind_dtd_elem";
	case kind_dtd_entity    : return "kind_dtd_entity";
	case kind_elem          : return "kind_elem";
    case kind_notation      : return "kind_notation";
	case kind_pi            : return "kind_pi";
	case kind_raw           : return "kind_raw";
    case kind_text          : return "kind_text";
    default                 : return "";
    }
}

enum kind kind_from_xml(const xmlChar *in) {
    if (xmlStrcmp(in, kind_all_str) == 0)           { return kind_all; }
    if (xmlStrcmp(in, kind_attr_str) == 0)          { return kind_attr; }
    if (xmlStrcmp(in, kind_cdata_str) == 0)         { return kind_cdata; }
    if (xmlStrcmp(in, kind_cdata_content_str) == 0) { return kind_cdata_content; }
    if (xmlStrcmp(in, kind_comment_str) == 0)       { return kind_comment; }
    if (xmlStrcmp(in, kind_doc_str) == 0)           { return kind_doc; }
    if (xmlStrcmp(in, kind_dtd_str) == 0)           { return kind_dtd; }
    if (xmlStrcmp(in, kind_dtd_attr_str) == 0)      { return kind_dtd_attr; }
    if (xmlStrcmp(in, kind_dtd_attlist_str) == 0)   { return kind_dtd_attlist; }
    if (xmlStrcmp(in, kind_dtd_elem_str) == 0)      { return kind_dtd_elem; }
    if (xmlStrcmp(in, kind_dtd_entity_str) == 0)    { return kind_dtd_entity; }
    if (xmlStrcmp(in, kind_elem_str) == 0)          { return kind_elem; }
    if (xmlStrcmp(in, kind_notation_str) == 0)      { return kind_notation; }
    if (xmlStrcmp(in, kind_pi_str) == 0)            { return kind_pi; }
    if (xmlStrcmp(in, kind_raw_str) == 0)           { return kind_raw; }
    if (xmlStrcmp(in, kind_text_str) == 0)          { return kind_text; }
    return -1;
}

enum action {
    action_start,
    action_write,
    action_end,
};

char *action_string(enum action action ) {
    switch (action) {
    case action_start : return "action_start";
    case action_write : return "action_write";
    case action_end   : return "action_end";
    default           : return "";
    }
}

xmlChar *ws_strip(xmlChar *in) {
    // FIXME: there has to be a better way to allocate this.
    xmlChar *new = xmlStrdup(in);
    xmlChar *read = in;
    xmlChar *write = new;
    
    // trim start
    for (; *read; read++) {
        if (*read != '\n' && *read != '\r' && *read != '\t' && *read != ' ') {
            break;
        }
    }

    int c = 0;
    for (; *read; read++) {
        if (*read != '\n' && *read != '\r' && *read != '\t' && *read != ' ') {
            c = 0;
            *write = *read; write++;
        } else {
            c++;
            if (c == 1) {
                *write = ' '; write++;
            }
        }
    }

    if (write > new && *(write - 1) == ' ') {
        *(write - 1) = '\0';
    } else {
        *write = '\0';
    }
    return new;
}


int bool_from_xml(xmlChar *is_pe, bool *value) {
    if (xmlStrcmp(is_pe, (xmlChar*)"true") == 0) {
        *value = true;
    } else if (xmlStrcmp(is_pe, (xmlChar*)"false") == 0) {
        *value = false;
    } else if (xmlStrcmp(is_pe, (xmlChar*)"yes") == 0) {
        *value = true;
    } else if (xmlStrcmp(is_pe, (xmlChar*)"no") == 0) {
        *value = false;
    } else {
        return 1;
    }
    return 0;
}

struct command {
    enum action action;
    enum kind kind;
    xmlChar *name;
    xmlChar *content;
    enum ws_mode ws_mode;
    struct params *params;
    struct command *next;
};

struct script {
    xmlChar *name;
    struct command *command;
    struct command *tail;
};

struct ctx_end {
    struct params *params;
    struct ctx_end *parent;
};

struct ct_ctx {
    xmlTextWriterPtr writer;
    struct command *cmd;
    struct params *params;
    struct ctx_end *end;
    bool in_dtd;

    char *error;
    int error_code;
    void (*error_free)(void *p);
};

struct params {
    void (*free)(void *p);
    int (*run)(struct ct_ctx *ctx);
};

void ctx_end_push(struct ct_ctx *ctx, struct params *params) {
    struct ctx_end *end = malloc(sizeof(struct ctx_end));
    end->params = params;
    end->parent = ctx->end;
    ctx->end = end;
}

int ctx_errf(struct ct_ctx *ctx, int code, char *fmt, ...) {
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

int ctx_err(struct ct_ctx *ctx, int code, char *error) {
    if (ctx->error != NULL && ctx->error_free != NULL) {
        ctx->error_free(ctx->error);
    }
    ctx->error_code = code;
    ctx->error = error;
    return code;
}

// this deletes nodes while their run method might still be executing.
void ctx_end_pop(struct ct_ctx *ctx) {
    if (ctx->end != NULL) {
        struct ctx_end *last = ctx->end;
        ctx->end = ctx->end->parent;
        last->params->free(last->params);
        free(last);
    }
}

void ctx_end(struct ct_ctx *ctx) {
    struct ctx_end *end = ctx->end;
    while (end) {
        // end->parent will get freed by end->params->run
        struct ctx_end *next = end->parent;
        ctx->params = end->params;
        end->params->free(end->params);
        free(end);
        end = next;
    }
}

void ctx_deinit(struct ct_ctx *ctx) {
    ctx_end(ctx);
    if (ctx->error != NULL && ctx->error_free != NULL) {
        ctx->error_free(ctx->error);
    }
}

// {{{ writeattr
struct params_write_attr {
    struct params params;
    xmlChar *prefix;
    xmlChar *uri;
};

int params_write_attr_run(struct ct_ctx *ctx) {
    struct params_write_attr *p = (struct params_write_attr *)ctx->params;
    int b = 0;
    if (p->prefix != NULL || p->uri != NULL) {
        b = xmlTextWriterWriteAttributeNS(ctx->writer, p->prefix, ctx->cmd->name, p->uri, ctx->cmd->content);
    } else {
        b = xmlTextWriterWriteAttribute(ctx->writer, ctx->cmd->name, ctx->cmd->content);
    }
    return b < 0;
}

void params_write_attr_free(void *f) {
    struct params_write_attr *p = f;
    if (p != NULL) {
        xmlFree(p->prefix);
        xmlFree(p->uri);
    }
    free(p);
}

struct params_write_attr *params_write_attr_create(
    xmlChar *prefix,
    xmlChar *uri,
    bool *valid
) {
    struct params_write_attr *p = calloc(sizeof(struct params_write_attr), 1);
    p->params.free = params_write_attr_free;
    p->params.run = params_write_attr_run;
    p->prefix = prefix;
    p->uri = uri;
    *valid = true;
    return p;
}
// }}} write_attr

// {{{ write_dtd_entnty
struct params_write_dtd_entity {
    struct params params;
    bool is_pe;
    xmlChar *ndata_id;
    xmlChar *system_id;
    xmlChar *public_id;
};

int params_write_dtd_entity_run(struct ct_ctx *ctx) {
    struct params_write_dtd_entity *p = (struct params_write_dtd_entity *)ctx->params;

    // TODO: this doesn't work with single-quoted entity content.
    // should submit a bug to the libxml project
    return xmlTextWriterWriteDTDEntity(
        ctx->writer, p->is_pe, ctx->cmd->name,
        p->public_id, p->system_id, 
        p->ndata_id, ctx->cmd->content
    ) < 0;
}

void params_write_dtd_entity_free(void *f) {
    struct params_write_dtd_entity *p = f;
    if (p != NULL) {
        xmlFree(p->ndata_id);
        xmlFree(p->system_id);
        xmlFree(p->public_id);
    }
    free(p);
}

struct params_write_dtd_entity *params_write_dtd_entity_create(
    xmlChar *is_pe,
    xmlChar *ndata_id,
    xmlChar *system_id,
    xmlChar *public_id,
    bool *valid
) {
    *valid = true;

    struct params_write_dtd_entity *p = calloc(sizeof(struct params_write_dtd_entity), 1);
    p->params.free = params_write_dtd_entity_free;
    p->params.run = params_write_dtd_entity_run;

    if (is_pe != NULL) {
        int rc = bool_from_xml(is_pe, &p->is_pe);
        if (rc != 0) {
            *valid = false;
        }
        xmlFree(is_pe);
    }

    p->ndata_id = ndata_id;
    p->system_id = system_id;
    p->public_id = public_id;
    return p;
}
// }}} write_dtd_entity

// {{{ write_pi
struct params_write_pi {
    struct params params;
    xmlChar *target;
};

int params_write_pi_run(struct ct_ctx *ctx) {
    struct params_write_pi *p = (struct params_write_pi *)ctx->params;
    return xmlTextWriterWritePI(ctx->writer, p->target, ctx->cmd->content) < 0;
}

void params_write_pi_free(void *f) {
    struct params_write_pi *p = f;
    if (p != NULL) {
        xmlFree(p->target);
    }
    free(p);
}

struct params_write_pi *params_write_pi_create(
    xmlChar *target,
    bool *valid
) {
    *valid = true;
    struct params_write_pi *p = calloc(sizeof(struct params_write_pi), 1);
    p->params.free = params_write_pi_free;
    p->params.run = params_write_pi_run;
    p->target = target;
    return p;
}
// }}} write_pi

// {{{ write_cdata
struct params_write_cdata {
    struct params params;
};

void params_write_cdata_free(void *f) {
    struct params_write_cdata *p = f;
    free(p);
}

int params_write_cdata_run(struct ct_ctx *ctx) {
    return xmlTextWriterWriteCDATA(ctx->writer, ctx->cmd->content) < 0;
}

struct params_write_cdata *params_write_cdata_create(bool *valid) {
    *valid = true;
    struct params_write_cdata *p = calloc(sizeof(struct params_write_cdata), 1);
    p->params.free = params_write_cdata_free;
    p->params.run = params_write_cdata_run;
    return p;
}
// }}} write_cdata

// {{{ write_cdata_content
struct params_write_cdata_content {
    struct params params;
};

void params_write_cdata_content_free(void *f) {
    struct params_write_cdata_content *p = f;
    free(p);
}

int params_write_cdata_content_run(struct ct_ctx *ctx) {
    return xmlTextWriterWriteRaw(ctx->writer, ctx->cmd->content) < 0;
}

struct params_write_cdata_content *params_write_cdata_content_create(bool *valid) {
    *valid = true;
    struct params_write_cdata_content *p = calloc(sizeof(struct params_write_cdata_content), 1);
    p->params.free = params_write_cdata_content_free;
    p->params.run = params_write_cdata_content_run;
    return p;
}
// }}} write_cdata_content

// {{{ write_comment
struct params_write_comment {
    struct params params;
};

void params_write_comment_free(void *f) {
    struct params_write_comment *p = f;
    free(p);
}

int params_write_comment_run(struct ct_ctx *ctx) {
    if (ctx->in_dtd) {
        // HACK: libxml won't allow comments to be written using 
        // xmlTextWriterWriteComment inside a DTD. it always returns
        // false.
        if (xmlTextWriterWriteRaw(ctx->writer, (xmlChar*)"<!--") < 0) {
            return 1;
        }
        if (xmlTextWriterWriteRaw(ctx->writer, ctx->cmd->content) < 0) {
            return 1;
        }
        if (xmlTextWriterWriteRaw(ctx->writer, (xmlChar*)"-->") < 0) {
            return 1;
        }
        return 0;
    } else {
        return xmlTextWriterWriteComment(ctx->writer, ctx->cmd->content) < 0;
    }
}

struct params_write_comment *params_write_comment_create(bool *valid) {
    *valid = true;
    struct params_write_comment *p = calloc(sizeof(struct params_write_comment), 1);
    p->params.free = params_write_comment_free;
    p->params.run = params_write_comment_run;
    return p;
}
// }}} write_comment

// {{{ write_dtd_attr
struct params_write_dtd_attr {
    struct params params;
    xmlChar *type;
    xmlChar *decl;
    bool required;
};

void params_write_dtd_attr_free(void *f) {
    struct params_write_dtd_attr *p = f;
    if (p != NULL) {
        xmlFree(p->decl);
        xmlFree(p->type);
    }
    free(p);
}

int params_write_dtd_attr_run(struct ct_ctx *ctx) {
    struct params_write_dtd_attr *p = (struct params_write_dtd_attr *)ctx->params;
    int ret = 0;
    if ((ret = xmlTextWriterWriteRaw(ctx->writer, (xmlChar*)" ")) < 0) {
        return 1;
    }
    if ((ret = xmlTextWriterWriteRaw(ctx->writer, ctx->cmd->name)) < 0) {
        return 1;
    }
    if ((ret = xmlTextWriterWriteRaw(ctx->writer, (xmlChar*)" ")) < 0) {
		return 1;
	}
    if (p->type != NULL) {
        if ((ret = xmlTextWriterWriteRaw(ctx->writer, p->type)) < 0) {
            return 1;
        }
        if ((ret = xmlTextWriterWriteRaw(ctx->writer, (xmlChar*)" ")) < 0) {
            return 1;
        }
    }
    if (p->decl != NULL) {
        if (p->required) {
            if ((ret = xmlTextWriterWriteRaw(ctx->writer, (xmlChar*)"#FIXED \"")) < 0) {
                return 1;
            }
        } else {
            if ((ret = xmlTextWriterWriteRaw(ctx->writer, (xmlChar*)"\"")) < 0) {
                return 1;
            }
        }
        if ((ret = xmlTextWriterWriteString(ctx->writer, p->decl)) < 0) {
            return 1;
        }
		if ((ret = xmlTextWriterWriteRaw(ctx->writer, (xmlChar*)"\"")) < 0) {
			return 1;
		}
	} else {
		if (p->required) {
            if ((ret = xmlTextWriterWriteRaw(ctx->writer, (xmlChar*)"#REQUIRED")) < 0) {
                return 1;
            }
		} else {
            if ((ret = xmlTextWriterWriteRaw(ctx->writer, (xmlChar*)"#IMPLIED")) < 0) {
                return 1;
            }
        }
    }
    return 0;
}

struct params_write_dtd_attr *params_write_dtd_attr_create(
    xmlChar *type, xmlChar *decl, xmlChar *required, bool *valid
) {
    *valid = true;
    struct params_write_dtd_attr *p = calloc(sizeof(struct params_write_dtd_attr), 1);
    p->params.free = params_write_dtd_attr_free;
    p->params.run = params_write_dtd_attr_run;
    p->decl = decl;
    p->type = type;
    if (required != NULL) {
        int rc = bool_from_xml(required, &p->required);
        if (rc != 0) {
            *valid = false;
        }
        xmlFree(required);
    }
    return p;
}
// }}} write_dtd_attr

// {{{ write_dtd_elem
struct params_write_dtd_elem {
    struct params params;
};

void params_write_dtd_elem_free(void *f) {
    struct params_write_dtd_elem *p = f;
    free(p);
}

int params_write_dtd_elem_run(struct ct_ctx *ctx) {
    return xmlTextWriterWriteDTDElement(ctx->writer, ctx->cmd->name, ctx->cmd->content) < 0;
}

struct params_write_dtd_elem *params_write_dtd_elem_create(bool *valid) {
    *valid = true;
    struct params_write_dtd_elem *p = calloc(sizeof(struct params_write_dtd_elem), 1);
    p->params.free = params_write_dtd_elem_free;
    p->params.run = params_write_dtd_elem_run;
    return p;
}
// }}} write_dtd_elem

// {{{ write_raw
struct params_write_raw {
    struct params params;
};

void params_write_raw_free(void *f) {
    struct params_write_raw *p = f;
    free(p);
}

int params_write_raw_run(struct ct_ctx *ctx) {
    return xmlTextWriterWriteRaw(ctx->writer, ctx->cmd->content) < 0;
}

struct params_write_raw *params_write_raw_create(xmlChar *next, bool *valid) {
    *valid = true;
    struct params_write_raw *p = calloc(sizeof(struct params_write_raw), 1);

    // the ctester doesn't support next=false - next must always be true
    // because that's how libxml works under the hood. you can't write
    // raw text between an attribute and the end of an opening tag in libxml,
    // but you can in the go tester.
    if (next != NULL) {
        bool nextv = false;
        int rc = bool_from_xml(next, &nextv);
        if (rc != 0) {
            *valid = false;
        }
        if (nextv != true) {
            *valid = false;
        }
        xmlFree(next);
    }

    p->params.free = params_write_raw_free;
    p->params.run = params_write_raw_run;
    return p;
}
// }}} write_raw

// {{{ write_text
struct params_write_text {
    struct params params;
};

void params_write_text_free(void *f) {
    struct params_write_text *p = f;
    free(p);
}

int params_write_text_run(struct ct_ctx *ctx) {
    return xmlTextWriterWriteString(ctx->writer, ctx->cmd->content) < 0;
}

struct params_write_text *params_write_text_create(bool *valid) {
    *valid = true;
    struct params_write_text *p = calloc(sizeof(struct params_write_text), 1);
    p->params.free = params_write_text_free;
    p->params.run = params_write_text_run;
    return p;
}
// }}} write_text

// {{{ end_dtd
struct params_end_dtd {
    struct params params;
};

void params_end_dtd_free(void *f) {
    struct params_end_dtd *p = f;
    free(p);
}

int params_end_dtd_run(struct ct_ctx *ctx) {
    ctx->in_dtd = false;
    return xmlTextWriterEndDTD(ctx->writer) < 0;
}

struct params_end_dtd *params_end_dtd_create(bool *valid) {
    *valid = true;
    struct params_end_dtd *p = calloc(sizeof(struct params_end_dtd), 1);
    p->params.free = params_end_dtd_free;
    p->params.run = params_end_dtd_run;
    return p;
}
// }}} end_dtd

// {{{ start_dtd
struct params_start_dtd {
    struct params params;
    xmlChar *public_id;
    xmlChar *system_id;
};

int params_start_dtd_run(struct ct_ctx *ctx) {
    struct params_start_dtd *p = (struct params_start_dtd *)ctx->params;
    ctx->in_dtd = true;
    bool valid = true;
    ctx_end_push(ctx, (struct params *)params_end_dtd_create(&valid));
    return xmlTextWriterStartDTD(ctx->writer, ctx->cmd->name, p->public_id, p->system_id) < 0;
}

void params_start_dtd_free(void *f) {
    struct params_start_dtd *p = f;
    if (p != NULL) {
        xmlFree(p->public_id);
        xmlFree(p->system_id);
    }
    free(p);
}

struct params_start_dtd *params_start_dtd_create(
    xmlChar *public_id,
    xmlChar *system_id,
    bool *valid
) {
    *valid = true;
    struct params_start_dtd *p = calloc(sizeof(struct params_start_dtd), 1);
    p->params.free = params_start_dtd_free;
    p->params.run = params_start_dtd_run;
    p->public_id = public_id;
    p->system_id = system_id;
    return p;
}
// }}} end_dtd

// {{{ write_notation
struct params_write_notation {
    struct params params;
    xmlChar *public_id;
    xmlChar *system_id;
};

int params_write_notation_run(struct ct_ctx *ctx) {
    struct params_write_notation *p = (struct params_write_notation *)ctx->params;
    return xmlTextWriterWriteDTDNotation(ctx->writer, ctx->cmd->name, p->public_id, p->system_id) < 0;
}

void params_write_notation_free(void *f) {
    struct params_write_notation *p = f;
    if (p != NULL) {
        xmlFree(p->public_id);
        xmlFree(p->system_id);
    }
    free(p);
}

struct params_write_notation *params_write_notation_create(
    xmlChar *public_id,
    xmlChar *system_id,
    bool *valid
) {
    *valid = true;
    struct params_write_notation *p = calloc(sizeof(struct params_write_notation), 1);
    p->params.free = params_write_notation_free;
    p->params.run = params_write_notation_run;
    p->public_id = public_id;
    p->system_id = system_id;
    return p;
}
// }}} write_notation

// {{{ end_elem
struct params_end_elem {
    struct params params;
    bool full;
};

int params_end_elem_run(struct ct_ctx *ctx) {
    struct params_end_elem *p = (struct params_end_elem *)ctx->params;
    int rc = 0;
    if (p->full) {
        rc = xmlTextWriterFullEndElement(ctx->writer) < 0;
    } else {
        rc = xmlTextWriterEndElement(ctx->writer) < 0;
    }
    ctx_end_pop(ctx);
    return rc;
}

void params_end_elem_free(void *f) {
    struct params_end_elem *p = f;
    free(p);
}

struct params_end_elem *params_end_elem_create(
    xmlChar *full,
    bool *valid
) {
    *valid = true;
    struct params_end_elem *p = calloc(1, sizeof(struct params_end_elem));
    p->params.free = params_end_elem_free;
    p->params.run = params_end_elem_run;
    if (full != NULL) {
        int rc = bool_from_xml(full, &p->full);
        if (rc != 0) {
            *valid = false;
        }
        xmlFree(full);
    }
    return p;
}
// }}} end_elem

// {{{ start_elem
struct params_start_elem {
    struct params params;
    xmlChar *prefix;
    xmlChar *uri;
};

int params_start_elem_run(struct ct_ctx *ctx) {
    struct params_start_elem *p = (struct params_start_elem *)ctx->params;

    bool valid = true;
    ctx_end_push(ctx, (struct params *)params_end_elem_create(NULL, &valid));

    if (p->prefix != NULL || p->uri != NULL) {
        return xmlTextWriterStartElementNS(ctx->writer, p->prefix, ctx->cmd->name, p->uri) < 0;
    } else {
        return xmlTextWriterStartElement(ctx->writer, ctx->cmd->name) < 0;
    }
}

void params_start_elem_free(void *f) {
    struct params_start_elem *p = f;
    if (p != NULL) {
        xmlFree(p->prefix);
        xmlFree(p->uri);
    }
    free(f);
}

struct params_start_elem *params_start_elem_create(
    xmlChar *prefix,
    xmlChar *uri,
    bool *valid
) {
    *valid = true;
    struct params_start_elem *p = calloc(sizeof(struct params_start_elem), 1);
    p->params.free = params_start_elem_free;
    p->params.run = params_start_elem_run;
    p->prefix = prefix;
    p->uri = uri;
    return p;
}
// }}} start_elem

// {{{ end_cdata
struct params_end_cdata {
    struct params params;
};

void params_end_cdata_free(void *f) {
    struct params_end_cdata *p = f;
    free(p);
}

int params_end_cdata_run(struct ct_ctx *ctx) {
    int rc = xmlTextWriterEndCDATA(ctx->writer) < 0;
    ctx_end_pop(ctx);
    return rc;
}

struct params_end_cdata *params_end_cdata_create(bool *valid) {
    *valid = true;
    struct params_end_cdata *p = calloc(sizeof(struct params_end_cdata), 1);
    p->params.free = params_end_cdata_free;
    p->params.run = params_end_cdata_run;
    return p;
}
// }}} end_cdata

// {{{ start_cdata
struct params_start_cdata {
    struct params params;
};

void params_start_cdata_free(void *f) {
    struct params_start_cdata *p = f;
    free(p);
}

int params_start_cdata_run(struct ct_ctx *ctx) {
    bool valid = true;
    ctx_end_push(ctx, (struct params *)params_end_cdata_create(&valid));
    return xmlTextWriterStartCDATA(ctx->writer) < 0;
}

struct params_start_cdata *params_start_cdata_create(bool *valid) {
    *valid = true;
    struct params_start_cdata *p = calloc(sizeof(struct params_start_cdata), 1);
    p->params.free = params_start_cdata_free;
    p->params.run = params_start_cdata_run;
    return p;
}
// }}} start_cdata

// {{{ end_comment
struct params_end_comment {
    struct params params;
};

void params_end_comment_free(void *f) {
    struct params_end_comment *p = f;
    free(p);
}

int params_end_comment_run(struct ct_ctx *ctx) {
    int rc = xmlTextWriterEndComment(ctx->writer) < 0;
    ctx_end_pop(ctx);
    return rc;
}

struct params_end_comment *params_end_comment_create(bool *valid) {
    *valid = true;
    struct params_end_comment *p = calloc(sizeof(struct params_end_comment), 1);
    p->params.free = params_end_comment_free;
    p->params.run = params_end_comment_run;
    return p;
}
// }}} end_comment

// {{{ start_comment
struct params_start_comment {
    struct params params;
};

void params_start_comment_free(void *f) {
    struct params_start_comment *p = f;
    free(p);
}

int params_start_comment_run(struct ct_ctx *ctx) {
    bool valid = true;
    ctx_end_push(ctx, (struct params *)params_end_comment_create(&valid));
    return xmlTextWriterStartComment(ctx->writer) < 0;
}

struct params_start_comment *params_start_comment_create(bool *valid) {
    *valid = true;
    struct params_start_comment *p = calloc(sizeof(struct params_start_comment), 1);
    p->params.free = params_start_comment_free;
    p->params.run = params_start_comment_run;
    return p;
}
// }}} start_comment

// {{{ end_doc
struct params_end_doc {
    struct params params;
};

void params_end_doc_free(void *f) {
    struct params_end_doc *p = f;
    free(p);
}

int params_end_doc_run(struct ct_ctx *ctx) {
    int rc = xmlTextWriterEndDocument(ctx->writer) < 0;
    ctx_end_pop(ctx);
    return rc;
}

struct params_end_doc *params_end_doc_create(bool *valid) {
    *valid = true;
    struct params_end_doc *p = calloc(sizeof(struct params_end_doc), 1);
    p->params.free = params_end_doc_free;
    p->params.run = params_end_doc_run;
    return p;
}
// }}} end_doc

// {{{ start_doc
struct params_start_doc {
    struct params params;
    xmlChar *encoding;
    xmlChar *version;
    xmlChar *standalone;
};

void params_start_doc_free(void *f) {
    struct params_start_doc *p = f;
    xmlFree(p->encoding);
    xmlFree(p->version);
    xmlFree(p->standalone);
    free(p);
}

int params_start_doc_run(struct ct_ctx *ctx) {
    struct params_start_doc *p = (struct params_start_doc *)ctx->params;

    bool valid = true;
    ctx_end_push(ctx, (struct params *)params_end_doc_create(&valid));

    return xmlTextWriterStartDocument(
        ctx->writer, 
        (const char *)p->version,
        (const char *)p->encoding, 
        (const char *)p->standalone
    ) < 0;
}

struct params_start_doc *params_start_doc_create(xmlChar *encoding, xmlChar *version, xmlChar *standalone, bool *valid) {
    *valid = true;
    struct params_start_doc *p = calloc(sizeof(struct params_start_doc), 1);
    p->params.free = params_start_doc_free;
    p->params.run = params_start_doc_run;

    p->encoding = encoding;
    p->version = version;
    p->standalone = standalone;
    return p;
}
// }}} start_doc

// {{{ end_dtd_attlist
struct params_end_dtd_attlist {
    struct params params;
};

void params_end_dtd_attlist_free(void *f) {
    struct params_end_dtd_attlist *p = f;
    free(p);
}

int params_end_dtd_attlist_run(struct ct_ctx *ctx) {
    int rc = xmlTextWriterEndDTDAttlist(ctx->writer) < 0;
    ctx_end_pop(ctx);
    return rc;
}

struct params_end_dtd_attlist *params_end_dtd_attlist_create(bool *valid) {
    *valid = true;
    struct params_end_dtd_attlist *p = calloc(sizeof(struct params_end_dtd_attlist), 1);
    p->params.free = params_end_dtd_attlist_free;
    p->params.run = params_end_dtd_attlist_run;
    return p;
}
// }}} end_dtd_attlist 

// {{{ start_dtd_attlist
struct params_start_dtd_attlist {
    struct params params;
};

void params_start_dtd_attlist_free(void *f) {
    struct params_start_dtd_attlist *p = f;
    free(p);
}

int params_start_dtd_attlist_run(struct ct_ctx *ctx) {
    bool valid = true;
    ctx_end_push(ctx, (struct params *)params_end_dtd_attlist_create(&valid));

    return xmlTextWriterStartDTDAttlist(ctx->writer, ctx->cmd->name) < 0;
}

struct params_start_dtd_attlist *params_start_dtd_attlist_create(bool *valid) {
    *valid = true;
    struct params_start_dtd_attlist *p = calloc(sizeof(struct params_start_dtd_attlist), 1);
    p->params.free = params_start_dtd_attlist_free;
    p->params.run = params_start_dtd_attlist_run;
    return p;
}
// }}} start_dtd_attlist

// {{{ end_all
struct params_end_all {
    struct params params;
};

void params_end_all_free(void *f) {
    struct params_end_all *p = f;
    free(p);
}

int params_end_all_run(struct ct_ctx *ctx) {
    struct ctx_end *end = ctx->end;
    int rc = 0;

    struct params *oldparams = ctx->params;
    while (end) {
        // end->parent will get freed by end->params->run
        struct ctx_end *next = end->parent;
        ctx->params = end->params;

        if ((rc = end->params->run(ctx)) != 0) {
            return rc;
        }
        end = next;
    }
    ctx->params = oldparams;
    ctx_end(ctx);
    return 0;
}

struct params_end_all *params_end_all_create(bool *valid) {
    *valid = true;
    struct params_end_all *p = calloc(sizeof(struct params_end_all), 1);
    p->params.free = params_end_all_free;
    p->params.run = params_end_all_run;
    return p;
}
// }}} end_all

int script_init(struct script *script) {
    (void)script;
    return 0;
}

int script_command_add(struct script *script, struct command *command) {
    if (script->command == NULL) {
        script->command = command;
        script->tail = command;
    } else {
        script->tail->next = command;
        script->tail = command;
    }
    return 0;
}

struct command *command_create() {
    struct command *cmd = calloc(sizeof(struct command), 1);
    return cmd;
}

void command_free(struct command *cmd) {
    if (cmd->params != NULL) {
        cmd->params->free(cmd->params);
    }
    xmlFree(cmd->name);
    xmlFree(cmd->content);
    free(cmd);
}

void script_deinit(struct script *script) {
    struct command *current = script->command;

    if (current != NULL) {
        struct command *next = NULL;
        do {
            next = current->next;
            command_free(current);
            current = next;
        } while (current != NULL);
    }

    script->command = NULL;
    xmlFree(script->name);
}

enum ws_mode ws_mode_from_xml(const xmlChar *in) {
    if (xmlStrcmp(in, ws_mode_strip_str) == 0) { return ws_mode_strip; }
    return -1;
}

enum action action_from_xml(const xmlChar *in) {
    if (xmlStrcmp(in, action_start_str) == 0) { return action_start; }
    if (xmlStrcmp(in, action_end_str) == 0)   { return action_end; }
    if (xmlStrcmp(in, action_write_str) == 0) { return action_write; }
    return -1;
}

int command_parse(xmlNode *node, struct script *script) {
    int rc = 0;

    struct command *cmd = command_create();
    cmd->name = xmlGetProp(node, (xmlChar*)"name");

    xmlChar *action = xmlGetProp(node, (xmlChar*)"action");
    cmd->action = action_from_xml(action);

    xmlChar *kind = xmlGetProp(node, (xmlChar*)"kind");
    cmd->kind = kind_from_xml(kind);

    xmlChar *ws_mode = xmlGetProp(node, (xmlChar*)"ws");
    cmd->ws_mode = ws_mode_from_xml(ws_mode);

    cmd->content = xmlNodeGetContent(node);
    switch (cmd->ws_mode) {
    case ws_mode_none:; break;
    case ws_mode_strip:;
        xmlChar *new = ws_strip(cmd->content);
        xmlFree(cmd->content);
        cmd->content = new;
    break;
    }
    
    bool valid = false;

    switch (cmd->action) {
    case action_write:
        switch (cmd->kind) {
        case kind_attr:
            cmd->params = (struct params *)params_write_attr_create(
                xmlGetProp(node, (xmlChar*)"prefix"),
                xmlGetProp(node, (xmlChar*)"uri"),
                &valid
            );
        break;
        
        case kind_cdata:
            cmd->params = (struct params *)params_write_cdata_create(&valid); break;
        case kind_cdata_content:
            cmd->params = (struct params *)params_write_cdata_content_create(&valid); break;
        case kind_comment:
            cmd->params = (struct params *)params_write_comment_create(&valid); break;
        case kind_dtd_elem:
            cmd->params = (struct params *)params_write_dtd_elem_create(&valid); break;

        case kind_dtd_attr:
            cmd->params = (struct params *)params_write_dtd_attr_create(
                xmlGetProp(node, (xmlChar*)"type"),
                xmlGetProp(node, (xmlChar*)"decl"),
                xmlGetProp(node, (xmlChar*)"required"),
                &valid
            ); 
        break;

        case kind_dtd_entity:
            cmd->params = (struct params *)params_write_dtd_entity_create(
                xmlGetProp(node, (xmlChar*)"is-pe"),
                xmlGetProp(node, (xmlChar*)"ndata-id"),
                xmlGetProp(node, (xmlChar*)"system-id"),
                xmlGetProp(node, (xmlChar*)"public-id"),
                &valid
            );
        break;

        case kind_notation:
            cmd->params = (struct params *)params_write_notation_create(
                xmlGetProp(node, (xmlChar*)"public-id"),
                xmlGetProp(node, (xmlChar*)"system-id"),
                &valid
            );
        break;

        case kind_pi:
            cmd->params = (struct params *)params_write_pi_create(
                xmlGetProp(node, (xmlChar*)"target"),
                &valid
            );
        break;

        case kind_raw:
            cmd->params = (struct params *)params_write_raw_create(
                xmlGetProp(node, (xmlChar*)"next"),
                &valid
            );
        break;

        case kind_text:
            cmd->params = (struct params *)params_write_text_create(&valid); break;

        default:;
            valid = false;
        }
    break;

    case action_start:
        switch (cmd->kind) {
        case kind_cdata:
            cmd->params = (struct params *)params_start_cdata_create(&valid); break;
        case kind_comment:
            cmd->params = (struct params *)params_start_comment_create(&valid); break;

        case kind_doc:
            cmd->params = (struct params *)params_start_doc_create(
                xmlGetProp(node, (xmlChar*)"encoding"),
                xmlGetProp(node, (xmlChar*)"version"),
                xmlGetProp(node, (xmlChar*)"standalone"),
                &valid
            );
        break;

        case kind_dtd:
            cmd->params = (struct params *)params_start_dtd_create(
                xmlGetProp(node, (xmlChar*)"public-id"),
                xmlGetProp(node, (xmlChar*)"system-id"),
                &valid
            );
        break;

        case kind_dtd_attlist:
            cmd->params = (struct params *)params_start_dtd_attlist_create(&valid); break;

        case kind_elem:
            cmd->params = (struct params *)params_start_elem_create(
                xmlGetProp(node, (xmlChar*)"prefix"),
                xmlGetProp(node, (xmlChar*)"uri"),
                &valid
            );
        break;

        default:;
            valid = node->properties == NULL;
        }
    break;

    case action_end:
        switch (cmd->kind) {
        case kind_all:
            cmd->params = (struct params *)params_end_all_create(&valid); break;
        case kind_cdata:
            cmd->params = (struct params *)params_end_cdata_create(&valid); break;
        case kind_comment:
            cmd->params = (struct params *)params_end_comment_create(&valid); break;
        case kind_doc:
            cmd->params = (struct params *)params_end_doc_create(&valid); break;
        case kind_dtd:
            cmd->params = (struct params *)params_end_dtd_create(&valid); break;
        case kind_dtd_attlist:
            cmd->params = (struct params *)params_end_dtd_attlist_create(&valid); break;
        case kind_elem:
            cmd->params = (struct params *)params_end_elem_create(
                xmlGetProp(node, (xmlChar*)"full"),
                &valid
            );
        break;

        default:;
            valid = node->properties == NULL;
        }
    break;

    default:
        rc = 1;
    break;
    }

    if (!valid) {
        // TODO: reason!
        fprintf(stderr, "script validation failed\n");
        rc = 1;
        goto cleanup;
    }
    if ((rc = script_command_add(script, cmd)) != 0) {
        goto cleanup;
    }

cleanup:
    xmlFree(ws_mode);
    xmlFree(kind);
    xmlFree(action);
    return rc;
}

int script_parse(xmlDocPtr doc, struct script *script) {
    int rc = 0;

    xmlNode *root = xmlDocGetRootElement(doc);
    if (root == NULL) {
        rc = 1;
        goto cleanup;
    }

    /* print_element_names(root_element); */
    if (xmlStrcmp(root->name, (xmlChar *)"script") != 0) {
        rc = 1;
        goto cleanup;
    }

    bool ctester = true;
    xmlChar *ctesterProp = xmlGetProp(root, (xmlChar *)"ctester");
    if (ctesterProp != NULL) {
        rc = bool_from_xml(ctesterProp, &ctester);
        xmlFree(ctesterProp);
        if (rc != 0) {
            goto cleanup;
        }
    }

    if (!ctester) { 
        rc = 1; goto cleanup;
    }

    if ((rc = script_init(script)) != 0) {
        goto cleanup;
    }

    // this allocates, it gets freed in script_deinit
    script->name = xmlGetProp(root, (xmlChar *)"name");

    xmlNode *cur_node = root->children;

    for (; cur_node; cur_node = cur_node->next) {
        if (xmlStrcmp(cur_node->name, (xmlChar *)"command") == 0) {
            if ((rc = command_parse(cur_node, script)) != 0) {
                goto cleanup;
            }
        } else {
            if ((cur_node->type == XML_TEXT_NODE && xmlIsBlankNode(cur_node))
             || (cur_node->type == XML_COMMENT_NODE) 
            ) {
                // ignore
            } else {
                rc = 1; goto cleanup;
            }
        }
    }

cleanup:
    return rc;
}

int script_run(struct script *script, bool indent) {
    xmlTextWriterPtr writer = xmlNewTextWriterFilename("/dev/stdout", 0);
    if (indent) {
        xmlTextWriterSetIndent(writer, true);
    }
    int rc = 0;

    int index = 0;

    struct command *current = script->command;
    struct ct_ctx ctx = {
        .writer = writer,
        .end = NULL,
    };

    for (; current; current = current->next) {
        ctx.cmd = current;
        ctx.params = current->params;

        /* fprintf(stderr, "%d.%d\n", current->action, current->kind); */
        if (ctx.params == NULL) {
            fprintf(stderr, "Unknown command at index %d (%s.%s)", index, action_string(current->action), kind_string(current->kind));
            goto cleanup;
        }
        if ((rc = current->params->run(&ctx)) != 0) {
            xmlErrorPtr e = xmlGetLastError();
            fprintf(stderr, "Command at index %d (%s.%s) failed\n", index, action_string(current->action), kind_string(current->kind));
            if (e != NULL) {
                fprintf(stderr, "%s\n", e->message);
            }
            goto cleanup;
        }
        index++;
    }

    xmlTextWriterFlush(writer);

cleanup:;
    ctx_deinit(&ctx);
    xmlFreeTextWriter(writer);
    return rc;
}

int main(int argc, char *argv[]) {
    int rc = 0;
    xmlDocPtr doc = xmlReadFd(STDIN_FILENO, NULL, "UTF-8", 0);

    bool indent = false;
    for (int i = 1; i < argc; i++) {
        if (strcmp(argv[i], "--indent") == 0 || strcmp(argv[i], "-indent") == 0) {
            indent = true;
        }
    }

    struct script script = {};
    if ((rc = script_parse(doc, &script)) != 0) {
        goto cleanup;
    }
    if ((rc = script_run(&script, indent)) != 0) {
        goto cleanup;
    }

cleanup:
    script_deinit(&script);

    xmlFreeDoc(doc);
    xmlCleanupParser();

    return rc;
}
