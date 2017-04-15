CFLAGS=-O2 -g -std=c11 -Wall -Wextra -Wno-missing-field-initializers

binaries = ctester testbuilder encfixer encextractor indexer nlfix nohigh normaliser
testers = nlfix-test

all: $(binaries)

ctester: CFLAGS+=`pkg-config --cflags libxml-2.0`
ctester: LDLIBS+=`pkg-config --libs   libxml-2.0`

indexer: CFLAGS+=`pkg-config --cflags expat sqlite3`
indexer: LDLIBS+=`pkg-config --libs   expat sqlite3` -lmagic

testbuilder: CFLAGS+=`pkg-config --cflags libxml-2.0 expat`
testbuilder: LDLIBS+=`pkg-config --libs   libxml-2.0 expat`

nlfix-test: CFLAGS+=-DREAD_SIZE=16
nlfix-test: nlfix.c
	$(CC) $(CPPFLAGS) $(CFLAGS) -o $@ nlfix.c

normaliser: CFLAGS+=`pkg-config --cflags libxml-2.0 expat`
normaliser: LDLIBS+=`pkg-config --libs   libxml-2.0 expat`

clean:
	rm -f $(binaries) $(testers)

