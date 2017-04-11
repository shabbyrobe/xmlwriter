package xmlwriter

import (
	"encoding/xml"
	"testing"
)

func BenchmarkWriterGeneral(b *testing.B) {
	for i := 0; i < b.N; i++ {
		w := Open(Null{})

		Must(w.StartDoc(Doc{}))
		Must(w.StartElem(Elem{Name: "foo"}))
		Must(w.StartElem(Elem{Name: "bar"}))
		Must(w.WriteAttr(Attr{Name: "a"}.Bool(true)))
		Must(w.StartElem(Elem{Name: "baz"}))
		Must(w.WriteElem(Elem{Name: "test", Attrs: []Attr{{Name: "foo"}}}))
		Must(w.WriteElem(Elem{Name: "test"}))
		Must(w.WriteElem(Elem{Name: "test"}))
		Must(w.WriteElem(Elem{Name: "test"}))
		Must(w.WriteElem(Elem{Name: "test"}))
		Must(w.StartComment(Comment{}))
		Must(w.WriteCommentContent("this is  a comment"))
		Must(w.WriteCommentContent("this is  a comment"))
		Must(w.EndComment())
		Must(w.WriteCData(CData{"pants pants revolution"}))
		Must(w.EndElemFull("baz"))
		Must(w.EndDoc())
		w.Flush()
	}
}

type Outer struct {
	Name   string  `xml:"name,attr"`
	Inners []Inner `xml:"inner"`
}

type Inner struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

func makeStruct(cnt int) *Outer {
	names := []string{"foo", "bar", "baz", "qux", "pants", "trou"}
	values := []string{"yep", "nup", "wahey", "ding", "dong"}
	o := &Outer{Name: "hi", Inners: make([]Inner, cnt)}
	for i := 0; i < cnt; i++ {
		o.Inners[i] = Inner{Name: names[i%len(names)], Value: values[i%len(values)]}
	}
	return o
}

func BenchmarkWriterHuge(b *testing.B) {
	benchmarkWriter(b, 30000)
}

func BenchmarkWriterSmall(b *testing.B) {
	benchmarkWriter(b, 10)
}

func benchmarkWriter(b *testing.B, cnt int) {
	b.StopTimer()
	o := makeStruct(cnt)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		w := Open(Null{})

		Must(w.StartDoc(Doc{}))
		Must(w.StartElem(Elem{Name: o.Name}))
		for _, c := range o.Inners {
			Must(w.StartElem(Elem{Name: "inner"}))
			Must(w.WriteAttr(Attr{Name: "name", Value: c.Name}, Attr{Name: "value", Value: c.Value}))
			Must(w.End(ElemNode))
		}
		Must(w.EndAllFlush())
	}
}

func benchmarkGolang(b *testing.B, cnt int) {
	b.StopTimer()
	o := makeStruct(cnt)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		Must(xml.NewEncoder(Null{}).Encode(o))
	}
}

func BenchmarkGolangHuge(b *testing.B) {
	benchmarkGolang(b, 30000)
}

func BenchmarkGolangSmall(b *testing.B) {
	benchmarkGolang(b, 10)
}
