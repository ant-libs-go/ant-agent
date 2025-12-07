package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	antagent "github.com/ant-libs-go/ant-agent"
	"github.com/ant-libs-go/util"
	openai "github.com/sashabaranov/go-openai"
)

const SearchAgentSystemPrompt = `你是一个搜索优化助手。你评估搜索结果并决定是否需要更多信息。`
const SearchAgentUserPromptFormat = `用户查询: %s
当前搜索结果:
%s

信息是否足以回答用户的查询？
如果是，请仅回复 "SUFFICIENT"。
如果否，请回复一个新的、更精细的搜索查询以查找缺失的信息。不要添加任何其他文本。
`

type SearchSubAgent struct {
	CommonAgent
	cfg *antagent.Config
	cli *openai.Client
}

func NewSearchSubAgent(cfg *antagent.Config) (r *SearchSubAgent) {
	r = &SearchSubAgent{
		cfg: cfg,
	}
	openaiCfg := openai.DefaultConfig(cfg.ApiKey)
	openaiCfg.BaseURL = cfg.ApiBase
	r.cli = openai.NewClientWithConfig(openaiCfg)

	r.AddSystemMessage(SearchAgentSystemPrompt)
	return
}

func (this *SearchSubAgent) Name() string {
	return "SearchSubAgent"
}

func (this *SearchSubAgent) Description() string {
	return "执行网络搜索以收集信息"
}

func (this *SearchSubAgent) Clone() Agent {
	r := &SearchSubAgent{
		cfg: this.cfg,
		cli: this.cli,
	}

	r.AddSystemMessage(SearchAgentSystemPrompt)
	return r
}

func (this *SearchSubAgent) Execute(ctx *Context, task *Task) (r *Result, err error) {
	fmt.Printf("\t 🔍 正在从互联网检索...\n")
	r = &Result{}

	query, ok := task.Parameters["query"].(string)
	if !ok {
		query = task.Description
	}

	var content string

	// 检索到的信息进行反思，最多反思 3 次
	for i := 0; i < 3; i++ {
		if content, err = this.SearchForTavily(query); err != nil {
			err = fmt.Errorf("网络检索发生异常: %v", err)
			return
		}

		util.IfDo(r.Output != "", func() { r.Output += "\n\n--- Additional Search Results ---\n" })
		r.Output += content
		this.AddUserMessage(fmt.Sprintf(SearchAgentUserPromptFormat, query, content))

		req := openai.ChatCompletionRequest{
			Model:       this.cfg.Model,
			Messages:    this.messages,
			Temperature: 0,
		}
		util.IfDo(this.cfg.Verbose, func() { LogStruct("SearchSubAgent LLM Request", req) })

		var resp openai.ChatCompletionResponse
		if resp, err = this.cli.CreateChatCompletion(context.Background(), req); err != nil {
			err = fmt.Errorf("LLM 请求发生异常: %v", err)
			return
		}
		util.IfDo(this.cfg.Verbose, func() { LogStruct("SearchSubAgent LLM Response", resp) })
		this.AddAssistantMessage(resp.Choices[0].Message.Content)

		llmResp := TrimLLMResp(resp.Choices[0].Message.Content)
		if strings.Contains(strings.ToUpper(llmResp), "SUFFICIENT") {
			fmt.Printf("\t 💬 检索完成，LLM 判定信息足以回答用户的查询\n")
			break
		}

		query = strings.TrimSpace(llmResp)
		fmt.Printf("\t 🔄 正在补充检索: %s\n", query)
	}

	return
}

