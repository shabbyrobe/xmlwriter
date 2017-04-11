#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <string.h>
#include <ctype.h>
#include <assert.h>

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

