package xmlwriter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"testing"

	tt "github.com/shabbyrobe/xmlwriter/testtool"
)

var memstats runtime.MemStats

func allocs() uint64 {
	runtime.ReadMemStats(&memstats)
	return memstats.Mallocs
}

type Null struct{}

func (w Null) Write(p []byte) (n int, err error) {
	return len(p), nil
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
	return Open(Null{}, o...)
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

func TestDoc(t *testing.T) {
	b, w := open()
	must(w.Start(Doc{}))
	tt.Equals(t, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n", str(b, w))
	must(w.EndDoc())
	tt.Equals(t, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n", str(b, w))
}

func TestElemSingle(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()

	ec.Must(w.Start(Elem{Name: "yep"}))
	tt.Equals(t, "<yep", str(b, w))

	ec.Must(w.EndElem())
	tt.Equals(t, "<yep/>", str(b, w))
}

func TestElemEndNamed(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()

	ec.Must(w.Start(Elem{Name: "yep"}))
	tt.Equals(t, "<yep", str(b, w))

	ec.Must(w.End(ElemNode, "yep"))
	tt.Equals(t, "<yep/>", str(b, w))
}

func TestElemPushMultiple(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()

	ec.Must(w.Start(Elem{Name: "yep"}))
	ec.Must(w.Start(Elem{Name: "yep"}))
	ec.Must(w.EndElemFull())
	ec.Must(w.EndElemFull())
	tt.Equals(t, "<yep><yep></yep></yep>", str(b, w))
}

func TestElemPushMultiplePrefixed(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Start(Elem{Name: "yep", Prefix: "woo"}))
	ec.Must(w.Start(Elem{Name: "yep", Prefix: "woo"}))
	ec.Must(w.EndElemFull())
	ec.Must(w.EndElemFull())
	tt.Equals(t, "<woo:yep><woo:yep></woo:yep></woo:yep>", str(b, w))
}

func TestElemSingleAttrOption(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Start(Elem{Name: "yep", Attrs: []Attr{
		{Name: "one", Value: "1"},
		{Name: "two", Value: "2"},
	}}))
	tt.Equals(t, "<yep one=\"1\" two=\"2\"", str(b, w))
	ec.Must(w.EndElem())
	tt.Equals(t, "<yep one=\"1\" two=\"2\"/>", str(b, w))
}

func TestElemSingleFullOption(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Start(Elem{Name: "yep", Full: true}))
	tt.Equals(t, "<yep", str(b, w))
	ec.Must(w.EndElem())
	tt.Equals(t, "<yep></yep>", str(b, w))
}

func TestElemSingleAttrOptionFullOption(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Start(Elem{Name: "yep", Full: true, Attrs: []Attr{
		{Name: "one", Value: "1"},
		{Name: "two", Value: "2"},
	}}))
	tt.Equals(t, "<yep one=\"1\" two=\"2\"", str(b, w))
	ec.Must(w.EndElem())
	tt.Equals(t, "<yep one=\"1\" two=\"2\"></yep>", str(b, w))
}

func TestElemSingleFullMethod(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Start(Elem{Name: "yep", Full: false}))
	tt.Equals(t, "<yep", str(b, w))
	ec.Must(w.EndElemFull())
	tt.Equals(t, "<yep></yep>", str(b, w))
}

func TestElemAttrDuplicatesElemPrefix(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Start(Elem{Name: "yep", Prefix: "ns1", URI: "http://uri"}))
	ec.Must(w.Write(Attr{Name: "yep", Value: "bar", Prefix: "ns1", URI: "http://uri"}))
	ec.Must(w.EndElemFull())
	tt.Equals(t, `<ns1:yep xmlns:ns1="http://uri" ns1:yep="bar"></ns1:yep>`, str(b, w))
}

func TestElemWriteTree(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Write(Elem{
		Name:  "foo",
		Attrs: []Attr{{Name: "a", Value: "b"}},
		Content: []Writable{
			Elem{Name: "bar"},
			Elem{Name: "baz", Content: []Writable{
				Elem{Name: "qux"},
			}},
		},
	}))
	tt.Equals(t, `<foo a="b"><bar/><baz><qux/></baz></foo>`, str(b, w))
}

func TestElemStartTree(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Start(Elem{Name: "foo", Content: []Writable{
		Elem{Name: "bar"},
		Elem{Name: "baz", Content: []Writable{
			Elem{Name: "qux"},
		}},
	}}))
	ec.Must(w.Write(Elem{Name: "pants"}))
	ec.Must(w.EndAll())

	tt.Equals(t, "<foo><bar/><baz><qux/></baz><pants/></foo>", str(b, w))
}

