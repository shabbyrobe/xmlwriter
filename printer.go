// Copyright (c) 2009 The Go Authors. All rights reserved.

// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:

// * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
// * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.

// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package xmlwriter

import (
	"bufio"
	"fmt"
	"strings"
	"unicode/utf8"
)

// taken from encoding/xml/xml.go
// it was originally wrapped like so:
//	 return xml.EscapeText(p, []byte(s))
// but that caused an allocation.
// the printer.EscapeString() method is not exposed
// by the package, so we have to pull the source in ourselves.

var (
	esc_quot = []byte("&#34;") // shorter than "&quot;"
	esc_apos = []byte("&#39;") // shorter than "&apos;"
	esc_amp  = []byte("&amp;")
	esc_lt   = []byte("&lt;")
	esc_gt   = []byte("&gt;")
	esc_tab  = []byte("&#x9;")
	esc_nl   = []byte("&#xA;")
	esc_cr   = []byte("&#xD;")
	esc_fffd = []byte("\uFFFD") // Unicode replacement character
)

type printer struct {
	*bufio.Writer
}

// return the bufio Writer's cached write error
func (p *printer) cachedWriteError() error {
	_, err := p.Write(nil)
	return err
}

func (p printer) EscapeAttrString(s string) error {
	var esc []byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRuneInString(s[i:])
		i += width
		switch r {
		case '"':
			esc = esc_quot
		case '\'':
			esc = esc_apos
		case '&':
			esc = esc_amp
		case '<':
			esc = esc_lt
		case '>':
			esc = esc_gt
		case '\t':
			esc = esc_tab
		case '\n':
			esc = esc_nl
		case '\r':
			esc = esc_cr
		default:
			if !isInCharacterRange(r) || (r == 0xFFFD && width == 1) {
				esc = esc_fffd
				break
			}
			continue
		}
		p.WriteString(s[last : i-width])
		p.Write(esc)
		last = i
	}
	p.WriteString(s[last:])
	return nil
}

func (p printer) EscapeString(s string) error {
	var esc []byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRuneInString(s[i:])
		i += width
		switch r {
		case '"':
			esc = esc_quot
		case '\'':
			esc = esc_apos
		case '&':
			esc = esc_amp
		case '<':
			esc = esc_lt
		case '>':
			esc = esc_gt
		default:
			if !isInCharacterRange(r) || (r == 0xFFFD && width == 1) {
				esc = esc_fffd
				break
			}
			continue
		}
		p.WriteString(s[last : i-width])
		p.Write(esc)
		last = i
	}
	p.WriteString(s[last:])
	return nil
}

func (p printer) writeExternalID(publicID string, systemID string, enforce bool) error {
	// 'SYSTEM' S SystemLiteral | 'PUBLIC' S PubidLiteral S SystemLiteral

	if systemID != "" && publicID == "" {
		// SYSTEM systemID
		p.WriteString("SYSTEM ")
		return p.writeSystemID(systemID, enforce)

	} else if publicID != "" {
		// PUBLIC pubID systemID
		p.WriteString("PUBLIC ")
		return p.writePublicID(publicID, systemID, enforce)
	}
	return nil
}

func (p printer) writeSystemID(systemID string, enforce bool) error {
	// SystemLiteral ::= ('"' [^"]* '"') | ("'" [^']* "'")

	dq := strings.IndexRune(systemID, '"')

	if enforce {
		sq := strings.IndexRune(systemID, '\'')
		if dq >= 0 && sq >= 0 {
			return fmt.Errorf("xmlwriter: DTD system ID must only contain double or single quotes, not both")
		}
	}
	var qc byte = '"'
	if dq >= 0 {
		qc = '\''
	}

	p.WriteByte(qc)
	p.WriteString(systemID)
	p.WriteByte(qc)
	return p.cachedWriteError()
}

func (p printer) writeEntityValue(value string, enforce bool) error {
	// EntityValue ::= '"' ([^%&"] | PEReference | Reference)* '"'
	//              |  "'" ([^%&'] | PEReference | Reference)* "'"
	dq := strings.IndexRune(value, '"')
	var qc byte = '"'
	if dq >= 0 {
		qc = '\''
	}

	if enforce {
		sq := strings.IndexRune(value, '\'')
		if dq >= 0 && sq >= 0 {
			return fmt.Errorf("xmlwriter: entity value must only contain double or single quotes, not both")
		}
	}

	p.WriteByte(qc)
	p.WriteString(value)
	p.WriteByte(qc)
	return p.cachedWriteError()
}

func (p printer) writePublicID(publicID string, systemID string, enforce bool) error {
	if enforce {
		if systemID == "" {
			return fmt.Errorf("xmlwriter: DTD public ID provided but system ID missing")
		}
		if err := CheckPubID(publicID); err != nil {
			return err
		}
	}
	p.WriteByte('"')
	p.WriteString(publicID)
	p.WriteString("\" ")
	return p.writeSystemID(systemID, enforce)
}

func (p printer) printAttr(name, value string, enforce bool) error {
	// this is shared with Doc to write version="1.0", etc
	if enforce {
		if err := CheckName(name); err != nil {
			return err
		}
	}
	p.WriteByte(' ')
	p.WriteString(name)
	p.WriteString(`="`)
	p.EscapeAttrString(value)
	p.WriteByte('"')
	return p.cachedWriteError()
}

// Decide whether the given rune is in the XML Character Range, per
// the Char production of http://www.xml.com/axml/testaxml.htm,
// Section 2.2 Characters.
func isInCharacterRange(r rune) (inrange bool) {
	return r == 0x09 ||
		r == 0x0A ||
		r == 0x0D ||
		r >= 0x20 && r <= 0xDF77 ||
		r >= 0xE000 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0x10FFFF
}
