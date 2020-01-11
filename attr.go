package xmlwriter

import (
	"fmt"
	"strconv"
)

// Attr represents an XML attribute to be written by the Writer.
type Attr struct {
	Prefix string
	URI    string
	Name   string
	Value  string
}

func (a Attr) writable() {}

func (a Attr) kind() NodeKind { return AttrNode }

// Bool writes a boolean to an attribute.
func (a Attr) Bool(v bool) Attr { a.Value = strconv.FormatBool(v); return a }

// Int writes an int to an attribute.
func (a Attr) Int(v int) Attr { a.Value = strconv.FormatInt(int64(v), 10); return a }

// Int8 writes an int8 to an attribute.
func (a Attr) Int8(v int8) Attr { a.Value = strconv.FormatInt(int64(v), 10); return a }

// Int16 writes an int16 to an attribute.
func (a Attr) Int16(v int16) Attr { a.Value = strconv.FormatInt(int64(v), 10); return a }

// Int32 writes an int32 to an attribute.
func (a Attr) Int32(v int32) Attr { a.Value = strconv.FormatInt(int64(v), 10); return a }

// Int64 writes an int64 to an attribute.
func (a Attr) Int64(v int64) Attr { a.Value = strconv.FormatInt(int64(v), 10); return a }

// Uint writes a uint to an attribute.
func (a Attr) Uint(v int) Attr { a.Value = strconv.FormatUint(uint64(v), 10); return a }

// Uint8 writes a uint8 to an attribute.
func (a Attr) Uint8(v uint8) Attr { a.Value = strconv.FormatUint(uint64(v), 10); return a }

// Uint16 writes a uint16 to an attribute.
func (a Attr) Uint16(v uint16) Attr { a.Value = strconv.FormatUint(uint64(v), 10); return a }

// Uint32 writes a uint32 to an attribute.
func (a Attr) Uint32(v uint32) Attr { a.Value = strconv.FormatUint(uint64(v), 10); return a }

// Uint64 writes a uint64 to an attribute.
func (a Attr) Uint64(v uint64) Attr { a.Value = strconv.FormatUint(uint64(v), 10); return a }

// Float32 writes a float32 to an attribute.
func (a Attr) Float32(v float32) Attr {
	a.Value = strconv.FormatFloat(float64(v), 'g', -1, 32)
	return a
}

// Float64 writes a float64 to an attribute.
func (a Attr) Float64(v float64) Attr { a.Value = strconv.FormatFloat(v, 'g', -1, 64); return a }

func (a Attr) write(w *Writer) error {
	if w.Enforce {
		if err := w.checkParent(noNodeFlag | elemNodeFlag); err != nil {
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
