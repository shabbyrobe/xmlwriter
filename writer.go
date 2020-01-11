package xmlwriter

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding"
)

const (
	initialNodeDepth = 8
	defaultBufsize   = 2048
)

// Writer writes XML to an io.Writer.
type Writer struct {
	printer  printer
	nodes    []node
	current  int
	encoding string

	last Event

	// Perform validation on output. Defaults to true when created using Open().
	Enforce bool

	// When enforcing and checking characters, excludes an additional set of
	// compatibility chars.  Defaults to true. See check.go/CheckChars.
	StrictChars bool

	// Version is placed into the `version="..."` attribute when writing a Doc{}.
	// Defaults to 1.0.
	Version string

	// Determines how much memory the internal buffer will use. Set to 0 to use
	// the default.
	InitialBufSize int

	// Defaults to \n.
	NewlineString string

	// Controls the indenting process used by the writer.
	Indenter Indenter
}

// Option is an option to the Writer.
type Option func(w *Writer)

// WithIndent sets the Writer up to indent XML elements to make them
// easier to read using the StandardIndenter:
//	w := xmlwriter.Open(b, xmlwriter.WithIndent())
func WithIndent() Option {
	return func(w *Writer) {
		w.Indenter = NewStandardIndenter()
	}
}

// WithIndentString configures the Writer with a StandardIndenter using
// a specific indent string:
//	w := xmlwriter.Open(b, xmlwriter.WithIndentString("    "))
func WithIndentString(indent string) Option {
	return func(w *Writer) {
		si := NewStandardIndenter()
		si.IndentString = indent
		w.Indenter = si
	}
}

func newWriter(w io.Writer, options ...Option) *Writer {
	xw := &Writer{}
	xw.current = -1
	xw.NewlineString = "\n"
	xw.nodes = make([]node, initialNodeDepth)
	xw.Enforce = true
	xw.StrictChars = true
	for _, o := range options {
		o(xw)
	}
	if xw.InitialBufSize <= 0 {
		xw.InitialBufSize = defaultBufsize
	}
	xw.printer = printer{Writer: bufio.NewWriterSize(w, xw.InitialBufSize)}
	return xw
}

// Open opens the Writer using the UTF-8 encoding.
func Open(w io.Writer, options ...Option) *Writer {
	xw := newWriter(w, options...)
	xw.encoding = "UTF-8"
	return xw
}

// OpenEncoding opens the Writer using the supplied encoding.
//
// This example opens an XML writer using the utf16-be encoding:
//
//	enc := unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM).NewEncoder()
//	w := xmlwriter.OpenEncoding(b, "utf-16be", enc)
//
// You should still write UTF-8 strings to the writer - they are converted
// on the fly to the target encoding.
//
func OpenEncoding(w io.Writer, encstr string, encoder *encoding.Encoder, options ...Option) *Writer {
	enc := encoding.HTMLEscapeUnsupported(encoder).Writer(w)
	xw := newWriter(enc, options...)
	xw.encoding = encstr
	return xw
}

// Depth returns the number of opened Startable nodes on the stack.
func (w *Writer) Depth() int {
	return w.current
}

// Next ensures that the current element is opened and the next one
// can be started. It is called for all node types except Raw{}. This
// is so that you can write raw strings inside elements that are open
// but not opened. See StateOpen and StateOpened for more details on
// this distinction.
func (w *Writer) Next() error {
	if w.current >= 0 {
		w.nodes[w.current].children++
		if w.nodes[w.current].state == StateOpen {
			if err := w.nodes[w.current].opened(w); err != nil {
				return err
			}
		}
	}
	return nil
}

// Block is a convenience function that takes a parent node and a list of
// direct children. The parent node is passed to Writer.Start(), the
// children are passed to Write(), then the parent passed to End().
func (w *Writer) Block(start Startable, nodes ...Writable) error {
	if err := w.Start(start); err != nil {
		return err
	}
	for _, n := range nodes {
		if err := w.Write(n); err != nil {
			return err
		}
	}
	if err := w.End(start.kind()); err != nil {
		return err
	}
	return nil
}

