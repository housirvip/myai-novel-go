package shared

import (
	"encoding/json"
	"strings"
	"time"
	"unicode"
)

const TimeLayout = "2006-01-02T15:04:05.000Z"

func NowISO() string {
	return time.Now().UTC().Format(TimeLayout)
}

func MarshalString(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func MarshalStringPtr(v any) *string {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	s := string(b)
	return &s
}

func UnmarshalString(src string, target any) error {
	if strings.TrimSpace(src) == "" {
		return nil
	}
	return json.Unmarshal([]byte(src), target)
}

// EstimateWordCount 估算汉字字数:中文字符算 1,英文按空格切词。
func EstimateWordCount(s string) int {
	if s == "" {
		return 0
	}
	hanCount := 0
	wordCount := 0
	inWord := false
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			hanCount++
			inWord = false
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				wordCount++
				inWord = true
			}
		} else {
			inWord = false
		}
	}
	return hanCount + wordCount
}

func DerefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func StrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func IntPtr(n int) *int { return &n }

func Int64Ptr(n int64) *int64 { return &n }

func DedupeInt64(in []int64) []int64 {
	seen := make(map[int64]bool, len(in))
	out := make([]int64, 0, len(in))
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func TrimSplit(s string, seps string) []string {
	if s == "" {
		return nil
	}
	out := make([]string, 0)
	start := 0
	for i, r := range s {
		if strings.ContainsRune(seps, r) {
			if i > start {
				v := strings.TrimSpace(s[start:i])
				if v != "" {
					out = append(out, v)
				}
			}
			start = i + 1
		}
	}
	if start < len(s) {
		v := strings.TrimSpace(s[start:])
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
