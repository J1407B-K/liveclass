package prompt

import "testing"

func TestEmbeddedSkillsLoadWithoutWorkingDirectory(t *testing.T) {
	t.Setenv("SKILLS_DIR", "")

	index, err := LoadSkillIndex()
	if err != nil {
		t.Fatalf("LoadSkillIndex() error = %v", err)
	}
	if len(index) != 6 {
		t.Fatalf("LoadSkillIndex() count = %d, want 6", len(index))
	}
	content, err := LoadSkillContent("lesson_plan")
	if err != nil {
		t.Fatalf("LoadSkillContent() error = %v", err)
	}
	if content == "" {
		t.Fatal("LoadSkillContent() returned empty content")
	}
}

func TestSkillsDirOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SKILLS_DIR", dir)
	if _, err := LoadSkillIndex(); err != nil {
		t.Fatalf("LoadSkillIndex() with override error = %v", err)
	}
}
