package agents

import (
	"fmt"

	markdown "github.com/MichaelMure/go-term-markdown"
	antagent "github.com/ant-libs-go/ant-agent"
)

type RenderSubAgent struct {
	CommonAgent
	cfg *antagent.Config
}

func NewRenderSubAgent(cfg *antagent.Config) (r *RenderSubAgent) {
	r = &RenderSubAgent{
		cfg: cfg,
	}
	return
}

func (this *RenderSubAgent) Name() string {
	return "RenderSubAgent"
}

func (this *RenderSubAgent) Description() string {
	return "将 Markdown 内容渲染为终端友好的格式"
}

func (this *RenderSubAgent) Clone() Agent {
	r := &RenderSubAgent{
		cfg: this.cfg,
	}
	return r
}

func (this *RenderSubAgent) Execute(ctx *Context, task *Task) (r *Result, err error) {
	fmt.Printf("\t 📝 正在渲染 Markdown 内容...\n")
	r = &Result{}

	for i := len(ctx.Tasks) - 1; i >= 0; i-- {
		if ctx.Tasks[i].Name != "ReportSubAgent" {
			continue
		}
		r.Output = string(markdown.Render(ctx.Tasks[i].Output, 80, 6))
	}

	fmt.Printf("\t 💬 渲染完成\n")
	return
}
