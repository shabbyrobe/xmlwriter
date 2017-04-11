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

type Writer struct {
	printer  printer
	nodes    []node
	current  int
	encoding string

	last Event

	// Perform validation on output
	Enforce bool

	// When checking characters, excludes an additional set of compatibility chars.
	// Defaults to true. See check.go/CheckChars.
	StrictChars bool

	// Version is placed into the `version="..."` attribute when writing a Doc{}.
	// Defaults to 1.0.
	Version string

	// Determines how much memory the internal buffer will use. Set to 0 to use
	// the default.
	InitialBufSize int

	NewlineString string

	Indenter Indenter
}

type Option func(w *Writer)

// WithIndent sets the Writer up to indent xml elements to make them
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

func Open(w io.Writer, options ...Option) *Writer {
	xw := newWriter(w, options...)
	xw.encoding = "UTF-8"
	return xw
}

func OpenEncoding(w io.Writer, encoding string, encoder *encoding.Encoder, options ...Option) *Writer {
	xw := newWriter(encoder.Writer(w), options...)
	xw.encoding = encoding
	return xw
}

func (w *Writer) Depth() int {
	return w.current
}

func (w *Writer) pushBegin(kind NodeKind, parents []NodeKind) error {
	if w.Enforce {
		if err := w.checkParent(parents...); err != nil {
			return err
		}
	}
	if len(w.nodes) <= w.current+1 {
		w.nodes = append(w.nodes, node{})
	}
	if err := w.Next(); err != nil {
		return err
	}
	return nil
}

func (w *Writer) pushEnd() error {
	w.current++
	if err := w.nodes[w.current].open(w); err != nil {
		return err
	}
	return nil
}

func (w *Writer) checkParent(kind ...NodeKind) error {
	k := NoNode
	if w.current >= 0 {
		k = w.nodes[w.current].kind
	}
	valid := false
	for _, check := range kind {
		if check == k {
			valid = true
			break
		}
	}
	if !valid {
		// this used to be an error value, but it caused the ...NodeKind
		// arg to escape to the heap.
		names := make([]string, len(kind))
		for i, k := range kind {
			names[i] = kindName[k]
		}
		return fmt.Errorf("xmlwriter: unexpected kind %s, expected %s", kindName[k], strings.Join(names, ", "))
	}
	return nil
}

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

func (w *Writer) pop(kind ...NodeKind) error {
	if w.current < 0 {
		return fmt.Errorf("xmlwriter: could not pop node")
	}
	valid := true
	if len(kind) > 0 {
		currentKind := w.nodes[w.current].kind
		for _, k := range kind {
			if currentKind != k {
				valid = false
				break
			}
		}
		if !valid {
			// this used to be an error value, but it caused the ...NodeKind
			// arg to escape to the heap.
			names := make([]string, len(kind))
			for i, k := range kind {
				names[i] = kindName[k]
			}
			return fmt.Errorf("xmlwriter: unexpected kind %s, expected %s", kindName[currentKind], strings.Join(names, ", "))
		}
	}
	if err := w.nodes[w.current].end(w); err != nil {
		return err
	}
	w.current--
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

func (w *Writer) Write(nodes ...Writable) error {
	for _, node := range nodes {
		if err := node.write(w); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) Start(nodes ...Startable) error {
	for _, node := range nodes {
		if err := node.start(w); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) Flush() error {
	return w.printer.Flush()
}

// start methods for startables

func (w *Writer) StartDoc(doc Doc) error {
	return doc.start(w)
}

func (w *Writer) StartComment(comment Comment) error {
	return comment.start(w)
}

func (w *Writer) StartCData(cdata CData) error {
	return cdata.start(w)
}

func (w *Writer) StartDTD(dtd DTD) error {
	return dtd.start(w)
}

func (w *Writer) StartDTDAttList(al DTDAttList) error {
	return al.start(w)
}

func (w *Writer) StartElem(elem Elem) error {
	return elem.start(w)
}

// write methods for startables

func (w *Writer) WriteCData(cdata CData) (err error) {
	return cdata.write(w)
}

func (w *Writer) WriteElem(elem Elem) (err error) {
	return elem.write(w)
}

func (w *Writer) WriteText(text string) (err error) {
	return Text(text).write(w)
}

func (w *Writer) WriteComment(comment Comment) (err error) {
	return comment.write(w)
}

// write methods for non-startable writables

func (w *Writer) WriteCDataContent(cdata string) error {
	return CDataContent(cdata).write(w)
}

func (w *Writer) WriteCommentContent(comment string) error {
	return CommentContent(comment).write(w)
}

func (w *Writer) WriteRaw(raw string) error {
	return Raw(raw).write(w)
}

func (w *Writer) WriteAttr(attrs ...Attr) (err error) {
	for _, a := range attrs {
		if err := a.write(w); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) WritePI(pi PI) error {
	return pi.write(w)
}

func (w *Writer) WriteDTDEntity(entity DTDEntity) error {
	return entity.write(w)
}

func (w *Writer) WriteDTDElem(el DTDElem) error {
	return el.write(w)
}

func (w *Writer) WriteNotation(n Notation) (err error) {
	return n.write(w)
}

func (w *Writer) WriteDTDAttr(attr DTDAttr) (err error) {
	return attr.write(w)
}

func (w *Writer) WriteDTDAttList(attlist DTDAttList) (err error) {
	return attlist.write(w)
}

// end methods

func (w *Writer) EndCData() (err error) {
	return w.End(CDataNode)
}

func (w *Writer) EndComment() (err error) {
	return w.End(CommentNode)
}

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

func (w *Writer) EndDTD() (err error) {
	return w.End(DTDNode)
}

func (w *Writer) EndDTDAttList() (err error) {
	return w.End(DTDAttListNode)
}

func (w *Writer) EndElemFull(name ...string) error {
	if w.current >= 0 {
		w.nodes[w.current].elem.Full = true
	}
	return w.End(ElemNode, name...)
}

func (w *Writer) EndElem(name ...string) error {
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
			return fmt.Errorf("xmlwriter: %s name '%s' did not match expected '%s'", kindName[kind], nname, exp)
		}
	case 2:
		switch kind {
		case ElemNode:
			if w.nodes[w.current].elem.Prefix != name[0] || w.nodes[w.current].elem.Name != name[1] {
				exp := name[0] + ":" + name[1]
				nname := w.nodes[w.current].elem.fullName()
				return fmt.Errorf("xmlwriter: %s name '%s' did not match expected '%s'", kindName[kind], nname, exp)
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

// EndToDepth  ends all nodes in the stack up to the supplied depth. The last
// node must match NodeKind and, if provided and applicable for the node type,
// the name. This is useful if you want to ensure that everything is closed
// inside a function:
//
//     func () {
//         d := w.Depth()
//         w.Start(Elem{Name: "pants"})
//         defer w.EndToDepth(d, NodeElem, "pants")
//         // lotsa stuff
//     }
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

func (w *Writer) writeIndent(next Event) error {
	return w.Indenter.Indent(w, w.last, next)
}
