package tester

import (
	"bytes"
	"testing"

	xw "github.com/shabbyrobe/xmlwriter"

	tt "github.com/shabbyrobe/xmlwriter/testtool"
	"golang.org/x/text/encoding/charmap"
)

func TestEncodingWindows1252(t *testing.T) {
	b := &bytes.Buffer{}
	enc := charmap.Windows1252.NewEncoder()
	w := xw.OpenEncoding(b, "windows-1252", enc)
	xw.Must(w.Start(xw.Doc{}))
	xw.Must(w.Start(xw.Elem{Name: "hello"}))
	xw.Must(w.Write(xw.Text("Résumé")))
	w.EndAllFlush()
	out := b.Bytes()

	// byte representation of expected windows-1252 encoded text
	check := []byte{'R', 0xE9, 's', 'u', 'm', 0xE9}
	tt.Assert(t, bytes.Contains(out, check))
}
