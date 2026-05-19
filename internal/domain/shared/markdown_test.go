package shared

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatChapterMarkdown_RoundTrip(t *testing.T) {
	title := "黑铁令"
	wc := 236
	twc := 1500
	meta := ChapterMarkdownMetadata{
		BookID: 1, ChapterNo: 3, Stage: "final",
		Title: &title, Status: "approved",
		WordCount: &wc, TargetWordCount: &twc,
		UpdatedAt: "2026-05-18T10:00:00Z",
	}
	summary := "林夜入宗并获得黑铁令"
	content := "黑铁令在他掌心微微震颤。\n\n他抬头望向远山。"
	md := FormatChapterMarkdown(meta, &summary, content)

	parsed, err := ParseChapterMarkdown(md)
	require.NoError(t, err)
	require.Equal(t, int64(1), parsed.Metadata.BookID)
	require.Equal(t, 3, parsed.Metadata.ChapterNo)
	require.Equal(t, "final", parsed.Metadata.Stage)
	require.NotNil(t, parsed.Title)
	require.Equal(t, "黑铁令", *parsed.Title)
	require.NotNil(t, parsed.Summary)
	require.Equal(t, summary, *parsed.Summary)
	require.Equal(t, strings.TrimSpace(content), parsed.Content)
	require.NotNil(t, parsed.Metadata.WordCount)
	require.Equal(t, 236, *parsed.Metadata.WordCount)
}

func TestParseChapterMarkdown_MissingFrontmatter(t *testing.T) {
	_, err := ParseChapterMarkdown("# Title\n\n## Content\n\nbody")
	require.Error(t, err)
	require.Contains(t, err.Error(), "front matter")
}

func TestParseChapterMarkdown_InvalidStage(t *testing.T) {
	raw := "---\nbook_id: 1\nchapter_no: 1\nstage: bogus\nstatus: x\nupdated_at: now\n---\n\n# T\n\n## Content\n\nx\n"
	_, err := ParseChapterMarkdown(raw)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid stage")
}

func TestParseChapterMarkdown_QuotedTitle(t *testing.T) {
	raw := "---\nbook_id: 1\nchapter_no: 1\nstage: draft\ntitle: \"含: 冒号 的标题\"\nstatus: drafted\nupdated_at: 2026-01-01\n---\n\n# 含: 冒号 的标题\n\n## Summary\n\n摘\n\n## Content\n\n正文\n"
	parsed, err := ParseChapterMarkdown(raw)
	require.NoError(t, err)
	require.NotNil(t, parsed.Metadata.Title)
	require.Equal(t, "含: 冒号 的标题", *parsed.Metadata.Title)
}

func TestParseChapterMarkdown_DefaultsAndMissingFields(t *testing.T) {
	raw := "---\nbook_id: 2\nchapter_no: 7\nstage: plan\ntitle: \"原题\"\n---\n\n# 另一个标题\n\n## Summary\n\n\n## Content\n\n正文\n"
	parsed, err := ParseChapterMarkdown(raw)
	require.NoError(t, err)
	require.Equal(t, "unknown", parsed.Metadata.Status)
	require.NotEmpty(t, parsed.Metadata.UpdatedAt)
	require.NotNil(t, parsed.Title)
	require.Equal(t, "另一个标题", *parsed.Title)
	require.Nil(t, parsed.Summary)
}

func TestParseChapterMarkdown_InvalidFrontMatterFields(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "missing colon", raw: "---\nbook_id 1\nchapter_no: 1\nstage: plan\n---\n\n# T\n\n## Content\n\nx\n"},
		{name: "bad book id", raw: "---\nbook_id: nope\nchapter_no: 1\nstage: plan\n---\n\n# T\n\n## Content\n\nx\n"},
		{name: "bad chapter no", raw: "---\nbook_id: 1\nchapter_no: nope\nstage: plan\n---\n\n# T\n\n## Content\n\nx\n"},
		{name: "bad word count", raw: "---\nbook_id: 1\nchapter_no: 1\nstage: plan\nword_count: nope\n---\n\n# T\n\n## Content\n\nx\n"},
		{name: "bad target word count", raw: "---\nbook_id: 1\nchapter_no: 1\nstage: plan\ntarget_word_count: nope\n---\n\n# T\n\n## Content\n\nx\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseChapterMarkdown(tt.raw)
			require.Error(t, err)
		})
	}
}

func TestFormatAndParseChapterMarkdown_EscapesFrontMatter(t *testing.T) {
	title := `含: 引号"和\\反斜杠`
	status := `draft:ready`
	updatedAt := `2026-05-18T10:00:00Z`
	meta := ChapterMarkdownMetadata{
		BookID: 9, ChapterNo: 11, Stage: StageNameDraft,
		Title: &title, Status: status, UpdatedAt: updatedAt,
	}
	md := FormatChapterMarkdown(meta, nil, "正文")
	parsed, err := ParseChapterMarkdown(md)
	require.NoError(t, err)
	require.NotNil(t, parsed.Metadata.Title)
	require.Equal(t, title, *parsed.Metadata.Title)
	require.Equal(t, status, parsed.Metadata.Status)
	require.Equal(t, updatedAt, parsed.Metadata.UpdatedAt)
}
