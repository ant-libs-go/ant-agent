package agents

import (
	"encoding/json"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type Agent interface {
	Name() string
	Description() string
	Clone() Agent
	Execute(ctx *Context, task *Task) (*Result, error)
}

type Context struct {
	Input  string  `json:"input"`
	Plans  string  `json:"plans"`
	Offset int     `json:"offset"`
	Tasks  []*Task `json:"tasks"`
}

func (this *Context) ClearChatHistory() {
	this.Input = ""
	this.Plans = ""
	this.Offset = 0
	this.Tasks = []*Task{}
}

type Result struct {
	Tasks  []*Task `json:"tasks"`
	Output string  `json:"output"`
}

type Task struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Output      string                 `json:"output"`
}

type CommonAgent struct {
	messages []openai.ChatCompletionMessage
}

func (this *CommonAgent) AddSystemMessage(content string) {
	this.messages = append([]openai.ChatCompletionMessage{
		openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: content,
		},
	}, this.messages...)
}

func (this *CommonAgent) AddUserMessage(content string) {
	this.messages = append(this.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: content,
	})
}

func (this *CommonAgent) AddAssistantMessage(content string) {
	this.messages = append(this.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: content,
	})
}

func (this *CommonAgent) AddFunctionMessage(content string) {
	this.messages = append(this.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleFunction,
		Content: content,
	})
}
func (this *CommonAgent) AddToolMessage(content string) {
	this.messages = append(this.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleTool,
		Content: content,
	})
}

func (this *CommonAgent) AddDeveloperMessage(content string) {
	this.messages = append(this.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleDeveloper,
		Content: content,
	})
}

func TrimLLMResp(inp string) string {
	// 如果存在 ```json 前缀，则剔除
	if idx := strings.Index(inp, "```json"); idx != -1 {
		inp = inp[idx+7:]
	} else if idx := strings.Index(inp, "```"); idx != -1 {
		inp = inp[idx+3:]
	}

	// 如果存在关闭的 ```，则剔除
	if idx := strings.LastIndex(inp, "```"); idx != -1 {
		inp = inp[:idx]
	}

	return strings.TrimSpace(inp)
}

func LogStruct(prefix string, obj interface{}) {
	b, _ := json.Marshal(obj)
	fmt.Printf("%s: %s\n", prefix, string(b))
}
