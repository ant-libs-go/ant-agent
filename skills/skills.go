package skills

type Skill struct {
	Path      string          `json:"path"`
	Meta      *SkillMeta      `json:"meta"`
	Body      string          `json:"body"` // SKILL.md 主体的原始 Markdown 内容
	Resources *SkillResources `json:"resources"`
}

type SkillMeta struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	AllowedTools []string `yaml:"allowed-tools"`
	Model        string   `yaml:"model,omitempty"`
	Author       string   `yaml:"author,omitempty"`
	Version      string   `yaml:"version,omitempty"`
	License      string   `yaml:"license,omitempty"`
}

type SkillResources struct {
	Scripts    []string `json:"scripts"`
	References []string `json:"references"`
	Assets     []string `json:"assets"`
	Templates  []string `json:"templates"`
}
