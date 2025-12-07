package agents

import (
	"context"
	"encoding/json"
	"fmt"

	antagent "github.com/ant-libs-go/ant-agent"
	"github.com/ant-libs-go/util"
	openai "github.com/sashabaranov/go-openai"
)

const PlanningAgentSystemPrompt = `
# 你是一个负责任务规划的 Agent，将用户请求分解为子任务。

## 你可以使用以下 SubAgent：
%s

## 对于给定的用户请求，创建一个包含任务序列的计划。每个任务应包含：
- name: 任意一个 SubAgent 的名称
- description:  SubAgent 应该做什么
- parameters: 任务的可选参数 (例如: {"query": "搜索词"})

## 仅返回具有此结构的有效 JSON 对象：
{
  "output": "总体计划描述",
  "tasks": [
    {"name": "SearchSubAgent", "description": "...", "parameters": {"query": "..."}},
    {"name": "AnalyzeSubAgent", "description": "..."},
    {"name": "ReportSubAgent", "description": "..."},
    {"name": "PPTSubAgent", "description": "根据报告生成幻灯片"},
    {"name": "RenderSubAgent", "description": "渲染报告"}
  ]
}

## 重要提示：
- 仅在用户明确请求幻灯片或演示文稿时包含 PPT 任务。
- 在 REPORT 任务之后始终包含 RENDER 任务，以生成最终的文本报告。
- 如果判定用户请求不需要进行任务规划，返回结果中指定 output 为回复用户的内容且 tasks 为空， 否则返回 tasks 且 output 为空。
- 保持计划简单且重点突出。通常 3-8 个任务就足够了。`

type PlanningAgent struct {
	CommonAgent
	cfg       *antagent.Config
	cli       *openai.Client
	subagents map[string]Agent
}

func NewPlanningAgent(cfg *antagent.Config) (r *PlanningAgent) {
	r = &PlanningAgent{
		cfg:       cfg,
		subagents: map[string]Agent{},
	}
	openaiCfg := openai.DefaultConfig(cfg.ApiKey)
	openaiCfg.BaseURL = cfg.ApiBase
	r.cli = openai.NewClientWithConfig(openaiCfg)

	r.AddSubAgent(NewSearchSubAgent(cfg))
	r.AddSubAgent(NewAnalyzeSubAgent(cfg))
	r.AddSubAgent(NewReportSubAgent(cfg))
	//r.AddSubAgent(NewPPTSubAgent(cfg))
	r.AddSubAgent(NewRenderSubAgent(cfg))

	subAgentsPrompt := ""
	for _, agent := range r.subagents {
		subAgentsPrompt += fmt.Sprintf("- %s: %s\n", agent.Name(), agent.Description())
	}
	r.AddSystemMessage(fmt.Sprintf(PlanningAgentSystemPrompt, subAgentsPrompt))
	return
}

func (this *PlanningAgent) Name() string {
	return "PlanningAgent"
}

func (this *PlanningAgent) Description() string {
	return "负责任务规划的 Agent，将用户请求分解为子任务"
}

func (this *PlanningAgent) Clone() Agent {
	r := &PlanningAgent{
		cfg:       this.cfg,
		cli:       this.cli,
		subagents: map[string]Agent{},
	}

	r.AddSubAgent(NewSearchSubAgent(this.cfg))
	r.AddSubAgent(NewAnalyzeSubAgent(this.cfg))
	r.AddSubAgent(NewReportSubAgent(this.cfg))
	//r.AddSubAgent(NewPPTSubAgent(cfg))
	r.AddSubAgent(NewRenderSubAgent(this.cfg))

	subAgentsPrompt := ""
	for _, agent := range r.subagents {
		subAgentsPrompt += fmt.Sprintf("- %s: %s\n", agent.Name(), agent.Description())
	}
	r.AddSystemMessage(fmt.Sprintf(PlanningAgentSystemPrompt, subAgentsPrompt))
	return r
}

func (this *PlanningAgent) AddSubAgent(agent Agent) {
	this.subagents[agent.Name()] = agent
}

func (this *PlanningAgent) GetSubAgent(name string) Agent {
	return this.subagents[name].Clone()
}

func (this *PlanningAgent) plan() (r *Result, err error) {
	req := openai.ChatCompletionRequest{
		Model:       this.cfg.Model,
		Messages:    this.messages,
		Temperature: 0,
	}
	util.IfDo(this.cfg.Verbose, func() { LogStruct("PlanningAgent LLM Request", req) })

	var resp openai.ChatCompletionResponse
	if resp, err = this.cli.CreateChatCompletion(context.Background(), req); err != nil {
		err = fmt.Errorf("LLM 请求发生异常: %v", err)
		return
	}
	util.IfDo(this.cfg.Verbose, func() { LogStruct("PlanningAgent LLM Response", resp) })
	this.AddAssistantMessage(resp.Choices[0].Message.Content)

	content := TrimLLMResp(resp.Choices[0].Message.Content)

	r = &Result{}
	if err = json.Unmarshal([]byte(content), r); err != nil {
		err = fmt.Errorf("LLM 应答无法解析: %v, %s", err, content)
		return
	}
	return
}

func (this *PlanningAgent) Execute(ctx *Context, task *Task) (r *Result, err error) {
	fmt.Printf("🧠 正在规划你的任务...\n")
	r = &Result{}

	for {
		var result *Result
		if result, err = this.plan(); err != nil {
			err = fmt.Errorf("任务规划异常: %v", err)
			return
		}

		if len(result.Tasks) == 0 {
			r.Output = result.Output
			return
		}

		fmt.Printf("📝 LLM 已经完成任务规划: \n")
		for idx, task := range result.Tasks {
			fmt.Printf(" %d. [%s] %s.\n", idx+1, task.Name, task.Description)
		}
		fmt.Printf("\n\n❓ 请确认是否认可该方案？认可请回复 继续/y/yes，否则请继续完善你的需求\n")

		var input string
		if input, err = antagent.GetInput(); err != nil {
			err = fmt.Errorf("用户输入获取异常: %v", err)
			return
		}
		if exists, _ := util.InSlice(input, []string{"继续", "y", "yes"}); exists {
			r = result
			return
		}

		this.AddUserMessage(input)
		fmt.Printf("🔄 正在重新规划你的任务...\n")
	}
}
