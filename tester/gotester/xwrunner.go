package main

// this should do a better job of sanity checking the script - it's
// too hard in the C version to get it nice.

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"

	xw "github.com/shabbyrobe/xmlwriter"
)

const (
	kindAll          = "all"
	kindAttr         = "attr"
	kindCData        = "cdata"
	kindCDataContent = "cdata-content"
	kindComment      = "comment"
	kindDoc          = "doc"
	kindDTD          = "dtd"
	kindDTDAttr      = "dtd-attr"
	kindDTDAttList   = "dtd-att-list"
	kindDTDElem      = "dtd-elem"
	kindDTDEntity    = "dtd-entity"
	kindElem         = "elem"
	kindNotation     = "notation"
	kindPI           = "pi"
	kindRaw          = "raw"
	kindText         = "text"
)

var wsStrip = regexp.MustCompile(`[\n\r\t ]+`)

// Script represents a gotester script.
type Script struct {
	XMLName  xml.Name  `xml:"script"`
	Name     string    `xml:"string"`
	Enforce  *bool     `xml:"enforce"`
	Commands []Command `xml:"command"`
}

// Command represents a gotester command.
type Command struct {
	XMLName xml.Name   `xml:"command"`
	Action  string     `xml:"action,attr"`
	Kind    string     `xml:"kind,attr"`
	Content string     `xml:",chardata"`
	Name    string     `xml:"name,attr"`
	WS      string     `xml:"ws,attr"`
	Params  []xml.Attr `xml:",any,attr"`
}

// ErrUnknownParam represents an unknown parameter error.
type ErrUnknownParam struct {
	Action string
	Kind   string
	Name   string
}

// NewErrUnknownParam makes an unknown param error.
func NewErrUnknownParam(command Command, name string) ErrUnknownParam {
	return ErrUnknownParam{command.Action, command.Kind, name}
}

// Error implements error.
func (e ErrUnknownParam) Error() string {
	return fmt.Sprintf("unknown param %s in command %s:%s", e.Name, e.Action, e.Kind)
}

// CleanContent cleans content.
func (c Command) CleanContent() string {
	r := c.Content
	if c.WS == "strip" {
		r = wsStrip.ReplaceAllString(strings.TrimSpace(r), " ")
	}
	return r
}

// XWRunner is an xwrunner.
type XWRunner struct {
	writer  io.Writer
	xwriter *xw.Writer
	options []xw.Option

	active bool
}

// WriterConfig configures the xmlwriter.Writer.
type WriterConfig struct {
	Enforce       bool
	StrictChars   bool
	Indent        bool
	IndentString  string
	NewlineString string
}

func (r *XWRunner) activate(enc *string) error {
	ev := "UTF-8"
	if enc != nil {
		ev = strings.ToUpper(*enc)
	}
	if ev == "UTF-8" {
		r.xwriter = xw.Open(r.writer, r.options...)
	} else {
		var enc *encoding.Encoder
		switch ev {
		case "ISO-8859-1":
			enc = charmap.ISO8859_1.NewEncoder()
		case "WINDOWS-1252":
			enc = charmap.Windows1252.NewEncoder()
		default:
			return fmt.Errorf("unsupported encoding %s", ev)
		}
		r.xwriter = xw.OpenEncoding(r.writer, ev, enc, r.options...)
	}
	r.active = true
	return nil
}

