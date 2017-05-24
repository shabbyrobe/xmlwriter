package xmlwriter

import (
	"strings"
	"testing"

	tt "github.com/shabbyrobe/xmlwriter/testtool"
)

func TestIndentElemAttr(t *testing.T) {
	result := strings.Join([]string{
		"<a>",
		" <b foo=\"bar\">",
		"  <c/>",
		"  <c/>",
		" </b>",
		"</a>",
	}, "\n")
	b, w := open(WithIndent())
	must(w.Start(Elem{Name: "a"}))
	must(w.Start(Elem{Name: "b", Attrs: []Attr{{Name: "foo", Value: "bar"}}}))
	must(w.Write(Elem{Name: "c"}, Elem{Name: "c"}))
	must(w.EndAll())
	tt.Equals(t, result, str(b, w))
}

func TestIndentElemTextComplex(t *testing.T) {
	result := strings.Join([]string{
		"<a>",
		" <b>Hi my name is <judge/>. Judge <my>",
		"   <name>",
		"    <is/>",
		"   </name> foo bar baz</my></b>",
		"</a>",
	}, "\n")
	b, w := open(WithIndent())
	(&ErrCollector{}).Must(
		w.Start(Elem{Name: "a"}, Elem{Name: "b"}),
		w.Write(Text("Hi my name is "), Elem{Name: "judge"}, Text(". Judge ")),
		w.Start(Elem{Name: "my"}, Elem{Name: "name"}),
		w.Write(Elem{Name: "is"}),
		w.End(ElemNode),
		w.Write(Text(" foo bar baz")),
		w.EndAll(),
	)
	tt.Equals(t, result, str(b, w))
}

func TestIndentEmptyInlineBetweenText(t *testing.T) {
	result := strings.Join([]string{
		"<a>",
		" <b>Hi my name is <judge/>.</b>",
		"</a>",
	}, "\n")
	b, w := open(WithIndent())
	must(w.Start(Elem{Name: "a"}, Elem{Name: "b"}))
	must(w.Write(Text("Hi my name is "), Elem{Name: "judge"}, Text(".")))
	must(w.EndAll())
	tt.Equals(t, result, str(b, w))
}

func TestIndentEmptyInlineAfterText(t *testing.T) {
	result := strings.Join([]string{
		"<a>",
		" <b>Hi my name is <judge/></b>",
		"</a>",
	}, "\n")
	b, w := open(WithIndent())
	must(w.Start(Elem{Name: "a"}, Elem{Name: "b"}))
	must(w.Write(Text("Hi my name is "), Elem{Name: "judge"}))
	must(w.EndAll())
	tt.Equals(t, result, str(b, w))
}

func TestIndentDoc(t *testing.T) {
	result := strings.Join([]string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		"<a>",
		" <b/>",
		"</a>",
	}, "\n") + "\n"
	b, w := open(WithIndent())
	must(w.Block(Doc{},
		Elem{Name: "a", Content: []Writable{Elem{Name: "b"}}},
	))
	tt.Equals(t, result, str(b, w))
}

func TestIndentDTD(t *testing.T) {
	result := strings.Join([]string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<!DOCTYPE pants [`,
		` <!NOTATION yep SYSTEM "sys">`,
		` <!ELEMENT foo EMPTY>`,
		` <!ENTITY hi "yep">`,
		` <!ATTLIST att`,
		`   foo CDATA #IMPLIED`,
		`   bar CDATA #IMPLIED`,
		` >`,
		`]>`,
		`<foo/>`,
	}, "\n") + "\n"
	b, w := open(WithIndent())
	must(w.Start(Doc{}, DTD{Name: "pants"}))
	must(w.Write(
		Notation{Name: "yep", SystemID: "sys"},
		DTDElem{Name: "foo", Decl: DTDElemEmpty},
		DTDEntity{Name: "hi", Content: "yep"},
		DTDAttList{Name: "att", Attrs: []DTDAttr{
			{Name: "foo", Type: DTDAttrString},
			{Name: "bar", Type: DTDAttrString},
		}},
	))
	must(w.End(DTDNode))
	must(w.Write(Elem{Name: "foo"}))
	must(w.EndAll())
	tt.Equals(t, result, str(b, w))
}

func TestIndentComment(t *testing.T) {
	result := strings.Join([]string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<a>`,
		` <b>`,
		`  <!--hi how are you-->`,
		` </b>`,
		`</a>`,
	}, "\n") + "\n"
	b, w := open(WithIndent())
	must(w.Start(Doc{}))
	must(w.Start(Elem{Name: "a"}))
	must(w.Start(Elem{Name: "b"}))
	must(w.Start(Comment{Content: "hi how are you"}))
	must(w.EndAll())

	tt.Equals(t, result, str(b, w))
}

func TestIndentCData(t *testing.T) {
	result := strings.Join([]string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<a>`,
		` <b><![CDATA[hi how are you]]></b>`,
		`</a>`,
	}, "\n") + "\n"
	b, w := open(WithIndent())
	must(w.Start(Doc{}))
	must(w.Start(Elem{Name: "a"}))
	must(w.Start(Elem{Name: "b"}))
	must(w.Start(CData{Content: "hi how are you"}))
	must(w.EndAll())

	tt.Equals(t, result, str(b, w))
}

func TestIndentRaw(t *testing.T) {
	result := strings.Join([]string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`wat<awat a="b"wat/>`,
	}, "\n") + "\n"
	b, w := open(WithIndent())

	// TODO: more permutations of elements
	(&ErrCollector{}).Must(
		w.Start(Doc{}),
		w.Write(Raw("wat")),
		w.Start(Elem{Name: "a"}),
		w.Write(Raw("wat")),
		w.Write(Attr{Name: "a", Value: "b"}),
		w.Write(Raw("wat")),
		w.EndAll(),
	)
	tt.Equals(t, result, str(b, w))
}
