#ifndef __XML_C
#define __XML_C

#include <expat.h>
#include <stdio.h>
#include <string.h>
#include <stdbool.h>

char *expat_errors[] = {
  [XML_ERROR_NONE] = "XML_ERROR_NONE",
  [XML_ERROR_NO_MEMORY] = "XML_ERROR_NO_MEMORY",
  [XML_ERROR_SYNTAX] = "XML_ERROR_SYNTAX",
  [XML_ERROR_NO_ELEMENTS] = "XML_ERROR_NO_ELEMENTS",
  [XML_ERROR_INVALID_TOKEN] = "XML_ERROR_INVALID_TOKEN",
  [XML_ERROR_UNCLOSED_TOKEN] = "XML_ERROR_UNCLOSED_TOKEN",
  [XML_ERROR_PARTIAL_CHAR] = "XML_ERROR_PARTIAL_CHAR",
  [XML_ERROR_TAG_MISMATCH] = "XML_ERROR_TAG_MISMATCH",
  [XML_ERROR_DUPLICATE_ATTRIBUTE] = "XML_ERROR_DUPLICATE_ATTRIBUTE",
  [XML_ERROR_JUNK_AFTER_DOC_ELEMENT] = "XML_ERROR_JUNK_AFTER_DOC_ELEMENT",
  [XML_ERROR_PARAM_ENTITY_REF] = "XML_ERROR_PARAM_ENTITY_REF",
  [XML_ERROR_UNDEFINED_ENTITY] = "XML_ERROR_UNDEFINED_ENTITY",
  [XML_ERROR_RECURSIVE_ENTITY_REF] = "XML_ERROR_RECURSIVE_ENTITY_REF",
  [XML_ERROR_ASYNC_ENTITY] = "XML_ERROR_ASYNC_ENTITY",
  [XML_ERROR_BAD_CHAR_REF] = "XML_ERROR_BAD_CHAR_REF",
  [XML_ERROR_BINARY_ENTITY_REF] = "XML_ERROR_BINARY_ENTITY_REF",
  [XML_ERROR_ATTRIBUTE_EXTERNAL_ENTITY_REF] = "XML_ERROR_ATTRIBUTE_EXTERNAL_ENTITY_REF",
  [XML_ERROR_MISPLACED_XML_PI] = "XML_ERROR_MISPLACED_XML_PI",
  [XML_ERROR_UNKNOWN_ENCODING] = "XML_ERROR_UNKNOWN_ENCODING",
  [XML_ERROR_INCORRECT_ENCODING] = "XML_ERROR_INCORRECT_ENCODING",
  [XML_ERROR_UNCLOSED_CDATA_SECTION] = "XML_ERROR_UNCLOSED_CDATA_SECTION",
  [XML_ERROR_EXTERNAL_ENTITY_HANDLING] = "XML_ERROR_EXTERNAL_ENTITY_HANDLING",
  [XML_ERROR_NOT_STANDALONE] = "XML_ERROR_NOT_STANDALONE",
  [XML_ERROR_UNEXPECTED_STATE] = "XML_ERROR_UNEXPECTED_STATE",
  [XML_ERROR_ENTITY_DECLARED_IN_PE] = "XML_ERROR_ENTITY_DECLARED_IN_PE",
  [XML_ERROR_FEATURE_REQUIRES_XML_DTD] = "XML_ERROR_FEATURE_REQUIRES_XML_DTD",
  [XML_ERROR_CANT_CHANGE_FEATURE_ONCE_PARSING] = "XML_ERROR_CANT_CHANGE_FEATURE_ONCE_PARSING",
  [XML_ERROR_UNBOUND_PREFIX] = "XML_ERROR_UNBOUND_PREFIX",
  [XML_ERROR_UNDECLARING_PREFIX] = "XML_ERROR_UNDECLARING_PREFIX",
  [XML_ERROR_INCOMPLETE_PE] = "XML_ERROR_INCOMPLETE_PE",
  [XML_ERROR_XML_DECL] = "XML_ERROR_XML_DECL",
  [XML_ERROR_TEXT_DECL] = "XML_ERROR_TEXT_DECL",
  [XML_ERROR_PUBLICID] = "XML_ERROR_PUBLICID",
  [XML_ERROR_SUSPENDED] = "XML_ERROR_SUSPENDED",
  [XML_ERROR_NOT_SUSPENDED] = "XML_ERROR_NOT_SUSPENDED",
  [XML_ERROR_ABORTED] = "XML_ERROR_ABORTED",
  [XML_ERROR_FINISHED] = "XML_ERROR_FINISHED",
  [XML_ERROR_SUSPEND_PE] = "XML_ERROR_SUSPEND_PE",
  [XML_ERROR_RESERVED_PREFIX_XML] = "XML_ERROR_RESERVED_PREFIX_XML",
  [XML_ERROR_RESERVED_PREFIX_XMLNS] = "XML_ERROR_RESERVED_PREFIX_XMLNS",
  [XML_ERROR_RESERVED_NAMESPACE_URI] = "XML_ERROR_RESERVED_NAMESPACE_URI",
};