// Write writes writable nodes.
func (w *Writer) Write(nodes ...Writable) error {
	for _, node := range nodes {
		if err := node.write(w); err != nil {
			return err
		}
	}
	return nil
}

// Start starts a startable node.
func (w *Writer) Start(nodes ...Startable) error {
	for _, node := range nodes {
		if err := node.start(w); err != nil {
			return err
		}
	}
	return nil
}

// Flush ensures the output buffer accumuated inside the Writer
// is fully written to the underlying io.Writer.
func (w *Writer) Flush() error {
	return w.printer.Flush()
}

// {{{ start methods for startables

// StartDoc pushes an XML document node onto the writer's stack.
func (w *Writer) StartDoc(doc Doc) error { return doc.start(w) }

// StartComment pushes an XML comment node onto the writer's stack.
// WriteCommentContent can be used to write contents.
func (w *Writer) StartComment(comment Comment) error { return comment.start(w) }

// StartCData pushes an XML CData node onto the writer's stack.
// WriteCDataContent can be used to write contents.
func (w *Writer) StartCData(cdata CData) error { return cdata.start(w) }

// StartDTD pushes a Document Type Declaration node onto the writer's
// stack.
func (w *Writer) StartDTD(dtd DTD) error { return dtd.start(w) }

// StartDTDAttList pushes a Document Type Declaration node onto the writer's
// stack.
func (w *Writer) StartDTDAttList(al DTDAttList) error { return al.start(w) }

// StartElem pushes an XML element node onto the writer's stack.
func (w *Writer) StartElem(elem Elem) error { return elem.start(w) }

// }}}

// {{{ write methods for startables

// WriteCData writes a complete XML CData section. It can be written inside an
// Elem or as a top-level node.
func (w *Writer) WriteCData(cdata CData) (err error) { return cdata.write(w) }

// WriteComment writes a complete XML Comment section. It can be written inside an
// Elem, a DTD, a Doc, or as a top-level node.
func (w *Writer) WriteComment(comment Comment) (err error) { return comment.write(w) }

// WriteElem writes a complete XML Element. It can be written inside an
// Elem, a Doc, or as a top-level node.
//
// Nested elements and attributes can be written by assigning to the members
// of the Elem struct to an arbitrary depth (though be warned that this can
// cause heap escapes):
//
//	e := Elem{
//		Name: "outer",
//		Attrs: []Attr{Attr{Name: "key", Value: "val"}},
//		Content: []Writable{Elem{Name: "inner"}},
//	}
//
func (w *Writer) WriteElem(elem Elem) (err error) { return elem.write(w) }

// }}}

// {{{ write methods for non-startable writables

// WriteCDataContent writes text inside an already-started XML CData node.
func (w *Writer) WriteCDataContent(cdata string) error { return CDataContent(cdata).write(w) }

// WriteCommentContent writes text inside an already-started XML Comment node.
func (w *Writer) WriteCommentContent(comment string) error { return CommentContent(comment).write(w) }

// WriteDTDEntity writes a DTD Entity definition to the output. It can be
// written inside a DTD or as a top-level node.
func (w *Writer) WriteDTDEntity(entity DTDEntity) error { return entity.write(w) }

// WriteDTDElem writes a DTD Element definition to the output. It can be
// written inside a DTD or as a top-level node.
func (w *Writer) WriteDTDElem(el DTDElem) error { return el.write(w) }

// WriteDTDAttr writes a DTD Attribute definition to the output. It can be
// written inside a DTDAttList or as a top-level node.
func (w *Writer) WriteDTDAttr(attr DTDAttr) (err error) { return attr.write(w) }

// WriteDTDAttList writes a DTD Attribute List to the output. It can be written inside
// a DTD or as a top-level node.
func (w *Writer) WriteDTDAttList(attlist DTDAttList) (err error) { return attlist.write(w) }

// WriteNotation writes an XML notation to the output. It can be written inside
// a DTD or as a top-level node.
func (w *Writer) WriteNotation(n Notation) (err error) { return n.write(w) }

// WritePI writes an XML processing instruction to the output. It can be
// written inside a Doc, an Elem or as a top-level node.
func (w *Writer) WritePI(pi PI) error { return pi.write(w) }

