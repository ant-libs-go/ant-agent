package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	antagent "github.com/ant-libs-go/ant-agent"
	"github.com/ant-libs-go/ant-agent/skills"
	"github.com/ant-libs-go/util"
	openai "github.com/sashabaranov/go-openai"
)

const SkillSubAgentSystemPrompt = `%s
## 技能上下文:
技能根目录：%s`

const SkillSubAgentUserPromptFormat = `用户的重要指令/请求: %s
当前任务目标：%s
上下文内容：
%s`

type SkillSubAgent struct {
	CommonAgent
	skill *skills.Skill
	cfg   *antagent.Config
	cli   *openai.Client
}

func NewSkillSubAgent(cfg *antagent.Config, skill *skills.Skill) (r *SkillSubAgent) {
	r = &SkillSubAgent{
		cfg:   cfg,
		skill: skill,
	}
	openaicfg := openai.DefaultConfig(cfg.ApiKey)
	openaicfg.BaseURL = cfg.ApiBase
	r.cli = openai.NewClientWithConfig(openaicfg)

	r.AddSystemMessage(fmt.Sprintf(SkillSubAgentSystemPrompt, skill.Body, skill.Path))
	return
}

func (this *SkillSubAgent) Name() string {
	return "SkillSubAgent"
}

func (this *SkillSubAgent) Description() string {
	return "skill 的执行代理"
}

func (this *SkillSubAgent) Clone() Agent {
	return nil
}

func (this *SkillSubAgent) Execute(ctx *Context, task *Task) (r *Result, err error) {
	fmt.Printf("\t 🔬 正在调用 skill[%s]...\n", this.skill.Meta.Name)
	r = &Result{}

	references := []string{}
	for i, t := range ctx.Tasks {
		if i >= ctx.Offset || len(t.Output) == 0 {
			continue
		}
		references = append(references, fmt.Sprintf("Output from %s task:\n%s", t.Name, t.Output))
	}
	this.AddUserMessage(fmt.Sprintf(SkillSubAgentUserPromptFormat, ctx.Input, task.Description, strings.Join(references, "\n\n")))

	for i := 0; i < 10; i++ {
		req := openai.ChatCompletionRequest{
			Model:       this.cfg.Model,
			Messages:    this.messages,
			Temperature: 0,
			Tools:       ctx.McpClient.GetTools(),
		}
		util.IfDo(this.cfg.Verbose, func() { LogStruct("SkillSubAgent LLM Request", req) })

		var resp openai.ChatCompletionResponse
		if resp, err = this.cli.CreateChatCompletion(context.Background(), req); err != nil {
			err = fmt.Errorf("LLM 请求发生异常: %v", err)
			return
		}
		util.IfDo(this.cfg.Verbose, func() { LogStruct("SkillSubAgent LLM Response", resp) })
		//this.AddAssistantMessage(resp.Choices[0].Message.Content)
		this.messages = append(this.messages, resp.Choices[0].Message)

		if len(resp.Choices[0].Message.ToolCalls) == 0 {
			r.Output = TrimLLMResp(resp.Choices[0].Message.Content)
			return
		}

		for _, toolCall := range resp.Choices[0].Message.ToolCalls {
			util.IfDo(this.cfg.Verbose, func() {
				fmt.Printf("SkillSubAgent ToolCall[%s]: %s\n", toolCall.Function.Name, toolCall.Function.Arguments)
			})

			var args map[string]interface{}
			if err = json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				err = fmt.Errorf("调用 tool[%s] 参数解析失败: %v", toolCall.Function.Name, err)
			}

			var toolResp interface{}
			if err == nil {
				if toolResp, err = ctx.McpClient.CallTool(context.Background(), toolCall.Function.Name, args); err != nil {
					err = fmt.Errorf("调用 tool[%s] 失败: %v", toolCall.Function.Name, err)
				}
			}

			b, _ := json.Marshal(toolResp)
			msg := string(b)
			if err != nil {
				msg = err.Error()
			}

			this.AddToolMessage(toolCall.ID, msg)
		}
	}

	err = errors.New("超出 tool 调用的最大次数")
	return
}
