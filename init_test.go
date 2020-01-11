package xmlwriter

import (
	"bytes"
	"io"
	"io/ioutil"
	"runtime"
)

var memstats runtime.MemStats

func allocs() uint64 {
	runtime.ReadMemStats(&memstats)
	return memstats.Mallocs
}

type DodgyWriter struct {
	writer     io.Writer
	shouldFail func(b []byte) (fail bool, len int, err error)
}

func (d *DodgyWriter) Write(b []byte) (len int, err error) {
	if fail, len, err := d.shouldFail(b); fail {
		return len, err
	}
	return d.writer.Write(b)
}

func open(o ...Option) (*bytes.Buffer, *Writer) {
	b := &bytes.Buffer{}
	w := Open(b, o...)
	return b, w
}

func openNull(o ...Option) *Writer {
	return Open(ioutil.Discard, o...)
}

func str(b *bytes.Buffer, w *Writer) string {
	must(w.Flush())
	return b.String()
}

func doWrite(node ...Writable) string {
	b, w := open()
	for _, n := range node {
		must(w.Write(n))
	}
	return str(b, w)
}

func doStart(node ...Startable) string {
	b, w := open()
	for _, n := range node {
		must(w.Start(n))
	}
	return str(b, w)
}

func doBlock(start Startable, children ...Writable) string {
	b, w := open()
	must(w.Block(start, children...))
	return str(b, w)
}

func doWriteErrMsg(nodes ...Writable) (ret string) {
	defer func() {
		if e := recover(); e != nil {
			ret = e.(error).Error()
		}
	}()
	doWrite(nodes...)
	return ""
}

func doStartErrMsg(nodes ...Startable) (ret string) {
	defer func() {
		if e := recover(); e != nil {
			ret = e.(error).Error()
		}
	}()
	doStart(nodes...)
	return ""
}

func doBlockErrMsg(start Startable, children ...Writable) (ret string) {
	defer func() {
		if e := recover(); e != nil {
			ret = e.(error).Error()
		}
	}()
	doBlock(start, children...)
	return ""
}
