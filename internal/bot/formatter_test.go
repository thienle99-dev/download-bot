package bot

import (
	"testing"
)

func TestFormatMarkdownToTelegramHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Plain text",
			input:    "Hello World",
			expected: "<blockquote>Hello World</blockquote>",
		},
		{
			name:     "Bold and Italic",
			input:    "This is **bold** and *italic* text.",
			expected: "<blockquote>This is <b>bold</b> and <i>italic</i> text.</blockquote>",
		},
		{
			name:     "Inline code",
			input:    "Use `go test` to run tests.",
			expected: "<blockquote>Use <code>go test</code> to run tests.</blockquote>",
		},
		{
			name:     "Markdown Link",
			input:    "Visit [Google](https://google.com) now.",
			expected: `<blockquote>Visit <a href="https://google.com">Google</a> now.</blockquote>`,
		},
		{
			name:     "Simple Code block",
			input:    "Here is code:\n```\nfmt.Println(\"test\")\n```\nDone.",
			expected: "<blockquote>Here is code:\n</blockquote><pre><code>fmt.Println(\"test\")</code></pre><blockquote>\nDone.</blockquote>",
		},
		{
			name:     "SQL Code block with language prefix",
			input:    "Run this:\n```sql\nSELECT * FROM users WHERE id = 1;\n```\nOK?",
			expected: "<blockquote>Run this:\n</blockquote><pre><code class=" + `"language-sql"` + ">SELECT * FROM users WHERE id = 1;</code></pre><blockquote>\nOK?</blockquote>",
		},
		{
			name:     "HTML characters escaping",
			input:    "AI says 1 < 2 && 3 > 2.",
			expected: "<blockquote>AI says 1 &lt; 2 &amp;&amp; 3 &gt; 2.</blockquote>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := FormatMarkdownToTelegramHTML(tt.input)
			if actual != tt.expected {
				t.Errorf("expected:\n%s\nactual:\n%s", tt.expected, actual)
			}
		})
	}
}