#define CRAP_START 0
#define CRAP_NAME 1
#define CRAP_QUOTE 2
#define CRAP_VAL 3

// dear lord we need our own attribute parser now?
// turns out we don't, i just didn't see the XMLDeclHandler call,
// the header file is a bit hard to read in places
int crap_atts_parse(const char *in, int len, char ***atts, int *read) {
    char qchar = 0;

    int state = CRAP_START;
    int nstart = 0, nlen = 0;
    int vstart = 0, vlen = 0;
    int rc = 0;

    // cheap and nasty way to preallocate needed memory
    int attmax = 0;
    for (int i = 0; i < len; i++) {
        if (in[i] == '=') {
            attmax++;
        }
    }
    *atts = calloc(sizeof(char *), (attmax*2) + 1);
    int attcur = 0;

    int i = 0;
    for (i = 0; i < len; i++) {
    repeat:
        switch (state) {
        case CRAP_START:
            if (in[i] == '?') goto done;
            if (in[i] == ' ') goto next;
            nstart = i; nlen = 0;
            state = CRAP_NAME;
            goto repeat;
        break;

        case CRAP_NAME:
            if (in[i] == ' ') { state = CRAP_START; goto repeat; }
            if (in[i] == '=') { state = CRAP_QUOTE; goto next; }
            if (in[i] == '?') { rc = 1; goto done; }
            nlen ++;
        break;

        case CRAP_QUOTE:
            if (in[i] == '"' || in[i] == '\'') {
                qchar = in[i];
                state = CRAP_VAL;
                vstart = i + 1; vlen = 0;
                goto next;
            } else {
                rc = 1;
                goto done;
            }
        break;

        case CRAP_VAL:
            if (in[i] == qchar) {
                (*atts)[attcur++] = strndup(&in[nstart], nlen);
                (*atts)[attcur++] = strndup(&in[vstart], vlen);
                state = CRAP_START;
                goto next;
            }
            vlen ++;
        break;
        }
    next:;
    }
done:;
    if (read != NULL) {
        *read = i;
    }
    if (state != CRAP_START) {
        rc = 1;
    }
    return rc;
}

void crap_atts_free(char **atts) {
    for (int i = 0; atts[i]; i += 1) {
        free(atts[i]);
    }
    free(atts);
}

char *raw_encoding_extract(char *buffer, size_t len) {
    char *ret = NULL;

    if (len > 6 && strncmp(buffer, "<?xml ", 6) == 0) {
        char **atts = NULL;
        char *bufptr = buffer + 5;
        int buflen = len;
        buflen -= 5;

        int att_read = 0;
        int rc = crap_atts_parse(bufptr, buflen, &atts, &att_read);
        if (rc != 0) {
            goto done;
        }

        for (int i = 0; atts[i]; i+=2) {
            if (strcmp(atts[i], "encoding")==0) {
                ret = strdup(atts[i+1]);
                goto done;
            }
        }
        crap_atts_free(atts);
    }

done:
    return ret;
}

#endif
