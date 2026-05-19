package bot

import (
	"regexp"
	"strings"
)

var (
	boldRegex   = regexp.MustCompile(`\*\*(.*?)\*\*`)
	italicRegex = regexp.MustCompile(`\*(.*?)\*`)
	linkRegex   = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)
)

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// FormatMarkdownToTelegramHTML converts basic Markdown (bold, italic, code blocks, links)
// into Telegram-compatible HTML. It ensures safety by escaping HTML entities first.
func FormatMarkdownToTelegramHTML(raw string) string {
	if raw == "" {
		return ""
	}

	// Step 1: Split into code blocks and normal text blocks using "```"
	segments := strings.Split(raw, "```")
	var result []string

	for i, seg := range segments {
		// Even index means normal text, odd index means code block
		if i%2 == 0 {
			if strings.TrimSpace(seg) == "" {
				// Keep linebreaks/whitespace but don't wrap empty blocks in blockquote
				result = append(result, escapeHTML(seg))
				continue
			}

			// Escape HTML first to prevent raw text from breaking Telegram parser
			escapedText := escapeHTML(seg)

			// Format inline code: `code` -> <code>code</code>
			// Splitting by ` is safer than regex for simple inline code
			inlineCodeParts := strings.Split(escapedText, "`")
			for j, part := range inlineCodeParts {
				if j%2 == 1 {
					inlineCodeParts[j] = "<code>" + part + "</code>"
				}
			}
			escapedText = strings.Join(inlineCodeParts, "")

			// Format bold: **text** -> <b>text</b>
			escapedText = boldRegex.ReplaceAllString(escapedText, "<b>$1</b>")

			// Format italic: *text* -> <i>text</i>
			escapedText = italicRegex.ReplaceAllString(escapedText, "<i>$1</i>")

			// Format Markdown link: [text](url) -> <a href="url">text</a>
			escapedText = linkRegex.ReplaceAllString(escapedText, `<a href="$2">$1</a>`)

			// Wrap the formatted text block in a blockquote
			result = append(result, "<blockquote>"+escapedText+"</blockquote>")
		} else {
			// Code block
			codeLines := strings.Split(seg, "\n")
			var lang string
			var codeContent string

			if len(codeLines) > 0 {
				firstLine := strings.TrimSpace(codeLines[0])
				// Check if the first line indicates a language (like sql, go, html)
				if len(firstLine) > 0 && len(firstLine) < 15 && !strings.Contains(firstLine, " ") {
					lang = firstLine
					codeContent = strings.Join(codeLines[1:], "\n")
				} else {
					codeContent = strings.Join(codeLines, "\n")
				}
			} else {
				codeContent = seg
			}

			// Clean starting and ending newlines in code blocks
			codeContent = strings.TrimPrefix(codeContent, "\n")
			codeContent = strings.TrimSuffix(codeContent, "\n")

			// Escape the code contents
			escapedCode := escapeHTML(codeContent)

			if lang != "" {
				result = append(result, `<pre><code class="language-`+lang+`">`+escapedCode+`</code></pre>`)
			} else {
				result = append(result, `<pre><code>`+escapedCode+`</code></pre>`)
			}
		}
	}

	return strings.Join(result, "")
}
