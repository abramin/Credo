package email

import (
	"strings"
	"unicode"
)

func DeriveNameFromEmail(email string) (string, string) {
	localPart := email
	if at := strings.IndexByte(email, '@'); at > 0 {
		localPart = email[:at]
	}

	parts := strings.FieldsFunc(localPart, func(r rune) bool {
		return r == '.' || r == '_' || r == '-' || r == '+'
	})

	if len(parts) == 0 {
		return "User", "User"
	}

	first := capitalize(parts[0])
	last := "User"
	if len(parts) > 1 {
		last = capitalize(parts[len(parts)-1])
	}

	return first, last
}

func capitalize(s string) string {
	if s == "" {
		return s
	}

	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
