package xmlwriter_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

func TestNoDeps(t *testing.T) {
	if os.Getenv("XMLWRITER_SKIP_MOD") != "" {
		// Use this to avoid this check if you need to use spew.Dump in tests:
		t.Skip()
	}

	fix, err := ioutil.ReadFile("go.mod.fix")
	if err != nil {
		panic(err)
	}
	bts, err := ioutil.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fixNL(fix), fixNL(bts)) {
		t.Fatal("go.mod contains unexpected content:\n" + string(bts))
	}
}

func fixNL(d []byte) []byte {
	d = bytes.Replace(d, []byte{13, 10}, []byte{10}, -1)
	d = bytes.Replace(d, []byte{13}, []byte{10}, -1)
	return d
}
