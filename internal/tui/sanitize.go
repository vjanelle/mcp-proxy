package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type logRedactor func(string) string

func noopRedactor(line string) string {
	return line
}

func sanitizeLogLine(line string, redact logRedactor) string {
	activeRedactor := redact
	if activeRedactor == nil {
		activeRedactor = noopRedactor
	}

	return escapeForTerminal(activeRedactor(line))
}

func escapeForTerminal(line string) string {
	var builder strings.Builder
	for idx := 0; idx < len(line); {
		r, size := utf8.DecodeRuneInString(line[idx:])
		if r == utf8.RuneError && size == 1 {
			builder.WriteString(fmt.Sprintf("\\x%02x", line[idx]))
			idx++
			continue
		}

		switch {
		case r == 0x1b || (r >= 0x80 && r <= 0x9f):
			builder.WriteString(fmt.Sprintf("\\x%02x", r))
		case r < 0x20 || r == 0x7f:
			builder.WriteString(fmt.Sprintf("\\u%04x", r))
		default:
			builder.WriteRune(r)
		}

		idx += size
	}

	return builder.String()
}
