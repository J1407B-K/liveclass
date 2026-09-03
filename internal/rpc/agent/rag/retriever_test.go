package rag

import "testing"

func TestExpandAndDeduplicateParents(t *testing.T) {
	got := ExpandAndDeduplicateParents([]DocChunk{{ID: "c1", ParentID: "p1", Text: "child1", ParentText: "parent"}, {ID: "c2", ParentID: "p1", Text: "child2", ParentText: "parent"}, {ID: "c3", ParentID: "p2", Text: "child3", ParentText: "other"}})
	if len(got) != 2 {
		t.Fatalf("parents = %d, want 2", len(got))
	}
	if got[0].ID != "p1" || got[0].Text != "parent" {
		t.Fatalf("parent not expanded: %#v", got[0])
	}
}