func (this *SearchSubAgent) SearchForTavily(query string) (r string, err error) {
	b, _ := json.Marshal(map[string]interface{}{
		"query":          query,
		"search_depth":   "basic",
		"max_results":    20,
		"include_images": true,
	})

	var req *http.Request
	if req, err = http.NewRequest("POST", "https://api.tavily.com/search", bytes.NewBuffer(b)); err != nil {
		err = fmt.Errorf("failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", this.cfg.TavilyApiKey))

	var resp *http.Response
	if resp, err = (&http.Client{Timeout: 30 * time.Second}).Do(req); err != nil {
		err = fmt.Errorf("failed to perform Tavily search: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("Tavily API returned status %d: %s", resp.StatusCode, string(body))
		return
	}

	var result struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
		Images []string `json:"images"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		err = fmt.Errorf("failed to decode Tavily response: %v", err)
		return
	}

	var sb bytes.Buffer
	for _, item := range result.Results {
		sb.WriteString(fmt.Sprintf("Title: %s\nURL: %s\nContent: %s\n\n", item.Title, item.URL, item.Content))
	}

	if len(result.Images) > 0 {
		sb.WriteString("\nRelevant Images:\n")
		for _, imgURL := range result.Images {
			sb.WriteString(fmt.Sprintf("- Image URL: %s\n", imgURL))
		}
		sb.WriteString("\n")
	}

	if sb.Len() == 0 {
		err = fmt.Errorf("no results found")
		return
	}

	r = sb.String()
	util.IfDo(this.cfg.Verbose, func() { LogStruct("SearchSubAgent SearchForTavily Result", r) })
	return
}

func (this *SearchSubAgent) SearchForDuckDuckGo(query string) (r string, err error) {
	var req *http.Request
	if req, err = http.NewRequest("GET", fmt.Sprintf("https://api.duckduckgo.com/?format=json&q=%s", url.QueryEscape(query)), nil); err != nil {
		err = fmt.Errorf("failed to create request: %v", err)
		return
	}

	var resp *http.Response
	if resp, err = (&http.Client{Timeout: 10 * time.Second}).Do(req); err != nil {
		err = fmt.Errorf("failed to perform DuckDuckGo search: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("DuckDuckGo API returned status %d: %s", resp.StatusCode, string(body))
		return
	}

	var result struct {
		AbstractText  string `json:"AbstractText"`
		AbstractURL   string `json:"AbstractURL"`
		RelatedTopics []struct {
			Text string `json:"Text"`
			URL  string `json:"URL"`
		} `json:"RelatedTopics"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		err = fmt.Errorf("failed to decode DuckDuckGo response: %v", err)
		return
	}

	if result.AbstractText != "" {
		r = fmt.Sprintf("%s (Source: %s)", result.AbstractText, result.AbstractURL)
		util.IfDo(this.cfg.Verbose, func() { LogStruct("SearchSubAgent SearchForDuckDuckGo Result", r) })
		return
	}
	// 如果没有摘要，则回退到相关主题
	if len(result.RelatedTopics) > 0 {
		var topics []string
		for _, topic := range result.RelatedTopics {
			topics = append(topics, topic.Text)
		}
		r = fmt.Sprintf("No direct abstract found. Related topics: %s", strings.Join(topics, "; "))
		util.IfDo(this.cfg.Verbose, func() { LogStruct("SearchSubAgent SearchForDuckDuckGo Result", r) })
		return
	}

	err = fmt.Errorf("no results found")
	return
}

func (this *SearchSubAgent) SearchForWikipedia(query string) (r string, err error) {
	var req *http.Request
	if req, err = http.NewRequest("GET", fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&format=json&prop=extracts&exintro=&explaintext=&redirects=1&titles=%s", url.QueryEscape(query)), nil); err != nil {
		err = fmt.Errorf("failed to create request: %v", err)
		return
	}

	var resp *http.Response
	if resp, err = (&http.Client{Timeout: 10 * time.Second}).Do(req); err != nil {
		err = fmt.Errorf("failed to perform Wikipedia search: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("Wikipedia API returned status %d: %s", resp.StatusCode, string(body))
		return
	}
	var result struct {
		Query struct {
			Pages map[string]struct {
				Extract string `json:"extract"`
			} `json:"pages"`
		} `json:"query"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		err = fmt.Errorf("failed to decode Wikipedia response: %v", err)
		return
	}

	for _, page := range result.Query.Pages {
		if page.Extract != "" {
			// 清理一些常见的维基百科 API 伪影
			extract := strings.ReplaceAll(page.Extract, "(listen)", "")
			extract = strings.TrimSpace(extract)
			r = extract
			util.IfDo(this.cfg.Verbose, func() { LogStruct("SearchSubAgent SearchForWikipedia Result", r) })
			return
		}
	}

	err = fmt.Errorf("No relevant Wikipedia entry found.")
	return
}
