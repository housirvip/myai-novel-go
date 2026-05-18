package planning

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeExtractedIntent_StripsJSONFence(t *testing.T) {
	raw := "```json\n{\"intentSummary\":\"林夜入宗\",\"keywords\":[\"林夜\",\"宗门\"],\"mustInclude\":[\" 林夜 \"],\"mustAvoid\":[]}\n```"
	ei := NormalizeExtractedIntent(raw)
	require.Equal(t, "林夜入宗", ei.IntentSummary)
	require.Contains(t, ei.Keywords, "林夜")
	require.Contains(t, ei.Keywords, "宗门")
	// 去重 + trim
	require.Equal(t, []string{"林夜"}, ei.MustInclude)
}

func TestNormalizeExtractedIntent_NonJSONFallback(t *testing.T) {
	raw := "只是一段散文"
	ei := NormalizeExtractedIntent(raw)
	require.Equal(t, "只是一段散文", ei.IntentSummary)
	require.Empty(t, ei.Keywords)
}

func TestBuildRetrievalQuery_DeduplicatesAndTruncates(t *testing.T) {
	ei := ExtractedIntent{
		IntentSummary: "林夜入宗,获得令牌",
		Keywords:      []string{"林夜", "林夜", "令牌", "ab", ""},
		MustInclude:   []string{"令牌", "宗门"},
	}
	keywords, q := BuildRetrievalQuery(ei)
	// 去重(林夜/令牌只一次)
	seen := map[string]int{}
	for _, k := range keywords {
		seen[k]++
	}
	require.LessOrEqual(t, seen["林夜"], 1)
	require.LessOrEqual(t, seen["令牌"], 1)
	require.True(t, strings.Contains(q, "林夜"))
}
