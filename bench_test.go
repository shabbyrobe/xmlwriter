package xmlwriter

import (
	"encoding/xml"
	"testing"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func BenchmarkWriterGeneral(b *testing.B) {
	for i := 0; i < b.N; i++ {
		w := Open(Null{})

		must(w.StartDoc(Doc{}))
		must(w.StartElem(Elem{Name: "foo"}))
		must(w.StartElem(Elem{Name: "bar"}))
		must(w.WriteAttr(Attr{Name: "a"}.Bool(true)))
		must(w.StartElem(Elem{Name: "baz"}))
		must(w.WriteElem(Elem{Name: "test", Attrs: []Attr{{Name: "foo"}}}))
		must(w.WriteElem(Elem{Name: "test"}))
		must(w.WriteElem(Elem{Name: "test"}))
		must(w.WriteElem(Elem{Name: "test"}))
		must(w.WriteElem(Elem{Name: "test"}))
		must(w.StartComment(Comment{}))
		must(w.WriteCommentContent("this is  a comment"))
		must(w.WriteCommentContent("this is  a comment"))
		must(w.EndComment())
		must(w.WriteCData(CData{"pants pants revolution"}))
		must(w.EndElemFull("baz"))
		must(w.EndDoc())
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

		must(w.StartDoc(Doc{}))
		must(w.StartElem(Elem{Name: o.Name}))
		for _, c := range o.Inners {
			must(w.StartElem(Elem{Name: "inner"}))
			must(w.WriteAttr(Attr{Name: "name", Value: c.Name}, Attr{Name: "value", Value: c.Value}))
			must(w.End(ElemNode))
		}
		must(w.EndAllFlush())
	}
}

func benchmarkGolang(b *testing.B, cnt int) {
	b.StopTimer()
	o := makeStruct(cnt)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		must(xml.NewEncoder(Null{}).Encode(o))
	}
}

func BenchmarkGolangHuge(b *testing.B) {
	benchmarkGolang(b, 30000)
}

func BenchmarkGolangSmall(b *testing.B) {
	benchmarkGolang(b, 10)
}
