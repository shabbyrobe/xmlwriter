#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <string.h>

// this little beast is because it was faster to write this than
// to find a cross-platform portable way to grep for bytes in
// the 0-127 range.

int main(void) {
    #define READ_SIZE 8192
    unsigned char buffer[READ_SIZE];
    bool done = false;

    while (!done) {
        size_t len = fread(&buffer, 1, READ_SIZE, stdin);
        if (len < READ_SIZE) {
            done = true;
        }
        for (size_t i = 0; i < len; i++) {
            if (buffer[i] > 127) {
                return 1;
            }
        }
    }
    return 0;
}
