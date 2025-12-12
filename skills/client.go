package skills

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

type SkillClient struct {
	skills map[string]*Skill
}

func NewSkillClient(path string) (r *SkillClient, err error) {
	r = &SkillClient{
		skills: make(map[string]*Skill),
	}
	if err = filepath.WalkDir(path, func(path string, d fs.DirEntry, er error) (err error) {
		if er != nil {
			err = er
			return
		}

		var skill *Skill
		if !d.IsDir() && d.Name() == "SKILL.md" {
			if skill, err = r.parseSkill(filepath.Dir(path)); err != nil { // 忽略解析失败的skill
				fmt.Println(fmt.Errorf("failed to parse skill: %w", err))
				return
			}
			r.skills[skill.Meta.Name] = skill
		}
		return
	}); err != nil {
		err = fmt.Errorf("failed to walk dir: %w", err)
		return
	}

	return
}

func (this *SkillClient) GetSkills() (r []*Skill) {
	for _, skill := range this.skills {
		r = append(r, skill)
	}
	return
}
