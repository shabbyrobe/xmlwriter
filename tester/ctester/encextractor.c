#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <string.h>
#include <ctype.h>
#include <assert.h>

#include "crapatts.c"

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
    #define READ_SIZE 4096
    char buffer[READ_SIZE];

    size_t len = fread(&buffer, sizeof(char), READ_SIZE, stdin);
    if (len > 5 && strncmp(buffer, "<?xml", 5) == 0) {
        char **atts = NULL;
        char *bufptr = buffer + 5;
        int buflen = len;
        buflen -= 5;

        int att_read = 0;
        int rc = crap_atts_parse(bufptr, buflen, &atts, &att_read);
        if (rc != 0) {
            fprintf(stderr, "Crap atts parser failed\n");
            return rc;
        }

        for (int i = 0; atts[i]; i+=2) {
            if (strcmp(atts[i], "encoding")==0) {
                printf("%s\n", strtoupper(atts[i+1]));
            }
        }
    }
    return 0;
}

