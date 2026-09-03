package rag

import (
	"fmt"
	"regexp"
	"strings"
)

type ChunkConfig struct{ ParentSize, ChildSize, Overlap int }

func DefaultChunkConfig() ChunkConfig {
	return ChunkConfig{ParentSize: 1600, ChildSize: 400, Overlap: 80}
}
func (c ChunkConfig) Validate() error {
	if c.ParentSize < 1 || c.ChildSize < 1 || c.ChildSize > c.ParentSize {
		return fmt.Errorf("invalid parent/child sizes")
	}
	if c.Overlap < 0 || c.Overlap >= c.ChildSize {
		return fmt.Errorf("overlap must be smaller than child size")
	}
	return nil
}

type ParentChunk struct {
	Index         int
	Heading, Text string
	Children      []ChildChunk
}
type ChildChunk struct {
	Index int
	Text  string
}

var markdownHeading = regexp.MustCompile(`(?m)^#{1,6}\s+.+$`)

// ChunkMarkdown respects heading boundaries first, then applies bounded parent
// and overlapping child windows. Sizes are Unicode character counts.
func ChunkMarkdown(content string, config ChunkConfig) ([]ParentChunk, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	sections := markdownSections(strings.TrimSpace(content))
	parents := make([]ParentChunk, 0)
	for _, section := range sections {
		for _, text := range splitBounded(section.text, config.ParentSize, 0) {
			parent := ParentChunk{Index: len(parents), Heading: section.heading, Text: strings.TrimSpace(text)}
			children := splitBounded(parent.Text, config.ChildSize, config.Overlap)
			for _, child := range children {
				parent.Children = append(parent.Children, ChildChunk{Index: len(parent.Children), Text: strings.TrimSpace(child)})
			}
			if parent.Text != "" {
				parents = append(parents, parent)
			}
		}
	}
	return parents, nil
}

type markdownSection struct{ heading, text string }

func markdownSections(content string) []markdownSection {
	if content == "" {
		return nil
	}
	indexes := markdownHeading.FindAllStringIndex(content, -1)
	if len(indexes) == 0 {
		return []markdownSection{{text: content}}
	}
	var sections []markdownSection
	if prefix := strings.TrimSpace(content[:indexes[0][0]]); prefix != "" {
		sections = append(sections, markdownSection{text: prefix})
	}
	for i, idx := range indexes {
		end := len(content)
		if i+1 < len(indexes) {
			end = indexes[i+1][0]
		}
		heading := strings.TrimSpace(content[idx[0]:idx[1]])
		sections = append(sections, markdownSection{heading: heading, text: strings.TrimSpace(content[idx[0]:end])})
	}
	return sections
}

func splitBounded(text string, size, overlap int) []string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return nil
	}
	var chunks []string
	step := size - overlap
	for start := 0; start < len(runes); start += step {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
		if end == len(runes) {
			break
		}
	}
	return chunks
}
