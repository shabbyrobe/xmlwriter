/*
xmlwriter provides a fast, non-cached, forward-only way to generate XML data.

The API is based heavily on libxml's xmlwriter API [1], which is itself
based on C#'s XmlWriter [2].

  [1] http://xmlsoft.org/html/libxml-xmlwriter.html
  [2] https://msdn.microsoft.com/en-us/library/system.xml.xmlwriter(v=vs.110).aspx

It offers some advantages over Go's default encoding/xml package and some
tradeoffs. You can have complete control of the generated documents and it uses
very little memory.

There are two styles for interacting with the writer: readable and heap-friendly.
If you don't care about a few heap escapes (and most of the time you won't), you
can use the more readable API. If you are writing a code generator or your
interactions with the API are minimal, you should use the direct API.


Creating

xmlwriter.Writer{} takes any io.Writer, along with a variable list of options.

	b := &bytes.Buffer{}
	w := xmlwriter.NewWriter(b)

xmlwriter options are based on Dave Cheney's functional options pattern
(https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis):

	w := xmlwriter.NewWriter(b, xmlwriter.WitnIndent())

Provided options are:
  - WithIndent()
  - WithIndentString(string)


Overview

Using the more human-friendly API, you might express a small tree of elements
like this. These nodes will escape to the heap like crazy, but judicious use
of this nesting can make code a lot more readable:

	ec := &xmlwriter.ErrCollector{}
	defer ec.Panic()
	ec.Do(
		w.Start(xmlwriter.Doc{}),
		w.Start(xmlwriter.Elem{
			Name: "foo", Attrs: []xmlwriter.Attr{
				{Name: "a1", Value: "val1"},
				{Name: "a2", Value: "val2"},
			},
			Content: []xmlwriter.Writable{
				xmlwriter.Comment{"hello"},
				xmlwriter.Elem{
					Name: "bar", Attrs: []xmlwriter.Attr{
						{Name: "a1", Value: "val1"},
						{Name: "a2", Value: "val2"},
					},
					Content: []xmlwriter.Writable{
						xmlwriter.Elem{Name: "baz"},
					},
				},
			},
		}),
		w.EndAllFlush(),
	)

The code can be made even less dense by importing xmlwriter with a prefix:
`import xw "github.com/shabbyrobe/xmlwriter"`

Using the more Heap-friendy API to produce the same output. This has a lot more
stutter and a lot worse signal to noise ratio, and it's harder to tell the
hierarchical relationship just by looking at the code, but there are no heap
escapes this way:

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

Use whichever API reads best in your code, but favour the latter style in
all code generators and performance hotspots.


Flush

xmlwriter.Writer extends bufio.Writer! Don't forget to flush otherwise you'll
lose data.

There are two ways to flush:

	- Writer.Flush()
	- Writer.EndAllFlush()

The EndAllFlush form is just a convenience, it calls EndAll() and Flush() for you.


Start and Write and Block

Nodes which can have children can be passed to `Writer.Start()`. This adds
them to the stack and opens them, allowing children to be added.

	w.Start(xmlwriter.Elem{Name: "foo"},
		xmlwriter.Elem{Name: "bar"},
		xmlwriter.Elem{Name: "baz"})
	w.EndAllFlush()

Becomes: <foo><bar><baz/></bar></foo>

Nodes which have no children, or nodes which can be opened and fully closed
with only a trivial amount of informatin, can be passed to `Writer.Write()`.
If written nodes are put on to the stack, they will be popped before Write
returns.

	w.Write(xmlwriter.Elem{Name: "foo"},
		xmlwriter.Elem{Name: "bar"},
		xmlwriter.Elem{Name: "baz"})

Becomes: <foo/><bar/><baz/>

Block takes a Startable and a variable number of Writable nodes. The Startable
will be opened, the Writables will be written, then the Startable will be closed:

	w.Block(xmlwriter.Elem{Name: "foo"},
		xmlwriter.Comment{"comment!"},
		xmlwriter.CData{"cdata."},
		xmlwriter.Elem{Name: "bar"},
	)

Becomes:
	<foo><!--comment!--><![CDATA[cdata.]><bar/></foo>


End

There are several ways to end an element. Choose the End that's right for you!

	- EndAll()
	- EndAllFlush()
	- EndAny()
	- End(NodeKind)
	- End(NodeKind, name...)
	- EndToDepth(int, NodeKind)
	- EndToDepth(int, NodeKind, name...)
	- EndDoc()
	- End...()
		Where ... is the name of a startable node struct, ends that node kind.
		Equivalent to


Nodes

Nodes as they are written can be in three states: StateOpen, StateOpened or
StateEnd. StateOpen == "<elem". StateOpened == "<elem>". StateEnd ==
"<elem></elem>".

The following Node structs are available for writing in the following
hierarchy. Nodes which are "Startable" (passed to `writer.Start(n)`) are marked
with an S. Nodes which are "Writable" (passed to `writer.Write(n)`) are marked
with a W.

- xmlwriter.Raw* (W)
- xmlwriter.Doc (S)
	- xmlwriter.DTD (S)
		- xmlwriter.DTDEntity (W)
		- xmlwriter.DTDElement (W)
		- xmlwriter.DTDAttrList (S, W)
			- xmlwriter.DTDAttr (W)
		- xmlwriter.Notation (W)
		- xmlwriter.Comment (S, W)
	- xmlwriter.PI (W)
	- xmlwriter.Comment (S, W)
	- xmlwriter.Elem (S, W)
		- xmlwriter.Attr (W)
		- xmlwriter.PI (W)
		- xmlwriter.Text (W)
		- xmlwriter.Comment (S, W)
			- xmlwriter.CommentContent (W)
		- xmlwriter.CData (S, W)
			- xmlwriter.CDataContent (W)

* `xmlwriter.Raw` can be written anywhere, at any time. If a node is in
the "open" state but not in the "opened" state, for example you have started
an element and written an attribute, writing "raw" will add the content to
the inside of the element opening tag unless you call `w.Next()`.

Every node has a corresponding NodeKind integer, which can be found by
affixing "Node" to the struct name, i.e. "xmlwriter.Elem" becomes
"xmlwriter.ElemNode". These are used for calls to Writer.End().

xmlwriter.Attr{} values can be assigned from any golang primitive like so:

	Attr{Name: "foo"}.Int(5)
	Attr{Name: "foo"}.Uint64(5)
	Attr{Name: "foo"}.Float64(1.234)

*/
package xmlwriter