// WriteText writes an XML text node to the output. It will be appropriately
// escaped. It can be written inside an Elem or as a top-level node.
func (w *Writer) WriteText(text string) (err error) { return Text(text).write(w) }

// WriteRaw writes a raw string to the output. This can be any string
// whatsoever - it does not have to be valid XML and will be written exactly as
// it is declared. Raw nodes can be written at any stage of the writing
// process.
func (w *Writer) WriteRaw(raw string) error { return Raw(raw).write(w) }

// WriteAttr writes one or more XML element attributes to the output.
func (w *Writer) WriteAttr(attrs ...Attr) (err error) {
	for _, a := range attrs {
		if err := a.write(w); err != nil {
			return err
		}
	}
	return nil
}

// }}}

// {{{ end methods

// EndDoc ends the current xmlwriter.Doc{} and any node in between. This is not
// the same as calling End(DocNode), which will only end a Doc{} if it is the
// current node on the stack.
func (w *Writer) EndDoc() (err error) {
	for {
		if w.current <= 0 {
			break
		}
		if err = w.pop(); err != nil {
			return
		}
	}
	return w.End(DocNode)
}

// EndCData pops a CData node from the writer's stack, or returns an error
// if the current node is not a CData.
func (w *Writer) EndCData() (err error) { return w.End(CDataNode) }

// EndComment pops a Comment node from the writer's stack, or returns an error
// if the current node is not a Comment.
func (w *Writer) EndComment() (err error) { return w.End(CommentNode) }

// EndDTD pops a DTD node from the writer's stack, or returns an error
// if the current node is not a DTD.
func (w *Writer) EndDTD() (err error) { return w.End(DTDNode) }

// EndDTDAttList pops a DTDAttList node from the writer's stack, or returns an
// error if the current node is not a DTDAttList.
func (w *Writer) EndDTDAttList() (err error) { return w.End(DTDAttListNode) }

// EndElem pops an Elem node from the writer's stack, or returns an error if
// the current node is not an Elem. If the Elem has had no children written, it
// will be closed using the short close style: "<tag/>"
func (w *Writer) EndElem(name ...string) error { return w.End(ElemNode, name...) }

// EndElemFull pops an Elem node from the writer's stack, or returns an error if
// the current node is not an Elem. It will always be closed using the full
// element close style even if it contains no children: "<tag></tag>".
func (w *Writer) EndElemFull(name ...string) error {
	if w.current >= 0 {
		w.nodes[w.current].elem.Full = true
	}
	return w.End(ElemNode, name...)
}

// EndAny ends the current node, regardless of what kind of node it is.
func (w *Writer) EndAny() error {
	if w.current < 0 {
		return fmt.Errorf("xmlwriter: could not pop node")
	}
	return w.pop()
}

// End ends the current node if it is of the kind specified, and also
// optionally (and if relevant) if the name matches.
// If one string is passed to name, it is compared to the node's Name field.
// If two strings are passed to name, the first is compared to the node's Prefix
// field, the second is compared to the node's Name field.
// This form works with the following node types: ElemNode, DTDNode,
// DTDAttListNode.
func (w *Writer) End(kind NodeKind, name ...string) error {
	if w.current < 0 {
		return fmt.Errorf("xmlwriter: could not pop node")
	}
	switch len(name) {
	case 0:
	case 1:
		nname := ""
		switch kind {
		case ElemNode:
			nname = w.nodes[w.current].elem.Name
		case DTDNode:
			nname = w.nodes[w.current].dtd.Name
		case DTDAttListNode:
			nname = w.nodes[w.current].dtdAttList.Name
		default:
			return fmt.Errorf("xmlwriter: tried to pop named, but node was not named")
		}
		if nname != name[0] {
			exp := name[0]
			return fmt.Errorf("xmlwriter: %s name %q did not match expected %q", kind.Name(), nname, exp)
		}
	case 2:
		switch kind {
		case ElemNode:
			if w.nodes[w.current].elem.Prefix != name[0] || w.nodes[w.current].elem.Name != name[1] {
				exp := name[0] + ":" + name[1]
				nname := w.nodes[w.current].elem.fullName()
				return fmt.Errorf("xmlwriter: %s name %q did not match expected %q", kind.Name(), nname, exp)
			}
		default:
			return fmt.Errorf("xmlwriter: tried to pop named, but node was not named")
		}
	default:
		return fmt.Errorf("xmlwriter: invalid name")
	}
	return w.pop(kind)
}

