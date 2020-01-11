package xmlwriter

import (
	"fmt"
)

// DTD represents a Document Type Definition to be written by the Writer.
type DTD struct {
	Name     string
	PublicID string
	SystemID string
}

func (d DTD) kind() NodeKind { return DTDNode }

func (d DTD) start(w *Writer) error {
	if err := w.pushBegin(DTDNode, noNodeFlag|docNodeFlag); err != nil {
		return err
	}
	np := &w.nodes[w.current+1]
	np.clear()
	np.kind = DTDNode
	np.flag = dtdNodeFlag
	np.dtd = d
	return w.pushEnd()
}

func (d DTD) open(n *node, w *Writer) error {
	if w.Enforce {
		if len(d.Name) == 0 {
			return fmt.Errorf("xmlwriter: DTD name must not be empty")
		}
		if err := CheckName(d.Name); err != nil {
			return err
		}
	}
	w.printer.WriteString("<!DOCTYPE ")
	w.printer.WriteString(d.Name)
	if d.PublicID != "" || d.SystemID != "" {
		w.printer.WriteByte(' ')
		return w.printer.writeExternalID(d.PublicID, d.SystemID, w.Enforce)
	}
	return w.printer.cachedWriteError()
}

func (d DTD) opened(n *node, w *Writer, prev NodeState) error {
	if n.children > 0 {
		w.printer.WriteString(" [")
	}
	return w.printer.cachedWriteError()
}

func (d DTD) end(n *node, w *Writer, prev NodeState) error {
	if n.children > 0 {
		w.printer.WriteString("]>")
	} else {
		w.printer.WriteString(">")
	}
	return w.printer.cachedWriteError()
}

const (
	// DTDElemEmpty is a DTDElem Decl used when the element is empty.
	DTDElemEmpty = "EMPTY"

	// DTDElemAny is a DTDElem Decl used when the element can contain any element.
	DTDElemAny = "ANY"

	// DTDElemPCData is a DTDElem Decl used when the element can contain parsed
	// character data.
	DTDElemPCData = "#PCDATA"
)

// DTDElem represents a DTD element definition to be written by the Writer.
//
// Examples:
//
//	DTDElem{Name: "elem", Decl: DTDElemEmpty} -> <!ELEMENT elem EMPTY>
//	DTDElem{Name: "elem", Decl: "(a|b)*"}     -> <!ELEMENT elem (a|b)*>
//
type DTDElem struct {
	Name string
	Decl string
}

func (d DTDElem) kind() NodeKind { return DTDElemNode }

func (d DTDElem) writable() {}

func (d DTDElem) write(w *Writer) error {
	if w.Enforce {
		if err := w.checkParent(noNodeFlag | dtdNodeFlag); err != nil {
			return err
		}
		if len(d.Name) == 0 {
			return fmt.Errorf("xmlwriter: ELEMENT name must not be empty")
		}
		if len(d.Decl) == 0 {
			return fmt.Errorf("xmlwriter: ELEMENT decl must not be empty")
		}
		if err := CheckName(d.Name); err != nil {
			return err
		}
	}

	if err := w.writeBeginNext(DTDElemNode); err != nil {
		return err
	}
	w.printer.WriteString("<!ELEMENT ")
	w.printer.WriteString(d.Name)
	w.printer.WriteByte(' ')
	w.printer.WriteString(d.Decl)
	w.printer.WriteString(">")
	if w.Indenter != nil {
		w.last = Event{StateEnded, DTDElemNode, 0}
	}
	return w.printer.cachedWriteError()
}

// DTDEntity represents a DTD entity definition to be written by the Writer.
//
// Examples:
//
//	DTDEntity{Name: "pants", Content: "&#62;"}
//	<!ENTITY pants "&#62">
//
//	DTDEntity{Name: "pants", SystemID: "sys"}
//	<!ENTITY pants SYSTEM "sys">
//
//	DTDEntity{Name: "pants", SystemID: "sys", IsPE: true}
//	<!ENTITY % pants SYSTEM "sys">
//
//	DTDEntity{Name: "pants", SystemID: "sys", PublicID: "pub", NDataID: "nd"}
//  <!ENTITY pants PUBLIC "pub" "sys" NDATA nd>
//
type DTDEntity struct {
	Name     string
	Content  string
	IsPE     bool
	PublicID string
	SystemID string
	NDataID  string
}

func (d DTDEntity) kind() NodeKind { return DTDEntityNode }

func (d DTDEntity) writable() {}

