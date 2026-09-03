package prompt

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"liveclass/internal/rpc/agent/skills"
	"os"
	"path/filepath"
	"strings"
)

// SkillMeta holds the frontmatter of a tool .md file
type SkillMeta struct {
	Name        string
	Description string
}

func skillFS() (fs.FS, error) {
	if dir := strings.TrimSpace(os.Getenv("SKILLS_DIR")); dir != "" {
		return os.DirFS(filepath.Clean(dir)), nil
	}
	return skills.Files, nil
}

// LoadSkillIndex scans skillsDir and returns name→description pairs.
// Only the frontmatter is read, keeping token cost minimal.
func LoadSkillIndex() ([]SkillMeta, error) {
	assets, err := skillFS()
	if err != nil {
		return nil, fmt.Errorf("skills fs: %w", err)
	}
	entries, err := fs.ReadDir(assets, ".")
	if err != nil {
		return nil, fmt.Errorf("skills dir: %w", err)
	}
	var index []SkillMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		file, err := assets.Open(e.Name())
		if err != nil {
			continue
		}
		meta, parseErr := parseFrontmatter(file)
		file.Close()
		if parseErr != nil || meta.Name == "" {
			continue
		}
		index = append(index, meta)
	}
	return index, nil
}

// LoadSkillContent returns the full body (below frontmatter) of a tool file.
func LoadSkillContent(name string) (string, error) {
	assets, err := skillFS()
	if err != nil {
		return "", fmt.Errorf("skills fs: %w", err)
	}
	data, err := fs.ReadFile(assets, name+".md")
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
func parseFrontmatter(r io.Reader) (SkillMeta, error) {
	var meta SkillMeta
	scanner := bufio.NewScanner(r)
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
