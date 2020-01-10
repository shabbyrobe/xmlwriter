package xmlwriter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shabbyrobe/xmlwriter/testtool"
)

func TestCheckName(t *testing.T) {
	for idx, tc := range []struct {
		name string
		yep  bool
	}{
		{"", true},
		{"a", true},
		{"-", false},
		{"a-", true},
		{"a-a", true},
		{"the-quick-brown-fox-jumped-over-the-lazy-dog", true},
		{":", true},
		{"!", false},
		{"\u00df", true},
	} {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			err := CheckName(tc.name)
			if tc.yep {
				testtool.OK(t, err)
			} else {
				testtool.Assert(t, err != nil)
			}
		})
	}
}

var BenchErr error

func BenchmarkCheckName(b *testing.B) {
	for _, sz := range []int{10, 50} {
		b.Run(fmt.Sprintf("ascii/%d", sz), func(b *testing.B) {
			v := strings.Repeat("a", sz)
			for i := 0; i < b.N; i++ {
				BenchErr = CheckName(v)
			}
		})

		b.Run(fmt.Sprintf("worst-case/%d", sz), func(b *testing.B) {
			v := "\u0370" + strings.Repeat("\u203F", sz-1)
			for i := 0; i < b.N; i++ {
				BenchErr = CheckName(v)
			}
		})
	}
}
