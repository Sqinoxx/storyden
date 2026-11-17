package tag_suggest

import (
	"context"
	"database/sql"
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/philippgille/chromem-go"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/services/summary_text"
)

// TestCalibrate measures the learned tag suggestion leave-one-out against a
// copy of real data, to re-tune the thresholds after changing the embedding
// model. It only runs when STORYDEN_CALIBRATE_DSN (postgres) and
// STORYDEN_CALIBRATE_OPENAI_URL (an OpenAI-compatible server such as Ollama)
// are set; STORYDEN_CALIBRATE_MODEL defaults to bge-m3.
func TestCalibrate(t *testing.T) {
	dsn, url := os.Getenv("STORYDEN_CALIBRATE_DSN"), os.Getenv("STORYDEN_CALIBRATE_OPENAI_URL")
	if dsn == "" || url == "" {
		t.Skip("calibration data not configured")
	}
	model := os.Getenv("STORYDEN_CALIBRATE_MODEL")
	if model == "" {
		model = "bge-m3"
	}

	ctx := context.Background()
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()

	threads := loadThreads(t, ctx, db)

	embed := chromem.NewEmbeddingFuncOpenAICompat(url, "ollama", model, nil)
	for _, th := range threads {
		th.query, err = embed(ctx, th.title+"\n\n"+summary_text.Clip(th.body, maxInputChars))
		require.NoError(t, err)
		th.doc, err = embed(ctx, th.title+"\n\n"+th.body)
		require.NoError(t, err)
	}

	t.Logf("%d published threads, current thresholds: sim>=%.2f share>=%.1f", len(threads), minNeighbourSimilarity, minVoteShare)
	for _, minSim := range []float64{0.5, 0.55, 0.6, 0.65, 0.7} {
		for _, share := range []float64{0.5, 0.6, 0.7, 0.8} {
			r := leaveOneOut(threads, minSim, share)
			t.Logf("sim>=%.2f share>=%.1f: precision %.0f%% (%d/%d), threads with a suggestion %d, recall %.0f%%",
				minSim, share, 100*r.precision(), r.correct, r.suggested, r.covered, 100*r.recall())
		}
	}
}

type calThread struct {
	title, body string
	tags        []string
	query, doc  []float32
}

func loadThreads(t *testing.T, ctx context.Context, db *sql.DB) []*calThread {
	rows, err := db.QueryContext(ctx, `select p.id, coalesce(p.title, ''), p.body, coalesce(string_agg(t.name, '|'), '')
		from posts p
		left join tag_posts tp on tp.post_id = p.id
		left join tags t on t.id = tp.tag_id
		where p.root_post_id is null and p.deleted_at is null and p.visibility = 'published'
		group by p.id order by p.id`)
	require.NoError(t, err)
	defer rows.Close()

	var out []*calThread
	for rows.Next() {
		var id, body, tags string
		th := &calThread{}
		require.NoError(t, rows.Scan(&id, &th.title, &body, &tags))
		content, err := datagraph.NewRichTextWithBlocks(body)
		require.NoError(t, err)
		th.body = content.Content.Plaintext()
		if tags != "" {
			th.tags = strings.Split(tags, "|")
		}
		out = append(out, th)
	}
	require.NoError(t, rows.Err())
	return out
}

type calResult struct{ suggested, correct, covered, relevant int }

func (r calResult) precision() float64 { return float64(r.correct) / math.Max(1, float64(r.suggested)) }
func (r calResult) recall() float64    { return float64(r.correct) / math.Max(1, float64(r.relevant)) }

func leaveOneOut(threads []*calThread, minSim, share float64) calResult {
	var r calResult
	for _, th := range threads {
		if len(th.tags) == 0 {
			continue
		}

		type scored struct {
			th  *calThread
			sim float64
		}
		var near []scored
		for _, other := range threads {
			if other != th {
				near = append(near, scored{other, cosine(th.query, other.doc)})
			}
		}
		sort.Slice(near, func(i, j int) bool { return near[i].sim > near[j].sim })

		var neighbours []Neighbour
		for _, n := range near {
			if len(neighbours) == nearestThreads || n.sim < minSim {
				break
			}
			neighbours = append(neighbours, Neighbour{Similarity: n.sim, Tags: n.th.tags})
		}

		want := map[string]bool{}
		for _, tag := range th.tags {
			want[strings.ToLower(tag)] = true
		}
		r.relevant += len(want)

		got := VoteWith(neighbours, minVotes, share)
		if len(got) > 0 {
			r.covered++
		}
		for _, tag := range got {
			r.suggested++
			if want[strings.ToLower(tag)] {
				r.correct++
			}
		}
	}
	return r
}
