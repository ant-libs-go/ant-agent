package agents

import (
	"context"
	"fmt"
	"strings"

	antagent "github.com/ant-libs-go/ant-agent"
	"github.com/ant-libs-go/util"
	"github.com/sashabaranov/go-openai"
)

const AnalyzeAgentSystemPrompt = `你是一个分析助手，负责综合和分析信息。请提供清晰、结构化的分析。`
const AnalyzeAgentUserPromptFormat = `用户的重要指令/请求: %s
分析以下信息并 %s:
%s

如果提供的信息不足以完成分析，你可以请求更多信息。
如果需要更多信息，请仅回复 'MISSING_INFO: <具体的搜索查询>'。例如: 'MISSING_INFO: 2024年Q3特斯拉财报数据'`

type AnalyzeSubAgent struct {
	CommonAgent
	cfg *antagent.Config
	cli *openai.Client
}

func NewAnalyzeSubAgent(cfg *antagent.Config) (r *AnalyzeSubAgent) {
	r = &AnalyzeSubAgent{
		cfg: cfg,
	}
	openaiCfg := openai.DefaultConfig(cfg.ApiKey)
	openaiCfg.BaseURL = cfg.ApiBase
	r.cli = openai.NewClientWithConfig(openaiCfg)

	r.AddSystemMessage(AnalyzeAgentSystemPrompt)
	return
}

func (this *AnalyzeSubAgent) Name() string {
	return "AnalyzeSubAgent"
}

func (this *AnalyzeSubAgent) Description() string {
	return "分析和综合收集到的信息"
}

func (this *AnalyzeSubAgent) Clone() Agent {
	r := &AnalyzeSubAgent{
		cfg: this.cfg,
		cli: this.cli,
	}

	r.AddSystemMessage(AnalyzeAgentSystemPrompt)
	return r
}

func (this *AnalyzeSubAgent) Execute(ctx *Context, task *Task) (r *Result, err error) {
	fmt.Printf("\t 🔬 正在通过已有信息分析...\n")
	r = &Result{}

	references := []string{}
	for i, t := range ctx.Tasks {
		if i >= ctx.Offset || len(t.Output) == 0 {
			continue
		}
		references = append(references, fmt.Sprintf("Output from %s task:\n%s", t.Name, t.Output))
	}
	this.AddUserMessage(fmt.Sprintf(AnalyzeAgentUserPromptFormat, ctx.Input, task.Description, strings.Join(references, "\n\n")))

	req := openai.ChatCompletionRequest{
		Model:       this.cfg.Model,
		Messages:    this.messages,
		Temperature: 0,
	}
	util.IfDo(this.cfg.Verbose, func() { LogStruct("AnalyzeSubAgent LLM Request", req) })

	var resp openai.ChatCompletionResponse
	if resp, err = this.cli.CreateChatCompletion(context.Background(), req); err != nil {
		err = fmt.Errorf("LLM 请求发生异常: %v", err)
		return
	}
	util.IfDo(this.cfg.Verbose, func() { LogStruct("AnalyzeSubAgent LLM Response", resp) })
	this.AddAssistantMessage(resp.Choices[0].Message.Content)

	llmResp := TrimLLMResp(resp.Choices[0].Message.Content)
	if strings.HasPrefix(llmResp, "MISSING_INFO:") {
		query := strings.TrimPrefix(llmResp, "MISSING_INFO:")
		fmt.Printf("\t 🔄 分析信息不完整，正在补充检索: %s\n", query)

		r.Tasks = append(r.Tasks, &Task{
			Name:        "SearchSubAgent",
			Description: "补充检索",
			Parameters:  map[string]interface{}{"query": query},
		}, task)
		return
	}

	r.Output = llmResp
	fmt.Printf("\t 💬 分析完成\n")
	return
}
