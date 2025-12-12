package main

import (
	"fmt"

	"github.com/ant-libs-go/ant-agent/agents"
)

var COMMANDS = make(map[string]func(ctx *agents.Context) (quit bool))

func init() {
	COMMANDS["\\help"] = func(ctx *agents.Context) bool {
		fmt.Println("\n📚 可用命令:")
		fmt.Println("  \\help    - 显示此帮助信息")
		fmt.Println("  \\clear   - 清除对话历史")
		fmt.Println("  \\podcast - 从上一份报告生成播客脚本")
		fmt.Println("  \\exit    - 退出聊天会话")
		fmt.Println("  \\quit    - 退出聊天会话")
		return false
	}

	COMMANDS["\\clear"] = func(ctx *agents.Context) bool {
		ctx.ClearChatHistory()
		fmt.Println("✨ 对话历史已清除")
		return false
	}

	COMMANDS["\\exit"] = func(ctx *agents.Context) bool {
		fmt.Println("👋 再见！")
		return true
	}

	COMMANDS["\\quit"] = func(ctx *agents.Context) bool {
		fmt.Println("👋 再见！")
		return true
	}
}
