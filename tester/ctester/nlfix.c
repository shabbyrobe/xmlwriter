#include <stdio.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>

// removes dependency on dos2unix, this is a little bit faster.
// probably full of bugs though.

#ifndef READ_SIZE
#define READ_SIZE 8192
#endif

int main(void) {
    char in[READ_SIZE] = {0};

    // if the last byte of the previous read was an \r and no bytes of the
    // current read are an \r, you will have an extra byte in the write.
    char out[READ_SIZE+1] = {0};

    size_t len = 0;
    size_t read = 0;
    size_t widx = 0;

    bool in_cr = false;
    do {
        len = fread(&in, 1, READ_SIZE, stdin);
        widx = 0;
        
        for (size_t a = 0; a < len; a++) {
            if (!in_cr) {
                if (in[a] == '\r') {
                    in_cr = true;
                } else {
                    out[widx++] = in[a];
                }
            } else {
                if (in[a] == '\r') {
                    out[widx++] = '\n';
                } else {
                    in_cr = false;
                    if (in[a] != '\n') {
                        out[widx++] = '\n';
                    }
                    out[widx++] = in[a];
                }
            }
        }

        read += len;
        fwrite(&out, 1, widx, stdout);
    }
    while (len == READ_SIZE);

    if (in_cr) {
        printf("\n");
    }
    return 0;
}
