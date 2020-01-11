package xmlwriter

import (
	"fmt"
	"strings"
)

// NodeKind is the kind of the node. Yep.
type NodeKind int

// Name returns a stable name for the NodeKind. If the NodeKind is invalid,
// the Name() will be empty. String() returns a human-readable representation
// for information purposes; if a stable string is required, use this instead.
func (n NodeKind) Name() string {
	if int(n) < nodeKindLength {
		return kindName[n]
	}
	return ""
}

// String returns a human-readable representation of the NodeKind. If a stable
// string is required, use Name().
func (n NodeKind) String() string {
	s := n.Name()
	if s == "" {
		s = "<unknown>"
	}
	return fmt.Sprintf("%s(%d)", s, n)
}

func (n NodeKind) flag() nodeFlag {
	return kindFlag[n]
}

// Range of allowed NodeKind values.
const (
	NoNode NodeKind = iota
	AttrNode
	CDataNode
	CDataContentNode
	CommentNode
	CommentContentNode
	DTDNode
	DTDAttListNode
	DTDAttrNode
	DTDElemNode
	DTDEntityNode
	DocNode
	ElemNode
	NotationNode
	PINode
	RawNode
	TextNode

	nodeKindLength int = iota
)

var kindName = [nodeKindLength]string{
	NoNode:             "none",
	AttrNode:           "attr",
	CDataNode:          "cdata",
	CDataContentNode:   "cdatacontent",
	CommentNode:        "comment",
	CommentContentNode: "commentcontent",
	DTDNode:            "dtd",
	DTDAttListNode:     "dtdattlist",
	DTDAttrNode:        "dtdattr",
	DTDElemNode:        "dtdelem",
	DTDEntityNode:      "dtdentity",
	DocNode:            "document",
	ElemNode:           "elem",
	NotationNode:       "notation",
	RawNode:            "raw",
	TextNode:           "text",
}

type nodeFlag int

const (
	noNodeFlag nodeFlag = 1 << iota
	attrNodeFlag
	cDataNodeFlag
	cDataContentNodeFlag
	commentNodeFlag
	commentContentNodeFlag
	dtdNodeFlag
	dtdAttListNodeFlag
	dtdAttrNodeFlag
	dtdElemNodeFlag
	dtdEntityNodeFlag
	docNodeFlag
	elemNodeFlag
	notationNodeFlag
	pINodeFlag
	rawNodeFlag
	textNodeFlag
)

var kindFlag = [nodeKindLength]nodeFlag{
	NoNode:             noNodeFlag,
	AttrNode:           attrNodeFlag,
	CDataNode:          cDataNodeFlag,
	CDataContentNode:   cDataContentNodeFlag,
	CommentNode:        commentNodeFlag,
	CommentContentNode: commentContentNodeFlag,
	DTDNode:            dtdNodeFlag,
	DTDAttListNode:     dtdAttListNodeFlag,
	DTDAttrNode:        dtdAttrNodeFlag,
	DTDElemNode:        dtdElemNodeFlag,
	DTDEntityNode:      dtdEntityNodeFlag,
	DocNode:            docNodeFlag,
	ElemNode:           elemNodeFlag,
	NotationNode:       notationNodeFlag,
	RawNode:            rawNodeFlag,
	TextNode:           textNodeFlag,
}

func (set nodeFlag) names() string {
	switch set {
	case noNodeFlag:
		return "none"
	case noNodeFlag | cDataNodeFlag:
		return "none, cdata"
	case noNodeFlag | commentNodeFlag:
		return "none, comment"
	case noNodeFlag | docNodeFlag:
		return "none, document"
	case noNodeFlag | dtdNodeFlag:
		return "none, dtd"
	case noNodeFlag | dtdAttListNodeFlag:
		return "none, dtdattlist"
	case noNodeFlag | elemNodeFlag:
		return "none, elem"
	case noNodeFlag | docNodeFlag | dtdNodeFlag | elemNodeFlag:
		return "none, document, dtd, elem"
	case noNodeFlag | docNodeFlag | elemNodeFlag:
		return "none, document, elem"

	default:
		var names = make([]string, 0, 4)
		for i := 0; i < nodeKindLength; i++ {
			nk := NodeKind(i)

			if set&nk.flag() != 0 {
				names = append(names, nk.Name())
			}
		}
		out := strings.Join(names, ", ")
		// panic(out)
		return out
	}
}