// EndAll ends every node on the stack
func (w *Writer) EndAll() error {
	for {
		if w.current < 0 {
			break
		}
		if err := w.pop(); err != nil {
			return err
		}
	}
	return nil
}

// EndToDepth ends all nodes in the stack up to the supplied depth. The last
// node must match NodeKind and, if provided and applicable for the node type,
// the name. This is useful if you want to ensure that everything you open inside
// a particular scope is closed at the end:
//
//	func outer() {
//		w.Start(Elem{Name: "pants"})
//		inner()
//      // result:
//      // <pants><foo><bar/></foo>
//	}
//	func inner() {
//		d := w.Depth()
//		defer w.EndToDepth(d, NodeElem)
//		w.Start(Elem{Name: "foo"})
//		w.Start(Elem{Name: "bar"})
//	}
//
func (w *Writer) EndToDepth(depth int, kind NodeKind, name ...string) error {
	limit := depth + 1
	for {
		if w.current <= limit {
			break
		}
		if err := w.pop(); err != nil {
			return err
		}
	}
	return w.End(kind, name...)
}

// EndAllFlush ends every node on the stack and calls Flush()
func (w *Writer) EndAllFlush() error {
	if err := w.EndAll(); err != nil {
		return err
	}
	return w.Flush()
}

// }}}

// {{{ Internal functions

func (w *Writer) writeIndent(next Event) error {
	return w.Indenter.Indent(w, w.last, next)
}

func (w *Writer) pushBegin(kind NodeKind, parents nodeFlag) error {
	if w.Enforce {
		if err := w.checkParent(parents); err != nil {
			return err
		}
	}
	if len(w.nodes) <= w.current+1 {
		w.nodes = append(w.nodes, node{})
	}
	return w.Next()
}

func (w *Writer) pushEnd() error {
	w.current++
	return w.nodes[w.current].open(w)
}

func (w *Writer) checkParent(nodeFlags nodeFlag) error {
	currentFlag := noNodeFlag
	if w.current >= 0 {
		currentFlag = w.nodes[w.current].flag
	}

	if currentFlag&nodeFlags == 0 {
		// this used to be an error value, but it caused the ...NodeKind
		// arg to escape to the heap.
		names := nodeFlags.names()

		var currentKind NodeKind
		if w.current >= 0 {
			currentKind = w.nodes[w.current].kind
		}

		return fmt.Errorf("xmlwriter: unexpected kind %s, expected %s", currentKind.Name(), names)
	}

	return nil
}

func (w *Writer) writeBeginNext(kind NodeKind) error {
	w.Next()
	return w.writeBeginCur(kind)
}

func (w *Writer) writeBeginCur(kind NodeKind) error {
	if w.Indenter != nil {
		if err := w.writeIndent(Event{StateOpen, kind, 0}); err != nil {
			return err
		}
		w.last = Event{StateEnded, kind, 0}
	}
	return nil
}

func (w *Writer) pop(kinds ...NodeKind) error {
	if w.current < 0 {
		return fmt.Errorf("xmlwriter: could not pop node")
	}

	valid := true
	if len(kinds) > 0 {
		currentKind := w.nodes[w.current].kind
		for _, k := range kinds {
			if currentKind != k {
				valid = false
				break
			}
		}
		if !valid {
			// this used to be an error value, but it caused the ...NodeKind
			// arg to escape to the heap for all branches, not just this one:
			names := make([]string, len(kinds))
			for i, nk := range kinds {
				names[i] = nk.Name()
			}
			return fmt.Errorf("xmlwriter: unexpected kind %s, expected %s", currentKind.Name(), strings.Join(names, ", "))
		}
	}
	if err := w.nodes[w.current].end(w); err != nil {
		return err
	}
	w.current--
	return nil
}

// }}}
