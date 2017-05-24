package xmlwriter

// Indenter allows custom indenting strategies to be written for pretty
// printing the resultant XML.
//
// Writing one of these little nightmares is not for the faint of heart - it's
// a monumental pain to get indenting rules right. See StandardIndenter for a
// glimpse into the abyss.
//
type Indenter interface {
	// Indent is called every time a node transitions from one State
	// to another. This allows whitespace (or anything, really) to be
	// injected in between any two pieces that the writer writes to the
	// document.
	Indent(w *Writer, last Event, next Event) error

	// Wrap is called on every Text and CommentContent node - if you
	// want your indenter to place text slabs into nicely formatted
	// blocks, the logic goes here.
	Wrap(content string) string
}

type indentLevel struct {
	e       Event
	indents int
}

// StandardIndenter implements a primitive Indenter strategy for pretty
// printing the Writer's XML output.
//
// StandardIndenter is used by the WithIndent writer option:
//	w := xmlwriter.Open(b, xmlwriter.WithIndent())
//
type StandardIndenter struct {
	// Output whitespace control
	IndentString string

	depth int
	stack []indentLevel
}

// NewStandardIndenter creates a StandardIndenter.
func NewStandardIndenter() *StandardIndenter {
	si := &StandardIndenter{
		IndentString: " ",
		stack:        make([]indentLevel, 0, initialNodeDepth),
	}
	si.stack = append(si.stack, indentLevel{})
	return si
}

// Wrap satisfies the Indenter interface.
func (s *StandardIndenter) Wrap(content string) string {
	return content
}

// Indent satisfies the Indenter interface.
func (s *StandardIndenter) Indent(w *Writer, last Event, next Event) error {
	// fmt.Print(next.String())

	isIndenting := (next.Node == ElemNode || next.Node == DTDNode ||
		next.Node == DTDAttListNode)

	isIndented := isIndenting
	isIndentedState := next.State == StateOpen || next.State == StateEnded

	lastIsIndenting := (last.Node == ElemNode || last.Node == DTDNode ||
		last.Node == DTDAttListNode)
	lastIsIndented := lastIsIndenting ||
		last.Node == DTDEntityNode ||
		last.Node == DTDAttrNode ||
		last.Node == DTDElemNode ||
		last.Node == NotationNode ||
		(last.State == StateEnded && last.Node == CommentNode)

	isInline := false
	stackDepth := s.depth

	if isIndenting {
		if next.State == StateOpened {
			s.stack = append(s.stack, indentLevel{e: last})
			s.depth++
		} else if next.State == StateEnded {
			isInline = s.stack[s.depth].indents == 0
			s.depth--
			s.stack = s.stack[:len(s.stack)-1]
		}
	} else if isIndentedState {
		isIndented = next.Node == DTDEntityNode ||
			next.Node == DTDAttrNode ||
			next.Node == DTDElemNode ||
			next.Node == NotationNode ||
			(next.State == StateOpen && next.Node == CommentNode)
	}

	// fmt.Printf(" indent:%d lii:%v lid:%v ii:%v id:%v ", s.depth, lastIsIndenting, lastIsIndented, isIndenting, isIndented)
	// fmt.Printf(" %d %v  ", stackDepth, isInline)

	pairIsIndented := ((lastIsIndenting && isIndented) ||
		(lastIsIndented && isIndenting) ||
		(isIndented && lastIsIndented)) &&
		last.Node != DocNode

	if pairIsIndented && isIndentedState {
		isEmptyElem := (next.Node == ElemNode && next.State == StateEnded && next.Children == 0)
		isInlineCloser := (next.State == StateEnded && isInline)

		if !isEmptyElem && !isInlineCloser {
			if next.State != StateEnded {
				s.stack[stackDepth].indents++
			}
			w.printer.WriteString(w.NewlineString)
			// strings.Join massacres the heap
			for i := 0; i < s.depth; i++ {
				w.printer.WriteString(s.IndentString)
			}
		}

	} else if next.Node == DocNode && next.State == StateEnded {
		w.printer.WriteString(w.NewlineString)
	}

	return w.printer.cachedWriteError()
}
