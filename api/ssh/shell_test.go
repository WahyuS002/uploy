package ssh

import "testing"

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty", value: "", want: "''"},
		{name: "whitespace", value: "app service", want: "'app service'"},
		{name: "apostrophe", value: "app's service", want: "'app'\\''s service'"},
		{name: "metacharacters", value: "$(touch /tmp/pwn); $HOME", want: "'$(touch /tmp/pwn); $HOME'"},
		{name: "newline", value: "first\nsecond", want: "'first\nsecond'"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ShellQuote(test.value); got != test.want {
				t.Fatalf("ShellQuote(%q) = %q; want %q", test.value, got, test.want)
			}
		})
	}
}
