#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <string.h>
#include <ctype.h>
#include <assert.h>

#include "string.c"
#include "xml.c"

int main(void) {
    #define READ_SIZE 4096
    char buffer[READ_SIZE];

    size_t len = fread(&buffer, sizeof(char), READ_SIZE, stdin);
    char *enc = raw_encoding_extract(buffer, len);
    if (enc != NULL) {
        printf("%s\n", strtoupper(enc));
        free(enc);
    }
    return 0;
}
