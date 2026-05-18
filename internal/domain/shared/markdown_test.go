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
