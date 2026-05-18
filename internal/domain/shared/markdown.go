package shared

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ChapterMarkdownMetadata 是导出/导入 markdown 时携带的章节元数据,
// 与原 Node 项目 src/shared/utils/markdown.ts 中 ChapterMarkdownMetadata 字节级对齐。
type ChapterMarkdownMetadata struct {
	BookID          int64
	ChapterNo       int
	Stage           string // plan / draft / review / final
	Title           *string
	Status          string
	WordCount       *int
	TargetWordCount *int
	UpdatedAt       string
}

type ParsedChapterMarkdown struct {
	Metadata ChapterMarkdownMetadata
	Title    *string
	Summary  *string
	Content  string
}

// FormatChapterMarkdown 输出格式与原项目一致:
// frontmatter (key: value) + # Title + ## Summary + ## Content。
func FormatChapterMarkdown(meta ChapterMarkdownMetadata, summary *string, content string) string {
	titleHeader := DerefStr(meta.Title)
	if titleHeader == "" {
		titleHeader = fmt.Sprintf("Chapter %d", meta.ChapterNo)
	}
	wcStr := ""
	if meta.WordCount != nil {
		wcStr = strconv.Itoa(*meta.WordCount)
	}
	twcStr := ""
	if meta.TargetWordCount != nil {
		twcStr = strconv.Itoa(*meta.TargetWordCount)
	}
	summaryStr := ""
	if summary != nil {
		summaryStr = *summary
	}
	lines := []string{
		"---",
		fmt.Sprintf("book_id: %d", meta.BookID),
		fmt.Sprintf("chapter_no: %d", meta.ChapterNo),
		fmt.Sprintf("stage: %s", meta.Stage),
		fmt.Sprintf("title: %s", escapeFrontMatterValue(DerefStr(meta.Title))),
		fmt.Sprintf("status: %s", escapeFrontMatterValue(meta.Status)),
		fmt.Sprintf("word_count: %s", wcStr),
		fmt.Sprintf("target_word_count: %s", twcStr),
		fmt.Sprintf("updated_at: %s", escapeFrontMatterValue(meta.UpdatedAt)),
		"---",
		"",
		"# " + titleHeader,
		"",
		"## Summary",
		"",
		summaryStr,
		"",
		"## Content",
		"",
		strings.TrimRight(content, " \n\t"),
		"",
	}
	return strings.Join(lines, "\n")
}

var (
	frontMatterRe = regexp.MustCompile(`(?s)\A---\n(.*?)\n---\n?`)
	titleRe       = regexp.MustCompile(`(?m)^#\s+(.+)$`)
	summaryRe     = regexp.MustCompile(`(?s)## Summary\s*\n(.*?)\n## Content\s*\n`)
	contentRe     = regexp.MustCompile(`(?s)## Content\s*\n(.*)$`)
)

// ParseChapterMarkdown 严格按导出协议解析,
// 任何缺失/格式错误返回 AppError code=invalid_markdown。
func ParseChapterMarkdown(raw string) (*ParsedChapterMarkdown, error) {
	fmMatch := frontMatterRe.FindStringSubmatchIndex(raw)
	if fmMatch == nil {
		return nil, BadRequest("invalid_markdown: missing front matter")
	}
	frontMatter := raw[fmMatch[2]:fmMatch[3]]
	body := raw[fmMatch[1]:]

	meta, err := parseMetadata(frontMatter)
	if err != nil {
		return nil, err
	}
	contentMatch := contentRe.FindStringSubmatch(body)
	if len(contentMatch) < 2 {
		return nil, BadRequest("invalid_markdown: missing ## Content section")
	}
	titleMatch := titleRe.FindStringSubmatch(body)
	var title *string
	if len(titleMatch) >= 2 {
		t := strings.TrimSpace(titleMatch[1])
		if t != "" {
			title = &t
		}
	}
	if title == nil {
		title = meta.Title
	}
	var summary *string
	if sm := summaryRe.FindStringSubmatch(body); len(sm) >= 2 {
		s := strings.TrimSpace(sm[1])
		if s != "" {
			summary = &s
		}
	}
	return &ParsedChapterMarkdown{
		Metadata: meta,
		Title:    title,
		Summary:  summary,
		Content:  strings.TrimSpace(contentMatch[1]),
	}, nil
}

func parseMetadata(frontMatter string) (ChapterMarkdownMetadata, error) {
	records := map[string]string{}
	for _, line := range strings.Split(frontMatter, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx == -1 {
			return ChapterMarkdownMetadata{}, BadRequest("invalid_markdown: invalid front matter line: " + line)
		}
		key := strings.TrimSpace(line[:idx])
		raw := strings.TrimSpace(line[idx+1:])
		raw = unquote(raw)
		records[key] = raw
	}
	stage := records["stage"]
	switch stage {
	case StageNamePlan, StageNameDraft, StageNameReview, StageNameFinal:
	default:
		return ChapterMarkdownMetadata{}, BadRequest(fmt.Sprintf("invalid_markdown: invalid stage in front matter: %q", stage))
	}
	bookID, err := strconv.ParseInt(records["book_id"], 10, 64)
	if err != nil {
		return ChapterMarkdownMetadata{}, BadRequest("invalid_markdown: invalid book_id in front matter")
	}
	chapterNo, err := strconv.Atoi(records["chapter_no"])
	if err != nil {
		return ChapterMarkdownMetadata{}, BadRequest("invalid_markdown: invalid chapter_no in front matter")
	}
	var title *string
	if v := records["title"]; v != "" {
		t := v
		title = &t
	}
	var wordCount *int
	if v := records["word_count"]; v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return ChapterMarkdownMetadata{}, BadRequest("invalid_markdown: invalid word_count")
		}
		wordCount = &n
	}
	var targetWordCount *int
	if v := records["target_word_count"]; v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return ChapterMarkdownMetadata{}, BadRequest("invalid_markdown: invalid target_word_count")
		}
		targetWordCount = &n
	}
	status := records["status"]
	if status == "" {
		status = "unknown"
	}
	updatedAt := records["updated_at"]
	if updatedAt == "" {
		updatedAt = NowISO()
	}
	return ChapterMarkdownMetadata{
		BookID:          bookID,
		ChapterNo:       chapterNo,
		Stage:           stage,
		Title:           title,
		Status:          status,
		WordCount:       wordCount,
		TargetWordCount: targetWordCount,
		UpdatedAt:       updatedAt,
	}, nil
}

func escapeFrontMatterValue(v string) string {
	if !strings.ContainsAny(v, ":\"") {
		return v
	}
	// 与原项目一致,用 JSON.stringify 等价的引号包裹
	b := strings.ReplaceAll(v, `\`, `\\`)
	b = strings.ReplaceAll(b, `"`, `\"`)
	return `"` + b + `"`
}

func unquote(v string) string {
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		inner := v[1 : len(v)-1]
		inner = strings.ReplaceAll(inner, `\"`, `"`)
		inner = strings.ReplaceAll(inner, `\\`, `\`)
		return inner
	}
	return v
}
