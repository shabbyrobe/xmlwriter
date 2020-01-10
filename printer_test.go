package xmlwriter

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"unicode/utf8"

	tt "github.com/shabbyrobe/xmlwriter/testtool"
)

func TestEscapeAttrString(t *testing.T) {
	ws := func(t *testing.T, in string) (out string) {
		t.Helper()
		var b bytes.Buffer
		p := printer{Writer: bufio.NewWriterSize(&b, 2048)}
		p.EscapeAttrString(in)
		p.Flush()
		if err := p.cachedWriteError(); err != nil {
			t.Fatal(err)
		}
		return b.String()
	}

	expect := func(t *testing.T, in string, out string) {
		tt.Assert(t, ws(t, in) == out, fmt.Sprint(in, "!=", out))
	}

	expect(t, "abc", "abc")
	expect(t, "a-c", "a-c")
	expect(t, "a\nb", "a&#xA;b")
	expect(t, "\nb", "&#xA;b")
	expect(t, "a\n", "a&#xA;")
}

func TestIsInCharacterRange(t *testing.T) {
	invalid := []rune{
		utf8.MaxRune + 1,
		0xD800, // surrogate min
		0xDFFF, // surrogate max
		-1,
	}
	for _, r := range invalid {
		if isInCharacterRange(r) {
			t.Errorf("rune %U considered valid", r)
		}
	}
}

func BenchmarkEscapeAttrString(b *testing.B) {
	for _, sz := range []int{10, 50, 300} {
		b.Run(fmt.Sprintf("ascii-%d", sz), func(b *testing.B) {
			p := printer{Writer: bufio.NewWriterSize(ioutil.Discard, 2048)}
			v := strings.Repeat("1", sz)

			for i := 0; i < b.N; i++ {
				p.EscapeAttrString(v)
				p.Reset(ioutil.Discard)
			}
		})

		b.Run(fmt.Sprintf("utf8-first-%d", sz), func(b *testing.B) {
			p := printer{Writer: bufio.NewWriterSize(ioutil.Discard, 2048)}
			v := "\uD000" + strings.Repeat("1", sz-1)

			for i := 0; i < b.N; i++ {
				p.EscapeAttrString(v)
				p.Reset(ioutil.Discard)
			}
		})

		b.Run(fmt.Sprintf("utf8-last-%d", sz), func(b *testing.B) {
			p := printer{Writer: bufio.NewWriterSize(ioutil.Discard, 2048)}
			v := strings.Repeat("1", sz-1) + "\uD000"

			for i := 0; i < b.N; i++ {
				p.EscapeAttrString(v)
				p.Reset(ioutil.Discard)
			}
		})
	}
}