func TestWriteAttrBare(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Write(Attr{Name: "foo"}))
	ec.Must(w.Write(Attr{Name: "bar"}))
	tt.Equals(t, ` foo="" bar=""`, str(b, w))
}

func TestWriteAttrInts(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Start(Elem{Name: "foo"}))
	ec.Must(w.Write(
		Attr{Name: "int"}.Int(10),
		Attr{Name: "intneg"}.Int(-10),
		Attr{Name: "int8"}.Int8(20),
		Attr{Name: "int16"}.Int16(30),
		Attr{Name: "int32"}.Int32(40),
		Attr{Name: "int64"}.Int64(50),
		Attr{Name: "uint"}.Uint(10),
		Attr{Name: "uint8"}.Uint8(20),
		Attr{Name: "uint16"}.Uint16(30),
		Attr{Name: "uint32"}.Uint32(40),
		Attr{Name: "uint64"}.Uint64(50),
	))
	ec.Must(w.EndAll())

	tt.Equals(t, `<foo int="10" intneg="-10" int8="20" int16="30" int32="40" int64="50"`+
		` uint="10" uint8="20" uint16="30" uint32="40" uint64="50"/>`, str(b, w))
}

func TestWriteAttrBool(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Start(Elem{Name: "foo"}))
	ec.Must(w.Write(
		Attr{Name: "yep"}.Bool(true),
		Attr{Name: "nup"}.Bool(false),
	))
	ec.Must(w.EndAll())

	tt.Equals(t, `<foo yep="true" nup="false"/>`, str(b, w))
}

func TestWriteAttrFloats(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Start(Elem{Name: "foo"}))
	ec.Must(w.Write(
		Attr{Name: "float32"}.Float32(123.45),
		Attr{Name: "float64"}.Float64(234.56),
	))
	ec.Must(w.EndAll())

	tt.Equals(t, `<foo float32="123.45" float64="234.56"/>`, str(b, w))
}

func TestWriteAttrPrefix(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Write(Attr{Name: "foo", Prefix: "yep"}))
	ec.Must(w.Write(Attr{Name: "bar", Prefix: "nup"}))
	tt.Equals(t, ` yep:foo="" nup:bar=""`, str(b, w))
}

func TestWriteElemAttrPrefix(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Start(Elem{Name: "elem"}))
	ec.Must(w.Write(Attr{Name: "foo", Prefix: "yep"}))
	ec.Must(w.Write(Attr{Name: "bar", Prefix: "nup"}))
	ec.Must(w.EndElem())
	tt.Equals(t, `<elem yep:foo="" nup:bar=""/>`, str(b, w))
}

func TestWriteElemAttrPrefixURI(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Start(Elem{Name: "elem"}))
	ec.Must(w.Write(Attr{Name: "foo", Prefix: "yep", URI: "http://esta"}))
	ec.Must(w.Write(Attr{Name: "bar", Prefix: "nup", URI: "http://otre"}))
	ec.Must(w.EndElem())
	tt.Equals(t, `<elem yep:foo="" nup:bar="" xmlns:yep="http://esta" xmlns:nup="http://otre"/>`, str(b, w))
}

func TestWriteElemAttrDuplicatePrefixDifferentURI(t *testing.T) {
	ec := &ErrCollector{}

	// FIXME: this should not succeed
	_, w := open()
	ec.Must(w.Start(Elem{Name: "elem"}))
	ec.Must(w.Write(Attr{Name: "foo", Prefix: "yep", URI: "http://esta"}))

	err := w.Write(Attr{Name: "bar", Prefix: "yep", URI: "http://otre"})
	tt.Assert(t, err.Error() == "uri already exists for ns prefix yep")
}

func TestWriteBadAttr(t *testing.T) {
	ec := &ErrCollector{}

	w := openNull()
	ec.Must(w.Start(Doc{}))
	tt.Equals(t,
		fmt.Errorf("xmlwriter: unexpected kind document, expected none, elem"),
		w.Write(Attr{Name: "yep"}))

	w = openNull()
	ec.Must(w.Start(DTD{Name: "dtd"}))
	tt.Equals(t,
		fmt.Errorf("xmlwriter: unexpected kind dtd, expected none, elem"),
		w.Write(Attr{Name: "yep"}))
}

