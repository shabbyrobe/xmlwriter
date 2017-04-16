CFLAGS=-O2 -g -std=c11 -Wall -Wextra -Wno-missing-field-initializers \
	-D_POSIX_C_SOURCE=200809L

# posix is for getline and strdup. get rid of them maybe?

binaries = ctester testbuilder encfixer encextractor indexer nohigh normaliser

all: $(binaries)

ctester: CFLAGS+=`pkg-config --cflags libxml-2.0`
ctester: LDLIBS+=`pkg-config --libs   libxml-2.0`

indexer: CFLAGS+=`pkg-config --cflags expat sqlite3`
indexer: LDLIBS+=`pkg-config --libs   expat sqlite3` -lmagic

testbuilder: CFLAGS+=`pkg-config --cflags libxml-2.0 expat`
testbuilder: LDLIBS+=`pkg-config --libs   libxml-2.0 expat`

normaliser: CFLAGS+=`pkg-config --cflags libxml-2.0 expat`
normaliser: LDLIBS+=`pkg-config --libs   libxml-2.0 expat`

clean:
	rm -f $(binaries)

