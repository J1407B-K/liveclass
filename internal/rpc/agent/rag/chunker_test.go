package rag

import "testing"

func TestChunkMarkdownBuildsParentsAndOverlappingChildren(t *testing.T) {
	parents, err := ChunkMarkdown("# 第一节\nabcdefghij\n## 第二节\nklmnopqrst", ChunkConfig{ParentSize: 20, ChildSize: 8, Overlap: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(parents) < 2 {
		t.Fatalf("parents = %d, want heading-aware sections", len(parents))
	}
	if len(parents[0].Children) < 2 {
		t.Fatalf("children = %d, want windows", len(parents[0].Children))
	}
	first := []rune(parents[0].Children[0].Text)
	second := []rune(parents[0].Children[1].Text)
	if string(first[len(first)-2:]) != string(second[:2]) {
		t.Fatalf("children do not overlap: %q / %q", first, second)
	}
}

func TestChunkConfigRejectsUnboundedOverlap(t *testing.T) {
	if _, err := ChunkMarkdown("x", ChunkConfig{ParentSize: 10, ChildSize: 5, Overlap: 5}); err == nil {
		t.Fatal("expected invalid overlap")
	}
}
