package ssh

import "strings"

// ShellQuote returns value as one POSIX shell argument.
func ShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
