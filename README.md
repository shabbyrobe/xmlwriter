xmlwriter
=========

[![GoDoc](https://godoc.org/github.com/shabbyrobe/xmlwriter?status.svg)](https://godoc.org/github.com/shabbyrobe/xmlwriter)
![Go](https://github.com/shabbyrobe/xmlwriter/workflows/Go/badge.svg)

xmlwriter is a pure-Go library providing a procedural XML generation API based
on libxml2's xmlwriter module. The package is extensively documented at
[GoDoc](https://godoc.org/github.com/shabbyrobe/xmlwriter).

Quick example:

```go
func main() {
    b := &bytes.Buffer{}
    w := xmlwriter.Open(b)
    ec := &xmlwriter.ErrCollector{}
    defer ec.Panic()

    ec.Do(
        w.StartDoc(xmlwriter.Doc{})
        w.StartElem(xmlwriter.Elem{Name: "foo"})
        w.WriteAttr(xmlwriter.Attr{Name: "a1", Value: "val1"})
        w.WriteAttr(xmlwriter.Attr{Name: "a2", Value: "val2"})
        w.WriteComment(xmlwriter.Comment{"hello"})
        w.StartElem(xmlwriter.Elem{Name: "bar"})
        w.WriteAttr(xmlwriter.Attr{Name: "a1", Value: "val1"})
        w.WriteAttr(xmlwriter.Attr{Name: "a2", Value: "val2"})
        w.StartElem(xmlwriter.Elem{Name: "baz"})
        w.EndAllFlush()
    )
    fmt.Println(b.String())
}
```

xmlwriter is about twice as quick as using the stdlib's `encoding/xml` and
offers total control of the output. If you don't require that level of control,
it's probably better to stick with `encoding/xml`

    BenchmarkWriterHuge-8     	     165	   7189290 ns/op	    4944 B/op	       4 allocs/op
    BenchmarkWriterSmall-8    	  299679	      4035 ns/op	    4944 B/op	       4 allocs/op
    BenchmarkGolangHuge-8      	      52	  21770422 ns/op	 4324496 B/op	   60008 allocs/op
    BenchmarkGolangSmall-8    	  139767	      8828 ns/op	    5936 B/op	      28 allocs/op

xmlwriter is exhaustively tested using a fairly insane mess of C scripts you
can find in the `tester/` directory.


License
-------

xmlwriter uses the Apache License 2.0. I pulled in about 60 lines of code from
the `xml/encoding` package in the Go sources and retained the copyright. Not sure 
the exact implications, IANAL. Please file an issue if I've done something wrong.

