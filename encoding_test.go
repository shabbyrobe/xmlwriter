package xmlwriter

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	tt "github.com/shabbyrobe/xmlwriter/testtool"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

func TestEncodingWindows1252(t *testing.T) {
	b := &bytes.Buffer{}
	enc := charmap.Windows1252.NewEncoder()
	w := OpenEncoding(b, "windows-1252", enc)
	tt.OK(t, w.Start(Doc{}))
	tt.OK(t, w.Start(Elem{Name: "hello"}))
	tt.OK(t, w.Write(Text("Résumé")))
	tt.OK(t, w.Write(Text("😀")))
	tt.OK(t, w.EndAllFlush())
	out := b.Bytes()

	// byte representation of expected windows-1252 encoded text -
	// attempting to decode as string yields panic
	check := []byte{'R', 0xE9, 's', 'u', 'm', 0xE9, '&', '#', '1', '2', '8', '5', '1', '2', ';'}
	tt.Assert(t, bytes.Contains(out, check))
}

func TestEncodingUTF16BE(t *testing.T) {
	b := &bytes.Buffer{}
	enc := unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM).NewEncoder()
	w := OpenEncoding(b, "utf-16be", enc)
	tt.OK(t, w.Start(Doc{}))
	tt.OK(t, w.Start(Elem{Name: "hello"}))
	tt.OK(t, w.Write(Text("Résumé")))
	tt.OK(t, w.Write(Text("😀")))
	tt.OK(t, w.EndAllFlush())
	out := b.Bytes()

	tt.Assert(t, bytes.HasPrefix(out, []byte{0xFE, 0xFF}))
	tt.Assert(t, bytes.Contains(out, []byte{0xD8, 0x3D, 0xDE, 0x00}))
	tt.Assert(t, bytes.Contains(out, []byte{0x00, 0x3C, 0x00, 0x68, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F}))
}

func TestEncodeRunesInISO88591(t *testing.T) {
	b := &bytes.Buffer{}
	enc := charmap.ISO8859_1.NewEncoder()
	w := OpenEncoding(b, "ISO-8859-1", enc)
	tt.OK(t, w.Start(Doc{}))
	tt.OK(t, w.Start(Elem{Name: "hello"}))
	tt.OK(t, w.Write(Text("😀")))
	tt.OK(t, w.EndAllFlush())
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
