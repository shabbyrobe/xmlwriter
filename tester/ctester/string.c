#ifndef __STRING_C
#define __STRING_C

#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <assert.h>
#include <stdarg.h>
#include <ctype.h>

char *strndup(const char *s, size_t n) {
    char* new = malloc(n+1);
    if (new) {
        strncpy(new, s, n);
        new[n] = '\0';
    }
    return new;
}

char* strtoupper(char* s) {
    assert(s != NULL);
    char* p = s;
    while (*p != '\0') {
        *p = toupper(*p);
        p++;
    }
    return s;
}

struct buf {
    char *bytes;
    size_t cap;
    size_t len;
};

void buf_deinit(struct buf *ctx) {
    free(ctx->bytes);
}

void buf_strnappend(struct buf *ctx, const char *ch, size_t len) {
    size_t pos = ctx->len;
    if (ctx->bytes == NULL || (ctx->len + len + 1) >= ctx->cap) {
        ctx->cap += ctx->len + len + 1;
        size_t v = ctx->cap;
        v--; v |= v >> 1; v |= v >> 2; v |= v >> 4; v |= v >> 8; v |= v >> 16; v++;
        ctx->cap = v;
        ctx->bytes = realloc(ctx->bytes, ctx->cap);
    }
    ctx->len += len;
    memcpy(ctx->bytes + pos, ch, len);
    ctx->bytes[ctx->len] = 0;
}

void buf_strappend(struct buf *ctx, const char *ch) {
    buf_strnappend(ctx, ch, strlen(ch));
}

// From https://github.com/littlstar/asprintf.c/blob/master/LICENSE
// MIT license
// Copyright (c) 2014 Little Star Media, Inc.
int vasprintf (char **str, const char *fmt, va_list args) {
  int size = 0;
  va_list tmpa;

  // copy
  va_copy(tmpa, args);

  // apply variadic arguments to
  // sprintf with format to get size
  size = vsnprintf(NULL, size, fmt, tmpa);

  // toss args
  va_end(tmpa);

  // return -1 to be compliant if
  // size is less than 0
  if (size < 0) { return -1; }

  // alloc with size plus 1 for `\0'
  *str = (char *) malloc(size + 1);

  // return -1 to be compliant
  // if pointer is `NULL'
  if (NULL == *str) { return -1; }

  // format string with original
  // variadic arguments and set new size
  size = vsprintf(*str, fmt, args);
  return size;
}

int asprintf (char **str, const char *fmt, ...) {
  int size = 0;
  va_list args;

  // init variadic argumens
  va_start(args, fmt);

  // format and get size
  size = vasprintf(str, fmt, args);

  // toss args
  va_end(args);

  return size;
}


#endif