func (d DTDEntity) write(w *Writer) error {
	if w.Enforce {
		if len(d.Name) == 0 {
			return fmt.Errorf("xmlwriter: ENTITY name must not be empty")
		}
		if err := CheckName(d.Name); err != nil {
			return err
		}
		if err := w.checkParent(noNodeFlag | dtdNodeFlag); err != nil {
			return err
		}
	}

	if err := w.writeBeginNext(DTDEntityNode); err != nil {
		return err
	}
	w.printer.WriteString("<!ENTITY ")
	if d.IsPE {
		w.printer.WriteString("% ")
	}
	w.printer.WriteString(d.Name)

	if d.SystemID != "" || d.PublicID != "" {
		w.printer.WriteByte(' ')

		// external ref
		if w.Enforce && d.Content != "" {
			return fmt.Errorf("xmlwriter: external ID and content cannot both be provided")
		}
		if err := w.printer.writeExternalID(d.PublicID, d.SystemID, w.Enforce); err != nil {
			return err
		}
		if d.NDataID != "" {
			if !d.IsPE {
				w.printer.WriteString(" NDATA ")
				if w.Enforce {
					if err := CheckName(d.NDataID); err != nil {
						return err
					}
				}
				w.printer.WriteString(d.NDataID)
			} else {
				return fmt.Errorf("xmlwriter: IsPE and NDataID both provided")
			}
		}

	} else {
		// explicit content (parental advisory)
		if w.Enforce && d.NDataID != "" {
			return fmt.Errorf("xmlwriter: external ID required for NDataID")
		}

		w.printer.WriteByte(' ')
		if err := w.printer.writeEntityValue(d.Content, w.Enforce); err != nil {
			return err
		}
	}

	w.printer.WriteString(">")
	if w.Indenter != nil {
		w.last = Event{StateEnded, DTDEntityNode, 0}
	}
	return w.printer.cachedWriteError()
}

// DTDAttList represents a DTD attribute list to be written by the Writer.
//
// Examples:
//
//	DTDAttList{Name: "yep", Attrs: []DTDAttr{
//		{Name: "a1", Type: DTDAttrString, Required: true},
//		{Name: "a2", Type: DTDAttrString, Required: true},
//	}}))
//	<!ATTLIST yep a1 CDATA #REQUIRED a2 CDATA #REQUIRED>
//
type DTDAttList struct {
	Name  string
	Attrs []DTDAttr
}

func (d DTDAttList) start(w *Writer) error {
	if err := w.pushBegin(DTDAttListNode, noNodeFlag|dtdNodeFlag); err != nil {
		return err
	}
	np := &w.nodes[w.current+1]
	np.clear()
	np.kind = DTDAttListNode
	np.flag = dtdAttListNodeFlag
	np.dtdAttList = d
	return w.pushEnd()
}

func (d DTDAttList) kind() NodeKind { return DTDAttListNode }

func (d DTDAttList) write(w *Writer) error {
	if err := w.StartDTDAttList(d); err != nil {
		return err
	}
	if err := w.EndDTDAttList(); err != nil {
		return err
	}
	return nil
}

func (d DTDAttList) open(n *node, w *Writer) error {
	if w.Enforce {
		if len(d.Name) == 0 {
			return fmt.Errorf("xmlwriter: DTD attlist name must not be empty")
		}
		if err := CheckName(d.Name); err != nil {
			return err
		}
	}
	w.printer.WriteString("<!ATTLIST ")
	w.printer.WriteString(d.Name)
	return w.printer.cachedWriteError()
}

func (d DTDAttList) opened(n *node, w *Writer, prev NodeState) error {
	for _, attr := range d.Attrs {
		if err := attr.write(w); err != nil {
			return err
		}
	}
	n.dtdAttList.Attrs = nil
	return nil
}

func (d DTDAttList) end(n *node, w *Writer, prev NodeState) error {
	return w.printer.WriteByte('>')
}

// DTDAttrType constrains the valid values for the Type property of the DTDAttr
// struct.
type DTDAttrType string

const (
	DTDAttrString   DTDAttrType = "CDATA"
	DTDAttrID       DTDAttrType = "ID"
	DTDAttrIDRef    DTDAttrType = "IDREF"
	DTDAttrIDRefs   DTDAttrType = "IDREFS"
	DTDAttrEntity   DTDAttrType = "ENTITY"
	DTDAttrEntities DTDAttrType = "ENTITIES"
	DTDAttrNmtoken  DTDAttrType = "NMTOKEN"
	DTDAttrNmtokens DTDAttrType = "NMTOKENS"
)

// DTDAttrDefaultType represents the possible values from the DefaultDecl production in
// the DTD spec:
// https://www.w3.org/TR/REC-xml/#NT-DefaultDecl
type DTDAttrDefaultType int

