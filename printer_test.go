package xmlwriter

import (
	"testing"
	"unicode/utf8"
)

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