func TestNest(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	expected := &bytes.Buffer{}
	for i := 0; i < 1000; i++ {
		ec.Must(w.Start(Elem{Name: "hi", Full: true}))
		expected.WriteString("<hi>")
	}
	for i := 0; i < 1000; i++ {
		expected.WriteString("</hi>")
	}
	ec.Must(w.EndAll())
	tt.Equals(t, expected.String(), str(b, w))
}

func TestNotation(t *testing.T) {
	tt.Equals(t, `<!NOTATION pants PUBLIC "pub">`,
		doWrite(Notation{Name: "pants", PublicID: "pub"}))
	tt.Equals(t, `<!NOTATION pants PUBLIC "pub" "sys">`,
		doWrite(Notation{Name: "pants", SystemID: "sys", PublicID: "pub"}))

	tt.Pattern(t, `(?i)requires external ID`,
		doWriteErrMsg(Notation{Name: "hi"}))
	tt.Pattern(t, `(?i)name must not be empty`,
		doWriteErrMsg(Notation{Name: "", SystemID: "sys", PublicID: "pub"}))
	tt.Pattern(t, `(?i)invalid name`,
		doWriteErrMsg(Notation{Name: "@$*(!$", SystemID: "sys", PublicID: "pub"}))
}

func TestPI(t *testing.T) {
	tt.Equals(t, `<?xml-stylesheet href="my-style.css"?>`,
		doWrite(PI{Target: "xml-stylesheet", Content: `href="my-style.css"`}))

	tt.Pattern(t, `(?i)may not be 'xml'`,
		doWriteErrMsg(PI{Target: "XML", Content: "yep"}))
	tt.Pattern(t, `(?i)invalid name`,
		doWriteErrMsg(PI{Target: "@$*(!$", Content: "yep"}))
	tt.Pattern(t, `(?i)may not contain '\?>'`,
		doWriteErrMsg(PI{Target: "name", Content: "?>"}))
}

func TestEndToDepth(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Start(Elem{Name: "foo"}))

	d := w.Depth()
	ec.Must(w.Start(Elem{Name: "bar"}))
	ec.Must(w.Start(Elem{Name: "baz"}))
	ec.Must(w.Start(Elem{Name: "qux"}))
	ec.Must(w.EndToDepth(d, ElemNode, "bar"))
	tt.Equals(t, `<foo><bar><baz><qux/></baz></bar>`, str(b, w))
}

func TestEndAny(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Start(Elem{Name: "foo"}))
	ec.Must(w.Start(Elem{Name: "bar"}))
	ec.Must(w.Start(Elem{Name: "baz"}))
	ec.Must(w.EndAny())
	ec.Must(w.EndAny())
	ec.Must(w.EndAny())
	e := w.EndAny()
	tt.Pattern(t, `could not pop node`, e.Error())
	tt.Equals(t, `<foo><bar><baz/></bar></foo>`, str(b, w))
}

func TestEndNamed(t *testing.T) {
	ec := &ErrCollector{}
	b, w := open()
	ec.Must(w.Start(DTD{Name: "foo"}))
	ec.Must(w.End(DTDNode, "foo"))
	tt.Equals(t, `<!DOCTYPE foo>`, str(b, w))

	_, w = open()
	ec.Must(w.Start(DTD{Name: "foo"}))
	tt.Pattern(t, `dtd name 'foo' did not match expected 'bar'`,
		w.End(DTDNode, "bar").Error())

	_, w = open()
	ec.Must(w.Start(DTDAttList{Name: "foo"}))
	tt.Pattern(t, `dtdattlist name 'foo' did not match expected 'bar'`,
		w.End(DTDAttListNode, "bar").Error())

	// FIXME: This could be better as an error.
	_, w = open()
	ec.Must(w.Start(Elem{Prefix: "yep", Name: "foo"}))
	ec.Must(w.End(ElemNode, "foo"))

	_, w = open()
	ec.Must(w.Start(Elem{Prefix: "yep", Name: "foo"}))
	ec.Must(w.End(ElemNode, "yep", "foo"))

	_, w = open()
	ec.Must(w.Start(Elem{Prefix: "yep", Name: "foo"}))
	tt.Pattern(t, `elem name 'yep:foo' did not match expected 'nup:foo'`,
		w.End(ElemNode, "nup", "foo").Error())

	_, w = open()
	ec.Must(w.Start(Elem{Prefix: "yep", Name: "foo"}))
	tt.Pattern(t, `elem name 'yep:foo' did not match expected 'yep:bar'`,
		w.End(ElemNode, "yep", "bar").Error())

	_, w = open()
	ec.Must(w.Start(Comment{}))
	tt.Pattern(t, `node was not named`, w.End(CommentNode, "foo").Error())

	_, w = open()
	ec.Must(w.Start(Comment{}))
	tt.Pattern(t, `node was not named`, w.End(CommentNode, "foo", "bar").Error())
}

