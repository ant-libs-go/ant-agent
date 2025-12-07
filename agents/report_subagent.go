package agents

import (
	"context"
	"fmt"
	"strings"

	antagent "github.com/ant-libs-go/ant-agent"
	"github.com/ant-libs-go/util"
	openai "github.com/sashabaranov/go-openai"
)

const ReportAgentSystemPrompt = `你是一个报告写作助手，负责创建格式良好、清晰且全面的 Markdown 格式报告。
使用适当的标题、列表和格式使报告易于阅读。
如果提供的信息包含带有 URL 和描述的图片，请选择最相关的图片，并使用标准 Markdown 图片语法 "![描述](URL)" 将其嵌入报告中。将图片放置在相关文本部分附近。`
const ReportAgentUserPromptFormat = `用户的重要指令/请求: %s
基于以下信息，%s：

%s`

type ReportSubAgent struct {
	CommonAgent
	cfg *antagent.Config
	cli *openai.Client
}

func NewReportSubAgent(cfg *antagent.Config) (r *ReportSubAgent) {
	r = &ReportSubAgent{
		cfg: cfg,
	}
	openaiCfg := openai.DefaultConfig(cfg.ApiKey)
	openaiCfg.BaseURL = cfg.ApiBase
	r.cli = openai.NewClientWithConfig(openaiCfg)

	r.AddSystemMessage(ReportAgentSystemPrompt)
	return
}

func (this *ReportSubAgent) Name() string {
	return "ReportSubAgent"
}

func (this *ReportSubAgent) Description() string {
	return "根据分析数据生成格式化报告"
}

func (this *ReportSubAgent) Clone() Agent {
	r := &ReportSubAgent{
		cfg: this.cfg,
		cli: this.cli,
	}

	r.AddSystemMessage(ReportAgentSystemPrompt)
	return r
}

func (this *ReportSubAgent) Execute(ctx *Context, task *Task) (r *Result, err error) {
	fmt.Printf("\t 📝 正在生成报告...\n")
	r = &Result{}

	references := []string{}
	for i, t := range ctx.Tasks {
		if i >= ctx.Offset || len(t.Output) == 0 {
			continue
		}
		references = append(references, fmt.Sprintf("Output from %s task:\n%s", t.Name, t.Output))
	}
	this.AddUserMessage(fmt.Sprintf(ReportAgentUserPromptFormat, ctx.Input, task.Description, strings.Join(references, "\n\n")))

	req := openai.ChatCompletionRequest{
		Model:       this.cfg.Model,
		Messages:    this.messages,
		Temperature: 0,
	}
	util.IfDo(this.cfg.Verbose, func() { LogStruct("ReportSubAgent LLM Request", req) })

	var resp openai.ChatCompletionResponse
	if resp, err = this.cli.CreateChatCompletion(context.Background(), req); err != nil {
		err = fmt.Errorf("LLM 请求发生异常: %v", err)
		return
	}
	util.IfDo(this.cfg.Verbose, func() { LogStruct("ReportSubAgent LLM Response", resp) })
	this.AddAssistantMessage(resp.Choices[0].Message.Content)

	llmResp := TrimLLMResp(resp.Choices[0].Message.Content)

	r.Output = llmResp
	fmt.Printf("\t 💬 生成报告完成\n")
	return
}
