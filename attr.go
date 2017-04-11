package xmlwriter

import (
	"fmt"
	"strconv"
)

type Attr struct {
	Prefix string
	URI    string
	Name   string
	Value  string
}

func (a Attr) writable() {}

func (a Attr) kind() NodeKind { return AttrNode }

func (a Attr) Bool(v bool) Attr    { a.Value = strconv.FormatBool(v); return a }
func (a Attr) Int(v int) Attr      { a.Value = strconv.FormatInt(int64(v), 10); return a }
func (a Attr) Int8(v int8) Attr    { a.Value = strconv.FormatInt(int64(v), 10); return a }
func (a Attr) Int16(v int16) Attr  { a.Value = strconv.FormatInt(int64(v), 10); return a }
func (a Attr) Int32(v int32) Attr  { a.Value = strconv.FormatInt(int64(v), 10); return a }
func (a Attr) Int64(v int64) Attr  { a.Value = strconv.FormatInt(int64(v), 10); return a }
func (a Attr) Uint(v int) Attr     { a.Value = strconv.FormatUint(uint64(v), 10); return a }
func (a Attr) Uint8(v int8) Attr   { a.Value = strconv.FormatUint(uint64(v), 10); return a }
func (a Attr) Uint16(v int16) Attr { a.Value = strconv.FormatUint(uint64(v), 10); return a }
func (a Attr) Uint32(v int32) Attr { a.Value = strconv.FormatUint(uint64(v), 10); return a }
func (a Attr) Uint64(v int64) Attr { a.Value = strconv.FormatUint(uint64(v), 10); return a }
func (a Attr) Float32(v float32) Attr {
	a.Value = strconv.FormatFloat(float64(v), 'g', -1, 32)
	return a
}
func (a Attr) Float64(v float64) Attr { a.Value = strconv.FormatFloat(v, 'g', -1, 64); return a }

func (a Attr) write(w *Writer) error {
	if w.Enforce {
		if err := w.checkParent(NoNode, ElemNode); err != nil {
			return err
		}
	}

	if w.Indenter != nil {
		if err := w.writeIndent(Event{StateOpen, AttrNode, 0}); err != nil {
			return err
		}
	}

	name := a.Name
	if a.Prefix != "" {
		name = a.Prefix + ":" + name
	}

	if a.URI != "" && w.current >= 0 {
		ns := ns{prefix: a.Prefix, uri: a.URI}
		found := false
		fail := false
		for _, existing := range w.nodes[w.current].elem.namespaces {
			if ns.prefix == existing.prefix {
				found = true
				if ns.uri != existing.uri {
					fail = true
				}
				break
			}
		}
		if fail {
			return fmt.Errorf("uri already exists for ns prefix %s", a.Prefix)
		} else if !found {
			w.nodes[w.current].elem.namespaces = append(w.nodes[w.current].elem.namespaces, ns)
		}
	}

	if err := w.printer.printAttr(name, a.Value, w.Enforce); err != nil {
		return err
	}
	if w.Indenter != nil {
		w.last = Event{StateEnded, AttrNode, 0}
	}
	return nil
}