func TestEndWrong(t *testing.T) {
	ec := &ErrCollector{}
	_, w := open()
	ec.Must(w.Start(Elem{Name: "foo"}))
	tt.Pattern(t, `unexpected kind elem, expected dtd`, w.End(DTDNode).Error())
}

func TestReadableAPI(t *testing.T) {
	b, w := open()
	ec := &ErrCollector{}
	defer ec.Panic()
	ec.Do(
		w.Start(Doc{}),
		w.Start(Elem{
			Name: "foo", Attrs: []Attr{
				{Name: "a1", Value: "val1"},
				{Name: "a2", Value: "val2"},
			},
			Content: []Writable{
				Comment{"hello"},
				Elem{
					Name: "bar", Attrs: []Attr{
						{Name: "a1", Value: "val1"},
						{Name: "a2", Value: "val2"},
					},
					Content: []Writable{
						Elem{Name: "baz"},
					},
				},
			},
		}),
		w.Start(Elem{Name: "bar"}),
		w.EndAllFlush(),
	)
	tt.Equals(t, `<?xml version="1.0" encoding="UTF-8"?>`+"\n"+
		`<foo a1="val1" a2="val2"><!--hello--><bar a1="val1" a2="val2"><baz/></bar><bar/></foo>`, b.String())
}

func TestLast(t *testing.T) {
	startLast := func(s Startable) Event {
		_, w := open(WithIndent())
		(&ErrCollector{}).Must(w.Start(s))
		return w.last
	}
	writeLast := func(n Writable) Event {
		_, w := open(WithIndent())
		(&ErrCollector{}).Must(w.Write(n))
		return w.last
	}
	tt.Equals(t, Event{StateOpen, ElemNode, 0}, startLast(Elem{Name: "foo"}))
	tt.Equals(t, Event{StateEnded, ElemNode, 0}, writeLast(Elem{Name: "foo"}))
	tt.Equals(t, Event{StateEnded, AttrNode, 0}, writeLast(Attr{Name: "foo", Value: "bar"}))
	tt.Equals(t, Event{StateEnded, AttrNode, 0}, startLast(Elem{Name: "foo", Attrs: []Attr{{Name: "foo", Value: "bar"}}}))
	tt.Equals(t, Event{StateOpen, DocNode, 0}, startLast(Doc{}))
	tt.Equals(t, Event{StateOpen, DTDNode, 0}, startLast(DTD{Name: "pants"}))

	tt.Equals(t, Event{StateOpen, CDataNode, 0}, startLast(CData{}))
	tt.Equals(t, Event{StateOpen, CommentNode, 0}, startLast(Comment{}))
	tt.Equals(t, Event{StateOpen, DTDNode, 0}, startLast(DTD{Name: "yep"}))
	tt.Equals(t, Event{StateOpen, DTDAttListNode, 0}, startLast(DTDAttList{Name: "yep"}))
	tt.Equals(t, Event{StateOpen, DocNode, 0}, startLast(Doc{}))

	tt.Equals(t, Event{StateEnded, NotationNode, 0}, writeLast(Notation{Name: "yep", SystemID: "sys"}))
	tt.Equals(t, Event{StateEnded, PINode, 0}, writeLast(PI{}))
	tt.Equals(t, Event{StateEnded, RawNode, 0}, writeLast(Raw("foo")))
	tt.Equals(t, Event{StateEnded, TextNode, 0}, writeLast(Text("foo")))
	tt.Equals(t, Event{StateEnded, CDataContentNode, 0}, writeLast(CDataContent("foo")))
	tt.Equals(t, Event{StateEnded, CommentContentNode, 0}, writeLast(CommentContent("foo")))
	tt.Equals(t, Event{StateEnded, DTDAttrNode, 0}, writeLast(DTDAttr{Name: "yep", Type: "CDATA"}))
	tt.Equals(t, Event{StateEnded, DTDElemNode, 0}, writeLast(DTDElem{Name: "yep", Decl: "yep"}))
	tt.Equals(t, Event{StateEnded, DTDEntityNode, 0}, writeLast(DTDEntity{Name: "yep", Content: "yep"}))
}