var writers = map[string]func(r *XWRunner, command Command) error{
	kindAttr: func(r *XWRunner, command Command) error {
		attr := xw.Attr{Name: command.Name, Value: command.CleanContent()}
		for _, p := range command.Params {
			switch p.Name.Local {
			case "prefix":
				attr.Prefix = p.Value
			case "uri":
				attr.URI = p.Value
			default:
				return NewErrUnknownParam(command, p.Name.Local)
			}
		}
		return r.xwriter.WriteAttr(attr)
	},
	kindCData: func(r *XWRunner, command Command) error {
		cdata := xw.CData{Content: command.CleanContent()}
		if len(command.Params) > 0 {
			return fmt.Errorf("unknown params")
		}
		return r.xwriter.WriteCData(cdata)
	},
	kindCDataContent: func(r *XWRunner, command Command) error {
		if len(command.Params) > 0 {
			return fmt.Errorf("unknown params")
		}
		return r.xwriter.WriteCDataContent(command.CleanContent())
	},
	kindComment: func(r *XWRunner, command Command) error {
		comment := xw.Comment{Content: command.CleanContent()}
		if len(command.Params) > 0 {
			return fmt.Errorf("unknown params")
		}
		return r.xwriter.WriteComment(comment)
	},
	kindDTDAttr: func(r *XWRunner, command Command) error {
		attr := xw.DTDAttr{
			Name: command.Name,
		}
		for _, p := range command.Params {
			switch p.Name.Local {
			case "decl":
				attr.Decl = p.Value
			case "type":
				// TODO: validate
				attr.Type = xw.DTDAttrType(p.Value)
			case "required":
				attr.Required = strings.ToLower(p.Value) == "true"
			default:
				return NewErrUnknownParam(command, p.Name.Local)
			}
		}
		return r.xwriter.WriteDTDAttr(attr)
	},
	kindDTDElem: func(r *XWRunner, command Command) error {
		if len(command.Params) > 0 {
			return fmt.Errorf("unknown params for DTD element")
		}
		elem := xw.DTDElem{
			Name: command.Name,
			Decl: command.CleanContent(),
		}
		return r.xwriter.WriteDTDElem(elem)
	},
	kindDTDEntity: func(r *XWRunner, command Command) error {
		entity := xw.DTDEntity{
			Name:    command.Name,
			Content: command.CleanContent(),
		}
		for _, p := range command.Params {
			switch p.Name.Local {
			case "is-pe":
				entity.IsPE = strings.ToLower(p.Value) == "true"
			case "ndata-id":
				entity.NDataID = p.Value
			case "system-id":
				entity.SystemID = p.Value
			case "public-id":
				entity.PublicID = p.Value
			default:
				return NewErrUnknownParam(command, p.Name.Local)
			}
		}
		return r.xwriter.WriteDTDEntity(entity)
	},
	kindNotation: func(r *XWRunner, command Command) error {
		notation := xw.Notation{
			Name: command.Name,
		}
		for _, p := range command.Params {
			switch p.Name.Local {
			case "system-id":
				notation.SystemID = p.Value
			case "public-id":
				notation.PublicID = p.Value
			default:
				return NewErrUnknownParam(command, p.Name.Local)
			}
		}
		return r.xwriter.WriteNotation(notation)
	},
	kindPI: func(r *XWRunner, command Command) error {
		target := ""
		for _, p := range command.Params {
			switch p.Name.Local {
			case "target":
				target = p.Value
			default:
				return NewErrUnknownParam(command, p.Name.Local)
			}
		}
		return r.xwriter.WritePI(xw.PI{Target: target, Content: command.CleanContent()})
	},
	kindRaw: func(r *XWRunner, command Command) error {
		// default mode should be to 'next' - this is for compat with the ctester.
		next := true

		for _, p := range command.Params {
			switch p.Name.Local {
			case "next":
				l := strings.ToLower(p.Value)
				if l != "yes" && l != "no" && l != "true" && l != "false" {
					return fmt.Errorf("invalid boolean value")
				}
				next = l == "yes" || l == "true"
			default:
				return NewErrUnknownParam(command, p.Name.Local)
			}
		}
		if next {
			if err := r.xwriter.Next(); err != nil {
				return err
			}
		}
		return r.xwriter.WriteRaw(command.CleanContent())
	},
	kindText: func(r *XWRunner, command Command) error {
		if len(command.Params) > 0 {
			return fmt.Errorf("unknown params for text")
		}
		return r.xwriter.WriteText(command.CleanContent())
	},
}

func (r *XWRunner) doWrite(command Command) error {
	if command.Kind != kindDoc && !r.active {
		if err := r.activate(nil); err != nil {
			return err
		}
	}
	h, ok := writers[command.Kind]
	if !ok {
		spew.Dump(command)
		return fmt.Errorf("unknown kind %s", command.Kind)
	}
	return h(r, command)
}

var starters = map[string]func(r *XWRunner, command Command) error{
	kindCData: func(r *XWRunner, command Command) error {
		if len(command.Params) > 0 {
			return fmt.Errorf("unknown params for cdata")
		}
		return r.xwriter.StartCData(xw.CData{})
	},
	kindComment: func(r *XWRunner, command Command) error {
		if len(command.Params) > 0 {
			return fmt.Errorf("unknown params for comment")
		}
		return r.xwriter.StartComment(xw.Comment{})
	},
	kindDoc: func(r *XWRunner, command Command) error {
		doc := xw.Doc{SuppressEncoding: true}
		for _, p := range command.Params {
			switch p.Name.Local {
			case "version":
				v := p.Value
				doc.ForcedVersion = &v
				doc.SuppressVersion = false
			case "encoding":
				v := p.Value
				doc.ForcedEncoding = &v
				doc.SuppressEncoding = false
			case "standalone":
				l := strings.ToLower(p.Value)
				if l != "yes" && l != "no" && l != "true" && l != "false" {
					return fmt.Errorf("invalid boolean value")
				}
				yep := (l == "yes" || l == "true")
				doc.Standalone = &yep
			default:
				return NewErrUnknownParam(command, p.Name.Local)
			}
		}
		if !r.active {
			if err := r.activate(doc.ForcedEncoding); err != nil {
				return err
			}
		}

		return r.xwriter.StartDoc(doc)
	},
	kindDTD: func(r *XWRunner, command Command) error {
		dtd := xw.DTD{Name: command.Name}
		for _, p := range command.Params {
			switch p.Name.Local {
			case "public-id":
				dtd.PublicID = p.Value
			case "system-id":
				dtd.SystemID = p.Value
			default:
				return fmt.Errorf("unknown dtd param")
			}
		}
		return r.xwriter.StartDTD(dtd)
	},
	kindDTDAttList: func(r *XWRunner, command Command) error {
		if len(command.Params) > 0 {
			return fmt.Errorf("unknown params dtd attlist")
		}
		al := xw.DTDAttList{Name: command.Name}
		return r.xwriter.StartDTDAttList(al)
	},
	kindElem: func(r *XWRunner, command Command) error {
		elem := xw.Elem{Name: command.Name}
		for _, p := range command.Params {
			switch p.Name.Local {
			case "uri":
				elem.URI = p.Value
			case "prefix":
				elem.Prefix = p.Value
			default:
				return fmt.Errorf("unknown element param")
			}
		}
		return r.xwriter.StartElem(elem)
	},
}

