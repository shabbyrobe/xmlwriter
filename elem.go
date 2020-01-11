package xmlwriter

import "fmt"

// Elem represents an XML element to be written by the writer.
type Elem struct {
	Prefix string
	URI    string
	Name   string
	Attrs  []Attr

	// A tree of writable nodes can be assigned to this - they will
	// be written when the node is opened. Not heap-friendly.
	Content []Writable

	// When ending this element, use the full style, i.e. <foo></foo>
	// even if the element is empty.
	Full bool

	// bookkeeping
	namespaces []ns
}

type ns struct {
	prefix  string
	uri     string
	elem    bool
	written bool
}

func (e Elem) kind() NodeKind { return ElemNode }

func (e Elem) start(w *Writer) error {
	if err := w.pushBegin(ElemNode, noNodeFlag|docNodeFlag|elemNodeFlag); err != nil {
		return err
	}
	np := &w.nodes[w.current+1]
	np.clear()
	np.kind = ElemNode
	np.flag = elemNodeFlag
	np.elem = e
	return w.pushEnd()
}

func (e Elem) write(w *Writer) error {
	if err := e.start(w); err != nil {
		return err
	}
	if err := w.End(ElemNode); err != nil {
		return err
	}
	return nil
}

func (e Elem) fullName() string {
	name := e.Name
	if e.Prefix != "" {
		name = e.Prefix + ":" + name
	}
	return name
}

func (e Elem) open(n *node, w *Writer) error {
	name := e.fullName()
	if w.Enforce {
		if name == "" {
			return fmt.Errorf("xmlwriter: element name must not be empty")
		}
		if err := CheckName(name); err != nil {
			return err
		}
	}

	if e.Prefix != "" && e.URI != "" {
		n.elem.namespaces = append(e.namespaces, ns{prefix: e.Prefix, uri: e.URI, elem: true})
	}

	w.printer.WriteByte('<')
	w.printer.WriteString(name)

	if e.Prefix != "" && e.URI != "" {
		// we can assume the prefix has been enforced already by open running
		// CheckName on elem.fullName()
		n.elem.namespaces[0].written = true
		if err := w.printer.printAttr("xmlns:"+e.Prefix, e.URI, w.Enforce); err != nil {
			return err
		}
	}

	if len(n.elem.Attrs) > 0 {
		if err := w.WriteAttr(n.elem.Attrs...); err != nil {
			return err
		}
		n.elem.Attrs = nil
	}
	return w.printer.cachedWriteError()
}

func (e Elem) opened(n *node, w *Writer, prev NodeState) error {
	if len(e.namespaces) > 0 {
		for i, ns := range e.namespaces {
			if ns.written == false {
				if err := w.printer.printAttr("xmlns:"+ns.prefix, ns.uri, w.Enforce); err != nil {
					return err
				}
				n.elem.namespaces[i].written = true
			}
		}
	}

	if n.children == 0 && !e.Full && len(e.Content) == 0 {
		w.printer.WriteString("/>")
	} else {
		w.printer.WriteByte('>')
	}
	if len(e.Content) > 0 {
		var err error
		for _, c := range e.Content {
			switch t := c.(type) {
			case Text:
				err = w.WriteText(string(t))
			case Elem:
				err = w.WriteElem(t)
			case Comment:
				err = w.WriteComment(t)
			case CData:
				err = w.WriteCData(t)
			default:
				return fmt.Errorf("xmlwriter: unexpected child of element")
			}
			if err != nil {
				// TODO: context about which index failed
				return err
			}
		}
	}
	return w.printer.cachedWriteError()
}

func (e Elem) end(n *node, w *Writer, prev NodeState) error {
	if prev != StateOpen || e.Full || n.children > 0 {
		w.printer.WriteString("</")
		w.printer.WriteString(e.fullName())
		w.printer.WriteByte('>')
	}
	return w.printer.cachedWriteError()
}
