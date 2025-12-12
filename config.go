package antagent

import (
	"github.com/urfave/cli/v3"
)

type Config struct {
	Model        string
	ApiBase      string
	ApiKey       string
	AutoApprove  bool
	Verbose      bool
	TavilyApiKey string
	SkillsDir    string
}

func DefaultCliFlags(config *Config) (r []cli.Flag) {
	return []cli.Flag{
		&cli.StringFlag{
			Name: "model", Usage: "OpenAI-compatible model name (falls back to OPENAI_MODEL env var)",
			Required:    true,
			Aliases:     []string{"m"},
			Sources:     cli.EnvVars("OPENAI_MODEL"),
			Destination: &config.Model,
		},
		&cli.StringFlag{
			Name: "api-base", Usage: "OpenAI-compatible API base URL (falls back to OPENAI_API_BASE env var)",
			Required:    true,
			Aliases:     []string{"b"},
			Sources:     cli.EnvVars("OPENAI_API_BASE"),
			Destination: &config.ApiBase,
		},
		&cli.StringFlag{
			Name: "api-key", Usage: "OpenAI-compatible API key (falls back to OPENAI_API_KEY env var)",
			Required:    true,
			Aliases:     []string{"k"},
			Sources:     cli.EnvVars("OPENAI_API_KEY"),
			Destination: &config.ApiKey,
		},
		&cli.BoolFlag{
			Name: "auto-approve", Usage: "Auto-approve all tool calls (WARNING: potentially unsafe)",
			Required:    false,
			Destination: &config.AutoApprove,
		},
		&cli.BoolFlag{
			Name: "verbose", Usage: "Enable verbose output",
			Required:    false,
			Aliases:     []string{"v"},
			Destination: &config.Verbose,
		},
		&cli.StringFlag{
			Name: "tavily-api-key", Usage: "Tavily API key (falls back to TAVILY_API_KEY env var)",
			Required:    false,
			Sources:     cli.EnvVars("TAVILY_API_KEY"),
			Destination: &config.TavilyApiKey,
		},
		&cli.StringFlag{
			Name: "skills-dir", Usage: "Skills directory (falls back to SKILLS_DIR env var)",
			Required:    false,
			Sources:     cli.EnvVars("SKILLS_DIR"),
			Destination: &config.SkillsDir,
		},
	}
}