func TestInvalidParent(t *testing.T) {
	start := true
	write := false
	for _, c := range []struct {
		startNode bool
		node      Node
		parent    Startable
	}{
		{start, Elem{Name: "elem"}, DTD{Name: "dtd"}},
		{write, Attr{Name: "attr"}, DTD{Name: "dtd"}},
		{write, Attr{Name: "attr"}, Comment{}},
	} {
		testInvalidParent(t, c.startNode, c.node, c.parent)
	}
}

func testInvalidParent(t *testing.T, startNode bool, node Node, parent Startable) {
	_, w := open()
	(&ErrCollector{}).Must(w.Start(parent))
	var err error
	if startNode {
		err = w.Start(node.(Startable))
	} else {
		err = w.Write(node.(Writable))
	}
	tt.Assert(t, err != nil)
	tt.Assert(t, strings.HasPrefix(err.Error(), "xmlwriter: unexpected kind "), err.Error())
}

// allows us to make sure that we can collect errors emitted by
// the underlying writer using the same cachedWriterError pattern
// used by the stdlib (initial versions of this lib were an unreadable
// mess of "if err := ...; err != nil" checks)
func TestDodgyWriterAttribute(t *testing.T) {
	last := ""
	for j := 0; j <= 6; j++ {
		b := &bytes.Buffer{}
		i := 0
		e := errors.New("failed")
		d := &DodgyWriter{
			writer: b,
			shouldFail: func(_ []byte) (fail bool, len int, err error) {
				if i >= j {
					return true, 0, e
				}
				i++
				return
			},
		}
		w := Open(d, func(w *Writer) {
			w.InitialBufSize = 1
		})
		err := w.WriteAttr(Attr{Name: "hi", Value: "yep"})
		tt.Equals(t, e, err)

		contents := b.String()
		if j >= 1 {
			tt.Assert(t, len(contents) > len(last))
		}
		last = contents
	}
}

