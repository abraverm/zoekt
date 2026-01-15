package zoekt

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/zoekt/query"
)

// We compare simple shards before and after the transformation
// explode(merge(shard1, shard2)).
//
// The exploded shards are expected to be semantically identical (same docs,
// contents, and branches), but may not be byte-identical because the on-disk
// format changed (v16 -> v17).
func TestExplode(t *testing.T) {
	v16Shards := []string{
		"./testdata/shards/repo_v16.00000.zoekt",
		"./testdata/shards/repo2_v16.00000.zoekt",
	}

	// repo name -> IndexMetadata
	m := make(map[string]*IndexMetadata, 2)

	// merge
	var files []IndexFile
	for _, fn := range v16Shards {
		f, err := os.Open(fn)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		indexFile, err := NewIndexFile(f)
		if err != nil {
			t.Fatal(err)
		}
		defer indexFile.Close()

		// We save indexMeta because the fields ID and IndexTime are the 2 sources of
		// non-determinism when building a new shard.
		repoMeta, indexMeta, err := ReadMetadata(indexFile)
		if err != nil {
			t.Fatal(err)
		}
		if len(repoMeta) != 1 {
			t.Fatal("this test assumes that indexFile contains only 1 repo")
		}
		m[repoMeta[0].Name] = indexMeta

		files = append(files, indexFile)
	}

	tmpDir := t.TempDir()
	tmpName, dstName, err := Merge(tmpDir, files...)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Rename(tmpName, dstName)
	if err != nil {
		t.Fatal(err)
	}

	// explode
	f, err := os.Open(dstName)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	indexFile, err := NewIndexFile(f)
	if err != nil {
		t.Fatal(err)
	}
	defer indexFile.Close()

	overwriteIndexTimeAndID := func(ib *IndexBuilder) {
		ib.ID = m[ib.repoList[0].Name].ID
		ib.IndexTime = m[ib.repoList[0].Name].IndexTime
	}
	exploded, err := explode(tmpDir, indexFile, overwriteIndexTimeAndID)
	if err != nil {
		t.Fatal(err)
	}
	for tmp, final := range exploded {
		err = os.Rename(tmp, final)
		if err != nil {
			t.Fatal(err)
		}
	}

	for _, s := range v16Shards {
		wantRepo, _, err := ReadMetadataPath(s)
		if err != nil {
			t.Fatal(err)
		}
		if len(wantRepo) != 1 {
			t.Fatal("this test assumes that shard contains only 1 repo")
		}
		gotPath := ShardName(tmpDir, wantRepo[0].Name, IndexFormatVersion, 0)
		checkEquivalentShards(t, s, gotPath)
	}
}

// checkEquivalentShards compares 2 shards semantically via search results over
// all documents (query.Const(true) + Whole=true).
func checkEquivalentShards(t *testing.T, shard1, shard2 string) {
	t.Helper()

	s1, err := loadShardForTest(shard1)
	if err != nil {
		t.Fatal(err)
	}
	defer s1.Close()

	s2, err := loadShardForTest(shard2)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	ctx := context.Background()
	q := &query.Const{Value: true}
	opts := &SearchOptions{Whole: true}

	r1, err := s1.Search(ctx, q, opts)
	if err != nil {
		t.Fatalf("Search(%s): %v", shard1, err)
	}
	r2, err := s2.Search(ctx, q, opts)
	if err != nil {
		t.Fatalf("Search(%s): %v", shard2, err)
	}

	clearScores(r1)
	clearScores(r2)

	if d := cmp.Diff(r1.Files, r2.Files); d != "" {
		t.Fatalf("shards not equivalent (-%s +%s):\n%s", shard1, shard2, d)
	}
}

func loadShardForTest(fn string) (Searcher, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}

	iFile, err := NewIndexFile(f)
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	s, err := NewSearcher(iFile)
	if err != nil {
		iFile.Close()
		return nil, err
	}

	return s, nil
}
