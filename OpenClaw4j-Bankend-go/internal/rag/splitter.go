package rag

import (
	"strings"
	"unicode/utf8"
)

// SplitText produces stable rune-based chunks. Overlap is clamped to preserve
// forward progress even when a malformed process_config requests a huge overlap.
func SplitText(text string, size, overlap int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}
	if size <= 0 {
		size = 800
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size / 5
	}
	runes := []rune(text)
	result := make([]string, 0, (len(runes)+size-1)/size)
	for start := 0; start < len(runes); {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		value := strings.TrimSpace(string(runes[start:end]))
		if value != "" {
			result = append(result, value)
		}
		if end == len(runes) {
			break
		}
		start = end - overlap
	}
	return result
}

func RuneCount(value string) int { return utf8.RuneCountInString(value) }
