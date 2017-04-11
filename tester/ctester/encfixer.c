#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <string.h>
#include <ctype.h>
#include <assert.h>

#include "crapatts.c"

// libxml writes the encoding in all caps regardless of 
// the input. if we override the encoding in the go version
// it might not be uppercased.
// 
// that crap attribute parser i wrote when i thought i had
// to handle the <?xml declaration by hand really came in handy
// in the end!!

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

char* strtoupper(char* s) {
    assert(s != NULL);
    char* p = s;
    while (*p != '\0') {
        *p = toupper(*p);
        p++;
    }
    return s;
}

int main(void) {
    #define READ_SIZE 8192
    char buffer[READ_SIZE];
    bool done = false;
    size_t read = 0;

    size_t len = fread(&buffer, sizeof(char), READ_SIZE, stdin);
    if (len < READ_SIZE) {
        done = true;
    }
    read += len;

    if (len > 5 && strncmp(buffer, "<?xml", 5) == 0) {
        char **atts = NULL;
        char *bufptr = buffer + 5;
        int buflen = len;
        buflen -= 5;
        fwrite("<?xml", 1, 5, stdout);

        int att_read = 0;
        int rc = crap_atts_parse(bufptr, buflen, &atts, &att_read);
        if (rc != 0) {
            fprintf(stderr, "Crap atts parser failed\n");
            return rc;
        }

        for (int i = 0; atts[i]; i+=2) {
            if (strcmp(atts[i], "version")==0) {
                printf(" %s=\"%s\"", atts[i], atts[i+1]);
            } else if (strcmp(atts[i], "encoding")==0) {
                printf(" %s=\"%s\"", atts[i], strtoupper(atts[i+1]));
            } else if (strcmp(atts[i], "standalone")==0) {
                printf(" %s=\"%s\"", atts[i], atts[i+1]);
            } else {
                fprintf(stderr, "Unparseable attribute %s\n", atts[i]);
                return 1;
            }
        }

        crap_atts_free(atts);
        bufptr += att_read;
        buflen -= att_read;
        fwrite(bufptr, 1, buflen, stdout);

    } else {
        goto passthrough;
    }

    while (!done) {
        len = fread(&buffer, 1, READ_SIZE, stdin);
        if (len < READ_SIZE) {
            done = true;
        }
        read += len;
    passthrough:
        fwrite(&buffer, 1, len, stdout);
    }
}