func (r *XWRunner) doStart(command Command) error {
	if command.Kind != kindDoc && !r.active {
		if err := r.activate(nil); err != nil {
			return err
		}
	}
	h, ok := starters[command.Kind]
	if !ok {
		spew.Dump(command)
		return fmt.Errorf("unknown kind %s", command.Kind)
	}
	return h(r, command)
}

var enders = map[string]func(r *XWRunner, command Command) error{
	kindAll: func(r *XWRunner, command Command) error {
		if len(command.Params) > 0 {
			return fmt.Errorf("unknown params")
		}
		return r.xwriter.EndAll()
	},
	kindCData: func(r *XWRunner, command Command) error {
		if len(command.Params) > 0 {
			return fmt.Errorf("unknown params")
		}
		return r.xwriter.EndCData()
	},
	kindComment: func(r *XWRunner, command Command) error {
		if len(command.Params) > 0 {
			return fmt.Errorf("unknown params")
		}
		return r.xwriter.EndComment()
	},
	kindDoc: func(r *XWRunner, command Command) error {
		if len(command.Params) > 0 {
			return fmt.Errorf("unknown params")
		}
		return r.xwriter.EndDoc()
	},
	kindDTD: func(r *XWRunner, command Command) error {
		if len(command.Params) > 0 {
			return fmt.Errorf("unknown params")
		}
		return r.xwriter.EndDTD()
	},
	kindDTDAttList: func(r *XWRunner, command Command) error {
		if len(command.Params) > 0 {
			return fmt.Errorf("unknown params")
		}
		return r.xwriter.EndDTDAttList()
	},
	kindElem: func(r *XWRunner, command Command) error {
		full := false
		for _, param := range command.Params {
			switch param.Name.Local {
			case "full":
				full = strings.ToLower(param.Value) == "true"
			default:
				return fmt.Errorf("unknown end element param")
			}
		}
		var err error
		if full {
			err = r.xwriter.EndElemFull()
		} else {
			err = r.xwriter.EndElem()
		}
		return err
	},
}

func (r *XWRunner) doEnd(command Command) error {
	if !r.active {
		return fmt.Errorf("xwrunner: not active")
	}
	h, ok := enders[command.Kind]
	if !ok {
		spew.Dump(command)
		return fmt.Errorf("unknown kind %s", command.Kind)
	}
	return h(r, command)
}

func (r *XWRunner) flush() error {
	if !r.active {
		return fmt.Errorf("xwrunner: not active")
	}
	err := r.xwriter.Flush()
	if err != nil {
		return err
	}
	r.xwriter = nil
	r.active = false
	return nil
}

// NewWriter creates a new xmlwriter.Writer for the script.
func (s *Script) NewWriter(b io.Writer, options ...xw.Option) *xw.Writer {
	w := xw.Open(b, options...)
	if s.Enforce != nil {
		w.Enforce = *s.Enforce
	}
	return w
}

// Run runs the script.
func (s *Script) Run(b io.Writer, options ...xw.Option) (err error) {
	xw := &XWRunner{writer: b, options: options}
	if err = s.Exec(xw); err != nil {
		return
	}
	if err = xw.flush(); err != nil {
		return
	}
	return nil
}

// Exec executes the script.
func (s *Script) Exec(xw *XWRunner) (err error) {
	for _, command := range s.Commands {

		// remove debugging attributes from all commands
		ps := make([]xml.Attr, 0, len(command.Params))
		for _, p := range command.Params {
			if p.Name.Local != "line" && p.Name.Local != "pos" && p.Name.Local != "fn" {
				ps = append(ps, p)
			}
		}
		command.Params = ps

		switch command.Action {
		case "write":
			if err = xw.doWrite(command); err != nil {
				return
			}
		case "start":
			if err = xw.doStart(command); err != nil {
				return
			}
		case "end":
			if err = xw.doEnd(command); err != nil {
				return
			}
		default:
			return fmt.Errorf("")
		}
	}
	return nil
}