func TestStartDoc(t *testing.T) {
	tt.Equals(t, `<?xml version="1.0" encoding="UTF-8"?>`+"\n", doStart(Doc{}))

	tt.Equals(t, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`+"\n",
		doStart(Doc{}.WithStandalone(true)))

	tt.Equals(t, `<?xml version="1.0" encoding="UTF-8" standalone="no"?>`+"\n",
		doStart(Doc{}.WithStandalone(false)))

	tt.Equals(t, `<?xml version="1.0" encoding="pants"?>`+"\n", doStart(Doc{}.ForceEncoding("pants")))
	tt.Equals(t, `<?xml version="pants" encoding="UTF-8"?>`+"\n", doStart(Doc{}.ForceVersion("pants")))
	tt.Equals(t, `<?xml version="1.0"?>`+"\n", doStart(Doc{SuppressEncoding: true}))
	tt.Equals(t, `<?xml encoding="UTF-8"?>`+"\n", doStart(Doc{SuppressVersion: true}))
	tt.Equals(t, `<?xml ?>`+"\n", doStart(Doc{SuppressVersion: true, SuppressEncoding: true}))
}

func TestCData(t *testing.T) {
	tt.Equals(t, `<![CDATA[]]>`, doWrite(CData{}))
	tt.Equals(t, `<![CDATA[`, doStart(CData{}))
	tt.Equals(t, `<![CDATA[foo]]>`, doBlock(CData{}, CDataContent("foo")))
	tt.Equals(t, `<![CDATA[yepfoo]]>`, doBlock(CData{"yep"}, CDataContent("foo")))
	tt.Equals(t, `<![CDATA[yepfoobar]]>`, doBlock(CData{"yep"}, CDataContent("foo"), CDataContent("bar")))
	tt.Equals(t, `<![CDATA[&"']]>`, doBlock(CData{`&"'`}))
	tt.Equals(t, `<![CDATA[&"']]>`, doBlock(CData{}, CDataContent(`&"'`)))
	tt.Equals(t, "<![CDATA[\nyep\n]]>", doWrite(CData{"\nyep\n"}))

	tt.Pattern(t, `may not contain ']]>'`, doWriteErrMsg(CData{"]]>"}))
}

func TestComment(t *testing.T) {
	tt.Equals(t, `<!---->`, doWrite(Comment{}))
	tt.Equals(t, `<!--`, doStart(Comment{}))
	tt.Equals(t, `<!--foo-->`, doBlock(Comment{}, CommentContent("foo")))
	tt.Equals(t, `<!--yepfoo-->`, doBlock(Comment{"yep"}, CommentContent("foo")))
	tt.Equals(t, `<!--yepfoobar-->`, doBlock(Comment{"yep"}, CommentContent("foo"), CommentContent("bar")))
	tt.Equals(t, `<!--&"'-->`, doBlock(Comment{`&"'`}))
	tt.Equals(t, `<!--&"'-->`, doBlock(Comment{}, CommentContent(`&"'`)))
	tt.Equals(t, "<!--\nyep\n-->", doWrite(Comment{"\nyep\n"}))

	tt.Pattern(t, `may not contain '--'`, doWriteErrMsg(Comment{"--"}))
}

func TestText(t *testing.T) {
	tt.Equals(t, `&amp;`, doWrite(Text("&")))
	tt.Equals(t, `hello`, doWrite(Text("hello")))
	tt.Equals(t, "\n", doWrite(Text("\n")))
}

func TestRaw(t *testing.T) {
	tt.Equals(t, `&amp;`, doWrite(Raw("&amp;")))
	tt.Equals(t, `hello`, doWrite(Raw("hello")))
	tt.Equals(t, "\n", doWrite(Raw("\n")))
}

func TestDTD(t *testing.T) {
	tt.Equals(t, `<!DOCTYPE hi>`, doBlock(DTD{Name: "hi"}))
	tt.Equals(t, `<!DOCTYPE hi SYSTEM "sys">`, doBlock(DTD{Name: "hi", SystemID: "sys"}))
	tt.Equals(t, `<!DOCTYPE hi PUBLIC "pub" "sys">`,
		doBlock(DTD{Name: "hi", PublicID: "pub", SystemID: "sys"}))
	tt.Pattern(t, `public ID provided but system ID missing`,
		doBlockErrMsg(DTD{Name: "hi", PublicID: "pub"}))
}

func TestDTDEntity(t *testing.T) {
	tt.Equals(t, `<!ENTITY hi "">`, doWrite(DTDEntity{Name: "hi"}))
	tt.Equals(t, `<!ENTITY hi "yep">`, doWrite(DTDEntity{Name: "hi", Content: "yep"}))
	tt.Equals(t, `<!ENTITY hi "it's">`, doWrite(DTDEntity{Name: "hi", Content: "it's"}))
	tt.Equals(t, `<!ENTITY hi 'it"s'>`, doWrite(DTDEntity{Name: "hi", Content: `it"s`}))

	tt.Equals(t, `<!ENTITY hi "&#20;&#20;">`, doWrite(DTDEntity{Name: "hi", Content: "&#20;&#20;"}))

	tt.Equals(t, `<!ENTITY % hi "yep">`,
		doWrite(DTDEntity{Name: "hi", Content: `yep`, IsPE: true}))

	tt.Equals(t, `<!ENTITY hi SYSTEM "sys">`,
		doWrite(DTDEntity{Name: "hi", SystemID: "sys"}))
	tt.Equals(t, `<!ENTITY % hi SYSTEM "sys">`,
		doWrite(DTDEntity{Name: "hi", SystemID: "sys", IsPE: true}))
	tt.Equals(t, `<!ENTITY hi PUBLIC "pub" "sys">`,
		doWrite(DTDEntity{Name: "hi", SystemID: "sys", PublicID: "pub"}))
	tt.Equals(t, `<!ENTITY hi PUBLIC "pub" "sys" NDATA nd>`,
		doWrite(DTDEntity{Name: "hi", SystemID: "sys", PublicID: "pub", NDataID: "nd"}))

	tt.Pattern(t, `(?i)name must not be empty`,
		doWriteErrMsg(DTDEntity{Name: ""}))
	tt.Pattern(t, `(?i)invalid name`,
		doWriteErrMsg(DTDEntity{Name: "!@&*#^!"}))
	tt.Pattern(t, `(?i)must only contain double or single quotes`,
		doWriteErrMsg(DTDEntity{Name: "hi", Content: `'"`}))
	tt.Pattern(t, `(?i)external ID required for NDataID`,
		doWriteErrMsg(DTDEntity{Name: "hi", NDataID: "nd"}))
	tt.Pattern(t, `(?i)external ID required for NDataID`,
		doWriteErrMsg(DTDEntity{Name: "hi", Content: "ding", NDataID: "nd"}))
	tt.Pattern(t, `(?i)public ID provided but system ID missing`,
		doWriteErrMsg(DTDEntity{Name: "hi", PublicID: "pub"}))
	tt.Pattern(t, `(?i)invalid name`,
		doWriteErrMsg(DTDEntity{Name: "hi", SystemID: "sys", NDataID: "(*@$&"}))
	tt.Pattern(t, `(?i)IsPE and NDataID both provided`,
		doWriteErrMsg(DTDEntity{Name: "hi", SystemID: "pub", NDataID: "nd", IsPE: true}))
	tt.Pattern(t, `(?i)external ID and content cannot both`,
		doWriteErrMsg(DTDEntity{Name: "hi", PublicID: "pub", Content: "yep"}))
	tt.Pattern(t, `(?i)external ID and content cannot both`,
		doWriteErrMsg(DTDEntity{Name: "hi", SystemID: "pub", Content: "yep"}))
	tt.Pattern(t, `(?i)external ID and content cannot both`,
		doWriteErrMsg(DTDEntity{Name: "hi", SystemID: "pub", Content: "yep"}))
}

func TestDTDAttList(t *testing.T) {
	tt.Equals(t, `<!ATTLIST yep>`, doWrite(DTDAttList{Name: "yep"}))
	tt.Equals(t, `<!ATTLIST yep`, doStart(DTDAttList{Name: "yep"}))

	tt.Equals(t, `<!ATTLIST yep a1 CDATA #IMPLIED>`,
		doWrite(DTDAttList{Name: "yep", Attrs: []DTDAttr{{Name: "a1", Type: "CDATA"}}}))

	tt.Equals(t, `<!ATTLIST yep a1 CDATA #IMPLIED a2 CDATA #IMPLIED>`,
		doWrite(DTDAttList{Name: "yep", Attrs: []DTDAttr{
			{Name: "a1", Type: DTDAttrString},
			{Name: "a2", Type: DTDAttrString},
		}}))

	tt.Equals(t, `<!ATTLIST yep a1 CDATA #REQUIRED a2 CDATA #REQUIRED>`,
		doWrite(DTDAttList{Name: "yep", Attrs: []DTDAttr{
			{Name: "a1", Type: DTDAttrString, Required: true},
			{Name: "a2", Type: DTDAttrString, Required: true},
		}}))
	tt.Equals(t, `<!ATTLIST yep a1 CDATA #FIXED "foo" a2 CDATA #FIXED "bar">`,
		doWrite(DTDAttList{Name: "yep", Attrs: []DTDAttr{
			{Name: "a1", Type: DTDAttrString, Decl: "foo", Required: true},
			{Name: "a2", Type: DTDAttrString, Decl: "bar", Required: true},
		}}))
	tt.Equals(t, `<!ATTLIST yep a1 CDATA "foo" a2 CDATA "bar">`,
		doWrite(DTDAttList{Name: "yep", Attrs: []DTDAttr{
			{Name: "a1", Type: DTDAttrString, Decl: "foo"},
			{Name: "a2", Type: DTDAttrString, Decl: "bar"},
		}}))

	tt.Equals(t, `<!ATTLIST yep a1 CDATA "foo" a2 CDATA "bar">`,
		doBlock(DTDAttList{Name: "yep"}, DTDAttr{Name: "a1", Type: DTDAttrString, Decl: "foo"},
			DTDAttr{Name: "a2", Type: DTDAttrString, Decl: "bar"}))

	tt.Pattern(t, `(?i)name must not be empty`, doWriteErrMsg(DTDAttList{}))
	tt.Pattern(t, `(?i)invalid name`, doWriteErrMsg(DTDAttList{Name: "1"}))
	tt.Pattern(t, `(?i)invalid name`, doWriteErrMsg(DTDAttList{Name: "$@"}))
}

func TestDTDElem(t *testing.T) {
	tt.Equals(t, `<!ELEMENT elem EMPTY>`,
		doWrite(DTDElem{Name: "elem", Decl: DTDElemEmpty}))
	tt.Equals(t, `<!ELEMENT elem (#PCDATA|child)*>`,
		doWrite(DTDElem{Name: "elem", Decl: "(#PCDATA|child)*"}))

	// FIXME: couldn't quite grok this from the spec yet, but it is
	// an example pasted verbatim of a valid element decl.
	// tt.Equals(t, `<!ELEMENT %name.para; %content.para;>`,
	//     doWrite(DTDElem{Name: `%name.para;`, Decl: `%content.para;`}))

	tt.Pattern(t, `(?i)name must not be empty`, doWriteErrMsg(DTDElem{}))
	tt.Pattern(t, `(?i)decl must not be empty`, doWriteErrMsg(DTDElem{Name: "elem"}))
	tt.Pattern(t, `(?i)invalid name`, doWriteErrMsg(DTDElem{Name: "@$Q$", Decl: DTDElemEmpty}))
}

func TestDTDAttr(t *testing.T) {
	tt.Equals(t, ` foo CDATA #IMPLIED`, doWrite(DTDAttr{Name: "foo", Type: DTDAttrString}))
	tt.Equals(t, ` foo CDATA #IMPLIED bar CDATA #IMPLIED`, doWrite(
		DTDAttr{Name: "foo", Type: "CDATA"},
		DTDAttr{Name: "bar", Type: "CDATA"}))

	tt.Pattern(t, `(?i)name must not be empty`, doWriteErrMsg(DTDAttr{}))
	tt.Pattern(t, `(?i)type must not be empty`, doWriteErrMsg(DTDAttr{Name: "1"}))
	tt.Pattern(t, `(?i)invalid name`, doWriteErrMsg(DTDAttr{Name: "1", Type: "CDATA"}))
	tt.Pattern(t, `(?i)invalid name`, doWriteErrMsg(DTDAttr{Name: "$@", Type: "CDATA"}))
}

func TestDTDWithChildren(t *testing.T) {
	tt.Equals(t, `<!DOCTYPE hi [<!ENTITY foo "yep">]>`,
		doBlock(DTD{Name: "hi"}, DTDEntity{Name: "foo", Content: "yep"}))
	tt.Equals(t, `<!DOCTYPE hi [<!ENTITY foo '"'>]>`,
		doBlock(DTD{Name: "hi"}, DTDEntity{Name: "foo", Content: `"`}))
	tt.Pattern(t, `must only contain double or single quotes`,
		doBlockErrMsg(DTD{Name: "hi"}, DTDEntity{Name: "foo", Content: `'"`}))
}

func TestWriteRaw(t *testing.T) {
	startableRaw := func(node Startable) string {
		b, w := open()
		ec := &ErrCollector{}
		ec.Must(w.Write(Raw("wat")), w.Start(node), w.Write(Raw("wat")), w.Next(), w.Write(Raw("wat")))
		return str(b, w)
	}
	tt.Equals(t, `wat<?xml version="1.0" encoding="UTF-8"?>`+"\n"+`watwat`, startableRaw(Doc{}))
	tt.Equals(t, `wat<!DOCTYPE yeswat [wat`, startableRaw(DTD{Name: "yes"}))
	tt.Equals(t, `wat<foowat>wat`, startableRaw(Elem{Name: "foo"}))
	tt.Equals(t, `wat<![CDATA[watfoowat`, startableRaw(CData{Content: "foo"}))
	tt.Equals(t, `wat<!--watfoowat`, startableRaw(Comment{Content: "foo"}))

	b, w := open()
	(&ErrCollector{}).Must(
		w.Write(Raw("wat")), w.Start(Elem{Name: "foo"}),
		w.Write(Raw("wat"), Attr{Name: "foo"}, Raw("wat")),
		w.Next(), w.Write(Raw("wat")),
	)
	tt.Equals(t, `wat<foowat foo=""wat>wat`, str(b, w))
}

func TestAllocs(t *testing.T) {
	ec := &ErrCollector{}
	w := Open(Null{})

	_ = allocs()

	before := allocs()
	ec.Must(w.StartDoc(Doc{}))
	ec.Must(w.StartElem(Elem{Name: "foo"}))
	ec.Must(w.StartElem(Elem{Name: "bar"}))
	ec.Must(w.WriteAttr(Attr{Name: "a"}.Bool(true)))
	ec.Must(w.StartElem(Elem{Name: "baz"}))
	ec.Must(w.WriteComment(Comment{"this is a comment"}))
	ec.Must(w.WriteCData(CData{"pants pants revolution"}))
	ec.Must(w.WriteRaw("pants pants revolution"))
	ec.Must(w.EndElem("baz"))
	ec.Must(w.EndElemFull("bar"))
	ec.Must(w.EndDoc())
	after := allocs()
	tt.Equals(t, uint64(0), after-before)
	w.Flush()
}
