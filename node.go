package xmlwriter

// NodeState is the state of the node being written.
type NodeState int

// Name returns a string representation fo the NodeState.
func (n NodeState) Name() string { return stateName[n] }

const (
	// StateOpen indicates the node is open but not opened, e.g. "<elem"
	StateOpen NodeState = iota

	// StateOpened indicates the node is fully opened, e.g. "<elem>"
	StateOpened

	// StateEnded indicates the node's end, e.g. "</elem>"
	StateEnded
)

var stateName = map[NodeState]string{
	StateOpen:   "open",
	StateOpened: "opened",
	StateEnded:  "ended",
}

// Node represents an item which the Writer can Start and/or Write. It
// is the wider type for Startable and Writable.
type Node interface {
	kind() NodeKind
}

// Writable is a node which can be passed to xmlwriter.Write(). Writables
// may or may not have children. Writable nodes can not be passed to
// xmlwriter.Start().
type Writable interface {
	Node
	write(w *Writer) error
}

// NOTE: startable node argument:
// Types passed in to the writer API should be value types not pointer types,
// but we still need to keep a writable reference to them available. these
// methods don't have pointer receivers, but the writable version is available
// via the bogus tagged union in the node argument. If a Startable needs to
// modify itself, it should do so via the node pointer.

// Startable is a node which can be passed to xmlwriter.Start() - any node
// which can have children is Startable.
type Startable interface {
	Node

	start(w *Writer) error
	open(n *node, w *Writer) error
	opened(n *node, w *Writer, prev NodeState) error
	end(n *node, w *Writer, prev NodeState) error
}

type node struct {
	children int
	state    NodeState

	// if any of the children added to this node wrote an indent,
	// this will be true. for e.g. libxml2 doesn't indent the closing tag
	// of an element if the element only contains cdata or comments
	hasIndenter bool

	// bogus tagged union, this keeps things from escaping to the heap
	kind       NodeKind
	flag       nodeFlag
	comment    Comment
	cdata      CData
	doc        Doc
	dtd        DTD
	dtdAttList DTDAttList
	elem       Elem
}

func (n *node) clear() {
	*n = node{}
}

func (n *node) open(w *Writer) error {
	var err error

	n.state = StateOpen

	if w.Indenter != nil {
		ev := Event{n.state, n.kind, n.children}
		if err := w.writeIndent(ev); err != nil {
			return err
		}
		w.last = Event{n.state, n.kind, n.children}
	}

	switch n.kind {
	case CommentNode:
		err = n.comment.open(n, w)
	case CDataNode:
		err = n.cdata.open(n, w)
	case DocNode:
		err = n.doc.open(n, w)
	case DTDNode:
		err = n.dtd.open(n, w)
	case DTDAttListNode:
		err = n.dtdAttList.open(n, w)
	case ElemNode:
		err = n.elem.open(n, w)
	default:
		// FIXME
		panic(nil)
	}
	return err
}

func (n *node) opened(w *Writer) error {
	var err error

	st := n.state
	n.state = StateOpened

	if w.Indenter != nil {
		ev := Event{n.state, n.kind, n.children}
		if err := w.writeIndent(ev); err != nil {
			return err
		}
	}

	switch n.kind {
	case CommentNode:
		err = n.comment.opened(n, w, st)
	case CDataNode:
		err = n.cdata.opened(n, w, st)
	case DocNode:
		err = n.doc.opened(n, w, st)
	case DTDNode:
		err = n.dtd.opened(n, w, st)
	case DTDAttListNode:
		err = n.dtdAttList.opened(n, w, st)
	case ElemNode:
		err = n.elem.opened(n, w, st)
	default:
		// FIXME
		panic(nil)
	}
	if w.Indenter != nil {
		w.last = Event{n.state, n.kind, n.children}
	}
	return err
}

func (n *node) end(w *Writer) error {
	var err error

	st := n.state
	if st == StateOpen {
		if err := w.nodes[w.current].opened(w); err != nil {
			return err
		}
	}

	n.state = StateEnded
	if w.Indenter != nil {
		ev := Event{n.state, n.kind, n.children}
		if err := w.writeIndent(ev); err != nil {
			return err
		}
	}

	switch n.kind {
	case CommentNode:
		err = n.comment.end(n, w, st)
	case CDataNode:
		err = n.cdata.end(n, w, st)
	case DocNode:
		err = n.doc.end(n, w, st)
	case DTDAttListNode:
		err = n.dtdAttList.end(n, w, st)
	case DTDNode:
		err = n.dtd.end(n, w, st)
	case ElemNode:
		err = n.elem.end(n, w, st)
	default:
		// FIXME
		panic(nil)
	}
	if w.Indenter != nil {
		w.last = Event{n.state, n.kind, n.children}
	}
	return err
}
