package tester

// These tests are in the tester package rather than the xmlwriter
// package to try to avoid issues with tools like dep vendoring more
// stuff than it should (https://github.com/golang/dep/issues/120).

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	xw "github.com/shabbyrobe/xmlwriter"

	tt "github.com/shabbyrobe/xmlwriter/testtool"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

func TestEncodingWindows1252(t *testing.T) {
	b := &bytes.Buffer{}
	enc := charmap.Windows1252.NewEncoder()
	w := xw.OpenEncoding(b, "windows-1252", enc)
	xw.Must(w.Start(xw.Doc{}))
	xw.Must(w.Start(xw.Elem{Name: "hello"}))
	xw.Must(w.Write(xw.Text("Résumé")))
	xw.Must(w.Write(xw.Text("😀")))
	xw.Must(w.EndAllFlush())
	out := b.Bytes()

	// byte representation of expected windows-1252 encoded text -
	// attempting to decode as string yields panic
	check := []byte{'R', 0xE9, 's', 'u', 'm', 0xE9, '&', '#', '1', '2', '8', '5', '1', '2', ';'}
	tt.Assert(t, bytes.Contains(out, check))
}

func TestEncodeRunesInISO88591(t *testing.T) {
	b := &bytes.Buffer{}
	enc := charmap.ISO8859_1.NewEncoder()
	w := xw.OpenEncoding(b, "ISO-8859-1", enc)
	xw.Must(w.Start(xw.Doc{}))
	xw.Must(w.Start(xw.Elem{Name: "hello"}))
	xw.Must(w.Write(xw.Text("😀")))
	xw.Must(w.EndAllFlush())
	out := b.String()

	check := "<hello>&#128512;</hello>"
	tt.Assert(t, strings.Contains(out, check))
}

func TestAssumptionsAboutHTMLEscaper(t *testing.T) {
	encoder := charmap.ISO8859_1.NewEncoder()

	for i := 0; i < 16384; i++ {
		b := &bytes.Buffer{}
		writer := encoding.HTMLEscapeUnsupported(encoder).Writer(b)
		dst := make([]byte, 32)
		r := rune(i)
		l := utf8.EncodeRune(dst, r)
		writer.Write(dst[:l])
		if i < 256 {
			tt.Equals(t, string([]byte{byte(i)}), b.String())
		} else {
			tt.Equals(t, fmt.Sprintf("&#%d;", i), b.String())
		}
	}
}
