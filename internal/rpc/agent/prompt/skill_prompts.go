package prompt

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SkillMeta holds the frontmatter of a tool .md file
type SkillMeta struct {
	Name        string
	Description string
}

// skillsDir is resolved once at init relative to this file's location.
// Override via SKILLS_DIR env var for deployment flexibility.
var skillsDir = func() string {
	if d := os.Getenv("SKILLS_DIR"); d != "" {
		return d
	}
	// default: <module_root>/internal/rpc/agent/skills
	return filepath.Join(os.Getenv("PWD"), "internal/rpc/agent/skills")
}()

// LoadSkillIndex scans skillsDir and returns name→description pairs.
// Only the frontmatter is read, keeping token cost minimal.
func LoadSkillIndex() ([]SkillMeta, error) {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("skills dir: %w", err)
	}
	var index []SkillMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		meta, err := parseFrontmatter(filepath.Join(skillsDir, e.Name()))
		if err != nil || meta.Name == "" {
			continue
		}
		index = append(index, meta)
	}
	return index, nil
}

// LoadSkillContent returns the full body (below frontmatter) of a tool file.
func LoadSkillContent(name string) (string, error) {
	path := filepath.Join(skillsDir, name+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("tool %q not found: %w", name, err)
	}
	return stripFrontmatter(string(data)), nil
}

// BuildAdvisorSystemPrompt constructs the advisor prompt from the live tool index.
func BuildAdvisorSystemPrompt(index []SkillMeta) string {
	var sb strings.Builder
	sb.WriteString("你是教学助手的意图分析模块。\n")
	sb.WriteString("根据用户消息，从以下技能中选择一个或多个最匹配的，输出技能名称列表。\n\n")
	sb.WriteString("可用技能：\n")
	for _, m := range index {
		fmt.Fprintf(&sb, "- %s：%s\n", m.Name, m.Description)
	}
	sb.WriteString("\n严格输出以下 JSON，不要有任何多余内容：\n")
	sb.WriteString(`{"skills": ["<技能名>", ...]}`)
	return sb.String()
}

// parseFrontmatter reads only the YAML frontmatter block of a .md file.
func parseFrontmatter(path string) (SkillMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return SkillMeta{}, err
	}
	defer f.Close()

	var meta SkillMeta
	scanner := bufio.NewScanner(f)
	inFront := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			if !inFront {
				inFront = true
				continue
			}
			break // end of frontmatter
		}
		if !inFront {
			continue
		}
		if k, v, ok := strings.Cut(line, ":"); ok {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			switch k {
			case "name":
				meta.Name = v
			case "description":
				meta.Description = v
			}
		}
	}
	return meta, scanner.Err()
}

// stripFrontmatter removes the leading --- block and returns the body.
func stripFrontmatter(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return content
	}
	// find closing ---
	rest := content[3:]
	idx := strings.Index(rest, "---")
	if idx == -1 {
		return content
	}
	return strings.TrimSpace(rest[idx+3:])
}
