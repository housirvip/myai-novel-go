package planning

import (
	"encoding/json"
	"strings"
)

// NormalizeExtractedIntent 把 LLM 返回的 JSON 文本(可能带 markdown 围栏)解析成 ExtractedIntent。
func NormalizeExtractedIntent(raw string) ExtractedIntent {
	cleaned := stripJSONFence(raw)
	var ei ExtractedIntent
	if err := json.Unmarshal([]byte(cleaned), &ei); err != nil {
		return ExtractedIntent{IntentSummary: strings.TrimSpace(raw)}
	}
	ei.Keywords = dedupeStrings(ei.Keywords)
	ei.MustInclude = dedupeStrings(ei.MustInclude)
	ei.MustAvoid = dedupeStrings(ei.MustAvoid)
	return ei
}

func BuildRetrievalQuery(intent ExtractedIntent) ([]string, string) {
	terms := append([]string{}, intent.Keywords...)
	terms = append(terms, intent.MustInclude...)
	terms = append(terms, splitTerms(intent.IntentSummary)...)
	terms = append(terms, intent.ContinuityCues...)
	terms = append(terms, intent.SettingCues...)
	terms = append(terms, intent.SceneCues...)
	for _, group := range [][]string{intent.EntityHints.Characters, intent.EntityHints.Factions, intent.EntityHints.Items, intent.EntityHints.Hooks, intent.EntityHints.WorldSettings, intent.EntityHints.Relations} {
		terms = append(terms, group...)
	}
	keywords := []string{}
	seen := map[string]bool{}
	for _, t := range terms {
		t = strings.TrimSpace(t)
		if t == "" || len(t) > 16 || seen[t] {
			continue
		}
		seen[t] = true
		keywords = append(keywords, t)
	}
	queryText := intent.IntentSummary
	if len(keywords) > 0 {
		queryText += " " + strings.Join(keywords, " ")
	}
	return keywords, strings.TrimSpace(queryText)
}

func splitTerms(s string) []string {
	if s == "" {
		return nil
	}
	out := make([]string, 0)
	cur := strings.Builder{}
	for _, r := range s {
		if r == ' ' || r == ',' || r == '，' || r == '、' || r == ';' || r == '；' || r == '。' || r == '.' || r == '!' || r == '?' || r == '\n' {
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

func dedupeStrings(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func stripJSONFence(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
