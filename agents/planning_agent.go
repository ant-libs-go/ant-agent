package agents

import (
	"context"
	"encoding/json"
	"fmt"

	antagent "github.com/ant-libs-go/ant-agent"
	"github.com/ant-libs-go/ant-agent/skills"
	"github.com/ant-libs-go/util"
	openai "github.com/sashabaranov/go-openai"
)

const PlanningAgentSystemPrompt = `
# 你是系统的主协调代理（Main Orchestrator Agent），你的任务是：解析用户请求 → 规划任务 → 对每个任务步骤选择执行方式 → 产生结构化计划。

你可以调用 2 种执行单元：
1. **Skill**：模型内部的可执行能力，用于轻量、纯逻辑、无需外部资源的任务。
2. **SubAgent**：独立的专家代理，适用于复杂、领域特化、需要进一步规划的任务。

## 你可以使用以下 Skill：
%s

## 你可以使用以下 SubAgent：
%s

## 对于给定的用户请求，创建一个包含任务序列的计划。每个任务应包含：
- name: 任意一个 Skill 或 SubAgent 的名称
- description:  Skill 或 SubAgent 应该做什么
- parameters: 任务的可选参数 (例如: {"query": "搜索词"})

## 仅返回具有此结构的有效 JSON 对象：
{
  "output": "总体计划描述",
  "tasks": [
    {"name": "CodeReviewSkill", "description": "..."},
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
	skills    map[string]*skills.Skill
	subagents map[string]Agent
}

func NewPlanningAgent(cfg *antagent.Config, agentss []Agent, skillss []*skills.Skill) (r *PlanningAgent) {
	r = &PlanningAgent{
		cfg:       cfg,
		skills:    map[string]*skills.Skill{},
		subagents: map[string]Agent{},
	}
	openaiCfg := openai.DefaultConfig(cfg.ApiKey)
	openaiCfg.BaseURL = cfg.ApiBase
	r.cli = openai.NewClientWithConfig(openaiCfg)

	for _, skill := range skillss {
		r.AddSkill(skill)
	}

	for _, agent := range agentss {
		r.AddSubAgent(agent)
	}

	skillsPrompt := ""
	for _, skill := range r.skills {
		skillsPrompt += fmt.Sprintf("- %s: %s\n", skill.Meta.Name, skill.Meta.Description)
	}

	subAgentsPrompt := ""
	for _, agent := range r.subagents {
		subAgentsPrompt += fmt.Sprintf("- %s: %s\n", agent.Name(), agent.Description())
	}
	r.AddSystemMessage(fmt.Sprintf(PlanningAgentSystemPrompt, skillsPrompt, subAgentsPrompt))
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
		skills:    map[string]*skills.Skill{},
		subagents: map[string]Agent{},
	}

	for _, skill := range this.skills {
		r.AddSkill(skill)
	}

	for _, agent := range this.subagents {
		r.AddSubAgent(agent.Clone())
	}

	skillsPrompt := ""
	for _, skill := range r.skills {
		skillsPrompt += fmt.Sprintf("- %s: %s\n", skill.Meta.Name, skill.Meta.Description)
	}

	subAgentsPrompt := ""
	for _, agent := range r.subagents {
		subAgentsPrompt += fmt.Sprintf("- %s: %s\n", agent.Name(), agent.Description())
	}
	r.AddSystemMessage(fmt.Sprintf(PlanningAgentSystemPrompt, skillsPrompt, subAgentsPrompt))
	return r
}
func (this *PlanningAgent) AddSkill(skill *skills.Skill) {
	this.skills[skill.Meta.Name] = skill
}

func (this *PlanningAgent) GetSkill(name string) *skills.Skill {
	return this.skills[name]
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
	this.AddUserMessage(ctx.Input)

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
