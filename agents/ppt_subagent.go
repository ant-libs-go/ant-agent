package agents

type PPTSubAgent struct {
	CommonAgent
}

func NewPPTSubAgent() *PPTSubAgent {
	return &PPTSubAgent{}
}

func (this *PPTSubAgent) Name() string {
	return "PPTSubAgent"
}

func (this *PPTSubAgent) Description() string {
	return "根据报告生成幻灯片 (HTML)"
}

func (this *PPTSubAgent) Clone() Agent {
	r := &PPTSubAgent{}
	return r
}

func (this *PPTSubAgent) Execute(ctx *Context, task *Task) (*Result, error) {
	return &Result{}, nil
}
