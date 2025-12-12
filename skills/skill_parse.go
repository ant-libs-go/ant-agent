package skills

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func (this *SkillClient) parseSkill(dir string) (r *Skill, err error) {
	r = &Skill{
		Path: dir,
		Meta: &SkillMeta{},
		Resources: &SkillResources{
			Scripts:    []string{},
			References: []string{},
			Assets:     []string{},
			Templates:  []string{},
		},
	}

	skillmdPath := filepath.Join(r.Path, "SKILL.md")
	var skillmdContent []byte
	if skillmdContent, err = os.ReadFile(skillmdPath); err != nil {
		err = fmt.Errorf("failed to read skill md: %w", err)
		return
	}

	parts := bytes.SplitN(skillmdContent, []byte("---"), 3)
	if len(parts) < 3 {
		err = fmt.Errorf("no YAML frontmatter found or format is incorrect")
		return
	}

	if err = yaml.Unmarshal(parts[1], r.Meta); err != nil {
		err = fmt.Errorf("failed to parse SKILL.md frontmatter: %w", err)
		return
	}

	r.Body = strings.TrimSpace(string(parts[2]))

	if r.Resources.Scripts, err = this.parseResourceFiles(r.Path, "scripts"); err != nil {
		r.Resources.Scripts = []string{}
		err = fmt.Errorf("failed to find scripts: %w", err)
		return
	}
	if r.Resources.References, err = this.parseResourceFiles(r.Path, "references"); err != nil {
		r.Resources.References = []string{}
		err = fmt.Errorf("failed to find references: %w", err)
		return
	}
	if r.Resources.Assets, err = this.parseResourceFiles(r.Path, "assets"); err != nil {
		r.Resources.Assets = []string{}
		err = fmt.Errorf("failed to find assets: %w", err)
		return
	}
	if r.Resources.Templates, err = this.parseResourceFiles(r.Path, "templates"); err != nil {
		r.Resources.Templates = []string{}
		err = fmt.Errorf("failed to find templates: %w", err)
		return
	}

	return
}

func (this *SkillClient) parseResourceFiles(skillPath, resourceDir string) (r []string, err error) {
	scanDir := filepath.Join(skillPath, resourceDir)

	if _, er := os.Stat(scanDir); os.IsNotExist(er) {
		return
	}

	err = filepath.WalkDir(scanDir, func(path string, d fs.DirEntry, er error) (err error) {
		if er != nil {
			err = er
			return
		}
		if d.IsDir() {
			return
		}

		var relPath string
		if relPath, err = filepath.Rel(skillPath, path); err != nil {
			return
		}
		r = append(r, relPath)
		return
	})

	return
}
