package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	antagent "github.com/ant-libs-go/ant-agent"
	"github.com/ant-libs-go/ant-agent/agents"
	"github.com/ant-libs-go/ant-agent/mcps"
	"github.com/ant-libs-go/ant-agent/skills"
	"github.com/ant-libs-go/util"
	"github.com/urfave/cli/v3"
)

func main() {
	cfg := &antagent.Config{}

	app := &cli.Command{
		Name:  "deepresearch",
		Usage: `Ant Deep Research CLI 是一个实现深度研究架构的命令行工具`,
		Flags: antagent.DefaultCliFlags(cfg),
		Action: func(c context.Context, cmd *cli.Command) (err error) {
			antagent.PrintLogo()
			fmt.Println(strings.Repeat("-", 60))

			util.IfDo(cfg.Verbose, func() { fmt.Printf("🧩 尝试初始化 MCP 配置\n") })
			mcpClient, err := mcps.NewMcpClient("./mcp.json")
			if err != nil {
				fmt.Printf("‼️ MCP 配置加载失败，如有必要请检查: %v\n", err)
			} else {
				util.IfDo(cfg.Verbose, func() { fmt.Printf("👍 MCP 配置初始化成功\n") })
			}

			util.IfDo(cfg.Verbose, func() { fmt.Printf("🧩 尝试初始化 SKILL 配置\n") })
			skillClient, err := skills.NewSkillClient(cfg.SkillsDir)
			if err != nil {
				fmt.Printf("‼️ SKILL 配置加载失败，如有必要请检查: %v\n", err)
			} else {
				util.IfDo(cfg.Verbose, func() { fmt.Printf("👍 SKILL 配置初始化成功\n") })
			}

			ctx := &agents.Context{
				Offset:    0,
				Tasks:     make([]*agents.Task, 0, 10),
				McpClient: mcpClient,
			}

			for {
				agent := agents.NewPlanningAgent(cfg,
					[]agents.Agent{
						agents.NewSearchSubAgent(cfg),
						agents.NewAnalyzeSubAgent(cfg),
						agents.NewReportSubAgent(cfg),
						//agents.NewPPTSubAgent(cfg)
						agents.NewRenderSubAgent(cfg),
					},
					skillClient.GetSkills())

				ctx.Input, err = antagent.GetInput()
				if err != nil {
					fmt.Printf("‼️ 获用户输入失败: %v\n", err)
					continue
				}
				if len(ctx.Input) == 0 {
					continue
				}

				if _, ok := COMMANDS[ctx.Input]; ok {
					if quit := COMMANDS[ctx.Input](ctx); quit {
						return nil
					}
					continue
				}

				result, err := agent.Execute(ctx, nil)
				if err != nil {
					fmt.Printf("‼️ %v\n", err)
					continue
				}

				if len(result.Tasks) == 0 {
					fmt.Printf("💬 LLM 判定无需进行任务规划，将直接回复：\n")
					fmt.Printf("%s\n", result.Output)
					continue
				}

				ctx.Tasks = result.Tasks
				ctx.Plans = result.Output

				for ctx.Offset = 0; ctx.Offset < len(ctx.Tasks); ctx.Offset++ {
					fmt.Printf("📍 步骤 %d/%d: [%s] %s\n", ctx.Offset+1, len(ctx.Tasks), ctx.Tasks[ctx.Offset].Name, ctx.Tasks[ctx.Offset].Description)
					var subagent agents.Agent

					skill := agent.GetSkill(ctx.Tasks[ctx.Offset].Name)
					if skill != nil {
						subagent = agents.NewSkillSubAgent(cfg, skill)
					} else {
						subagent = agent.GetSubAgent(ctx.Tasks[ctx.Offset].Name)
					}
					if subagent == nil {
						fmt.Printf("‼️ SubAgent[%s]未找到，请检查是否正确配置\n", ctx.Tasks[ctx.Offset].Name)
						continue
					}
					result, err := subagent.Execute(ctx, ctx.Tasks[ctx.Offset])
					if err != nil {
						fmt.Printf("‼️ %v\n", err)
						continue
					}
					// 动态规划
					if len(result.Tasks) > 0 {
						fmt.Printf("🔄 动态规划更新: 插入 %d 个新任务\n", len(result.Tasks))
						rear := append([]*agents.Task{}, ctx.Tasks[ctx.Offset+1:]...)
						ctx.Tasks = append(ctx.Tasks[:ctx.Offset+1], append(result.Tasks, rear...)...)
					}
					// 保留 subagent 的输出结果
					ctx.Tasks[ctx.Offset].Output = result.Output

					fmt.Printf("👍 任务运行成功，进度 %d/%d\n", ctx.Offset+1, len(ctx.Tasks))
				}

				fmt.Printf("\n📄 最终报告:\n")
				fmt.Printf("%s\n", ctx.Tasks[len(ctx.Tasks)-1].Output)
			}
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
