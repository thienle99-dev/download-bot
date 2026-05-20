package ai

import (
	"strings"
	"testing"
)

func TestSanitizeAIContentRemovesCitationMarkers(t *testing.T) {
	input := "Câu trả lời trước \uE200cite\uE202turn0view0\uE201 và sau."

	got := sanitizeAIContent(input)
	want := "Câu trả lời trước  và sau."

	if got != want {
		t.Fatalf("sanitizeAIContent() = %q, want %q", got, want)
	}
}

func TestCitationStreamSanitizerRemovesSplitMarkers(t *testing.T) {
	var parts []string
	sanitizer := newCitationStreamSanitizer(func(token string) {
		parts = append(parts, token)
	})

	for _, token := range []string{
		"Xin chào ",
		"\uE200ci",
		"te\uE202turn0",
		"view0\uE201",
		" bạn.",
	} {
		sanitizer.Write(token)
	}
	sanitizer.Flush()

	got := strings.Join(parts, "")
	want := "Xin chào  bạn."

	if got != want {
		t.Fatalf("stream output = %q, want %q", got, want)
	}
}

func TestCitationStreamSanitizerDropsUnclosedMarkerOnFlush(t *testing.T) {
	var parts []string
	sanitizer := newCitationStreamSanitizer(func(token string) {
		parts = append(parts, token)
	})

	sanitizer.Write("Nội dung sạch ")
	sanitizer.Write("\uE200cite\uE202turn0view0")
	sanitizer.Flush()

	got := strings.Join(parts, "")
	want := "Nội dung sạch "

	if got != want {
		t.Fatalf("stream output = %q, want %q", got, want)
	}
}
