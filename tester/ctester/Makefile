CFLAGS=-O2 -g -std=c11 -Wall -Wextra

all: attrsorter ctester testbuilder encfixer encextractor nohigh nlfix

ctester: CFLAGS+=`pkg-config --cflags libxml-2.0`
ctester: LDLIBS+=`pkg-config --libs libxml-2.0`

testbuilder: CFLAGS+=`pkg-config --cflags libxml-2.0 expat`
testbuilder: LDLIBS+=`pkg-config --libs libxml-2.0 expat`

attrsorter: CFLAGS+=`pkg-config --cflags libxml-2.0 expat`
attrsorter: LDLIBS+=`pkg-config --libs libxml-2.0 expat`

nlfix-test: CFLAGS+=-DREAD_SIZE=16
nlfix-test: nlfix.c
	$(CC) $(CPPFLAGS) $(CFLAGS) -o $@ nlfix.c
	
