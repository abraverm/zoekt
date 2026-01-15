// Copyright 2025 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zoekt

import (
	"bytes"
	"fmt"
	"testing"
)

// TestLargeBranchCount verifies that we can handle repositories with more than 64 branches,
// which was the limit when using uint64 bitmasks.
func TestLargeBranchCount(t *testing.T) {
	numBranches := 100

	// Create a repository with 100 branches.
	branches := make([]RepositoryBranch, numBranches)
	for i := 0; i < numBranches; i++ {
		branches[i] = RepositoryBranch{
			Name:    fmt.Sprintf("branch-%d", i),
			Version: fmt.Sprintf("v%d", i),
		}
	}

	repo := &Repository{
		Name:     "test-large-branch-repo",
		Branches: branches,
	}

	b, err := NewIndexBuilder(repo)
	if err != nil {
		t.Fatalf("NewIndexBuilder failed: %v", err)
	}

	// Add a document that appears in branches 0, 64, and 99 (testing across byte boundaries).
	doc := Document{
		Name:     "test.txt",
		Content:  []byte("test content"),
		Branches: []string{"branch-0", "branch-64", "branch-99"},
	}

	if err := b.Add(doc); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Verify the branch mask was created correctly.
	if len(b.branchMasks) != 1 {
		t.Fatalf("expected 1 branch mask, got %d", len(b.branchMasks))
	}

	mask := b.branchMasks[0]

	// Check that the expected bits are set.
	if !getBit(mask, 0) {
		t.Error("expected bit 0 (branch-0) to be set")
	}
	if !getBit(mask, 64) {
		t.Error("expected bit 64 (branch-64) to be set")
	}
	if !getBit(mask, 99) {
		t.Error("expected bit 99 (branch-99) to be set")
	}

	// Verify we can write and read back the shard.
	var buf bytes.Buffer
	if err := b.Write(&buf); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Create an in-memory index file.
	f := &memIndexFile{data: buf.Bytes()}

	// Read it back.
	d, err := loadIndexData(f)
	if err != nil {
		t.Fatalf("loadIndexData failed: %v", err)
	}

	// Verify the branch mask was preserved.
	if len(d.fileBranchMasks) != 1 {
		t.Fatalf("expected 1 file branch mask, got %d", len(d.fileBranchMasks))
	}

	readMask := d.fileBranchMasks[0]
	if !getBit(readMask, 0) || !getBit(readMask, 64) || !getBit(readMask, 99) {
		t.Error("branch mask was not preserved correctly after write/read")
	}
}

// memIndexFile is a simple in-memory implementation of IndexFile for testing.
type memIndexFile struct {
	data []byte
}

func (m *memIndexFile) Read(off uint32, sz uint32) ([]byte, error) {
	if off+sz > uint32(len(m.data)) {
		return nil, fmt.Errorf("read past end of data")
	}
	return m.data[off : off+sz], nil
}

func (m *memIndexFile) Size() (uint32, error) {
	return uint32(len(m.data)), nil
}

func (m *memIndexFile) Close() {}

func (m *memIndexFile) Name() string {
	return "mem"
}