const (
	// If Value is empty, this attribute is DTDAttrImplied, otherwise it will be a
	// standard DTDAttrDefault value (without #FIXED). If you need to represent the
	// empty string as your default value, you will need to explicitly use DTDAttrDefault
	// or DTDAttrFixed.
	DTDAttrInfer DTDAttrDefaultType = iota

	// DTDAttr.Value represents the default for this attribute. Value may be an
	// empty string.
	DTDAttrDefault

	// The attribute MUST always be provided. DTDAttr.Value must be empty.
	DTDAttrRequired

	// No default value is provided. DTDAttr.Value must be empty.
	DTDAttrImplied

	// The #FIXED keyword states that the attribute MUST always have the default value.
	// If the declaration is neither #REQUIRED nor #IMPLIED, then the AttValue value
	// contains the declared default value.
	DTDAttrFixed
)

// DTDAttr represents a DTD attribute to be written by the Writer.
type DTDAttr struct {
	Name    string
	Type    DTDAttrType
	Default DTDAttrDefaultType
	Value   string
}

func (d DTDAttr) kind() NodeKind { return DTDAttrNode }

func (d DTDAttr) write(w *Writer) error {
	if w.Enforce {
		if err := w.checkParent(noNodeFlag | dtdAttListNodeFlag); err != nil {
			return err
		}
		if len(d.Name) == 0 {
			return fmt.Errorf("xmlwriter: DTD attr name must not be empty")
		}
		if len(d.Type) == 0 {
			return fmt.Errorf("xmlwriter: DTD attr type must not be empty")
		}
		if err := CheckName(d.Name); err != nil {
			return err
		}
	}

	if err := w.writeBeginNext(DTDAttrNode); err != nil {
		return err
	}

	// HACK: if there are no parents and we are writing these outside an
	// attlist, this leading space will always be present.
	w.printer.WriteByte(' ')

	w.printer.WriteString(d.Name)
	w.printer.WriteByte(' ')
	w.printer.WriteString(string(d.Type))
	w.printer.WriteByte(' ')

	dflt := d.Default
	if dflt == DTDAttrInfer {
		if d.Value != "" {
			dflt = DTDAttrDefault
		} else {
			dflt = DTDAttrImplied
		}
	}

	switch dflt {
	case DTDAttrDefault:
		w.printer.WriteString(`"`)
		w.printer.EscapeAttrString(d.Value)
		w.printer.WriteByte('"')

	case DTDAttrFixed:
		w.printer.WriteString(`#FIXED `)
		w.printer.WriteString(`"`)
		w.printer.EscapeAttrString(d.Value)
		w.printer.WriteByte('"')

	case DTDAttrRequired:
		if d.Value != "" {
			return fmt.Errorf("xmlwriter: #REQUIRED DTD attr must not declare Value")
		}
		w.printer.WriteString("#REQUIRED")

	case DTDAttrImplied:
		if d.Value != "" {
			return fmt.Errorf("xmlwriter: #IMPLIED DTD attr must not declare Value")
		}
		w.printer.WriteString("#IMPLIED")

	default:
		return fmt.Errorf("xmlwriter: unknown DTDAttr default type")
	}

	if w.Indenter != nil {
		w.last = Event{StateEnded, DTDAttrNode, 0}
	}

	return w.printer.cachedWriteError()
}

// Notation represents an XML notation declaration to be written by the Writer.
// https://www.w3.org/TR/xml/#dt-notation
type Notation struct {
	Name     string
	SystemID string
	PublicID string
}

func (n Notation) kind() NodeKind { return NotationNode }

func (n Notation) write(w *Writer) error {
	if w.Enforce {
		if err := w.checkParent(noNodeFlag | dtdNodeFlag); err != nil {
			return err
		}
		if len(n.Name) == 0 {
			return fmt.Errorf("xmlwriter: NOTATION name must not be empty")
		}
		if err := CheckName(n.Name); err != nil {
			return err
		}
		if len(n.PublicID) == 0 && len(n.SystemID) == 0 {
			return fmt.Errorf("xmlwriter: NOTATION requires external ID: '<!NOTATION' S Name S (ExternalID | PublicID) S? '>'")
		}
	}

	if err := w.writeBeginNext(NotationNode); err != nil {
		return err
	}
	w.printer.WriteString("<!NOTATION ")
	w.printer.WriteString(n.Name)
	w.printer.WriteByte(' ')
	if n.SystemID != "" {
		if err := w.printer.writeExternalID(n.PublicID, n.SystemID, w.Enforce); err != nil {
			return err
		}
	} else if n.PublicID != "" {
		if err := w.printer.writePublicID(n.PublicID, n.SystemID, w.Enforce); err != nil {
			return err
		}
	}
	w.printer.WriteString(">")
	if w.Indenter != nil {
		w.last = Event{StateEnded, NotationNode, 0}
	}
	return w.printer.cachedWriteError()
}
