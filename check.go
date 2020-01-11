package xmlwriter

import (
	"fmt"
)

// CheckEncoding validates the characters in a Doc{} node's encoding="..."
// attribute based on the following production rule:
//	[A-Za-z] ([A-Za-z0-9._] | '-')*
func CheckEncoding(encoding string) error {
	for i, rn := range encoding {
		if (rn >= 'A' && rn <= 'Z') ||
			(rn >= 'a' && rn <= 'z') {
			continue
		}
		if i != 0 {
			if rn == '-' || rn == '.' || rn == '_' ||
				(rn >= '0' && rn <= '9') {
				continue
			}
		}
		return fmt.Errorf("xmlwriter: invalid encoding at position %d: %c", i, rn)
	}
	return nil
}

// CheckName ensures a string satisfies the following production
// rules: https://www.w3.org/TR/xml/#NT-NameStartChar, with the exception
// that it does not return an error on an empty string.
func CheckName(name string) error {
	var start int
	var rn rune
	for start, rn = range name {
		if start == 0 {
			if rn > 0xFFFF || nameChar[uint16(rn)] != 1 {
				return fmt.Errorf("xmlwriter: invalid name at position %d: %c", 0, rn)
			}

		} else {
			break
		}
	}

	for i, rn := range name[start:] {
		if rn > 0xFFFF || nameChar[uint16(rn)] == 0 {
			return fmt.Errorf("xmlwriter: invalid name at position %d: %c", start+i, rn)
		}
	}

	return nil
}

// CheckChars ensures a string contains characters which are valid in a Text{}
// node: https://www.w3.org/TR/xml/#NT-Char
// The 'strict' argument (which xmlwriter should activate by default) ensures
// that unicode characters referenced in the note are also excluded.
func CheckChars(chars string, strict bool) error {
	for i, rn := range chars {
		if rn == 0x9 || rn == 0xA || rn == 0xD ||
			(rn >= 0x20 && rn <= 0xD7FF) ||
			(rn >= 0xE000 && rn <= 0xFFFD) ||
			(rn >= 0x10000 && rn <= 0x10FFFF) {
			continue
		}
		if strict {
			// Document authors are encouraged to avoid "compatibility
			// characters", as defined in section 2.3 of [Unicode]. The
			// characters defined in the following ranges are also discouraged.
			// They are either control characters or permanently undefined
			// Unicode characters:
			if (rn >= 0x7F && rn <= 0x84) || (rn >= 0x86 && rn <= 0x9F) || (rn >= 0xFDD0 && rn <= 0xFDEF) ||
				// FIXME: these are't really ranges, we don't need >= and <=
				(rn >= 0x1FFFE && rn <= 0x1FFFF) || (rn >= 0x2FFFE && rn <= 0x2FFFF) || (rn >= 0x3FFFE && rn <= 0x3FFFF) ||
				(rn >= 0x4FFFE && rn <= 0x4FFFF) || (rn >= 0x5FFFE && rn <= 0x5FFFF) || (rn >= 0x6FFFE && rn <= 0x6FFFF) ||
				(rn >= 0x7FFFE && rn <= 0x7FFFF) || (rn >= 0x8FFFE && rn <= 0x8FFFF) || (rn >= 0x9FFFE && rn <= 0x9FFFF) ||
				(rn >= 0xAFFFE && rn <= 0xAFFFF) || (rn >= 0xBFFFE && rn <= 0xBFFFF) || (rn >= 0xCFFFE && rn <= 0xCFFFF) ||
				(rn >= 0xDFFFE && rn <= 0xDFFFF) || (rn >= 0xEFFFE && rn <= 0xEFFFF) || (rn >= 0xFFFFE && rn <= 0xFFFFF) ||
				(rn >= 0x10FFFE && rn <= 0x10FFFF) {
				continue
			}
		}
		return fmt.Errorf("xmlwriter: invalid chars at position %d: %c", i, rn)
	}
	return nil
}

// CheckPubID validates a string according to the following production rule:
// https://www.w3.org/TR/xml/#NT-PubidLiteral
func CheckPubID(pubid string) error {
	for i, rn := range pubid {
		if rn == 0x20 || rn == 0xD || rn == 0xA || rn == '\'' ||
			rn == '-' || rn == '(' || rn == ')' || rn == '+' ||
			rn == ',' || rn == '.' || rn == '/' || rn == ':' ||
			rn == '=' || rn == '?' || rn == ';' || rn == '!' ||
			rn == '*' || rn == '#' || rn == '@' || rn == '$' ||
			rn == '_' || rn == '%' ||
			(rn >= 'A' && rn <= 'Z') ||
			(rn >= 'a' && rn <= 'z') ||
			(rn >= '0' && rn <= '9') {
			continue
		}
		return fmt.Errorf("xmlwriter: invalid pubid at position %d: %c", i, rn)
	}
	return nil
}
