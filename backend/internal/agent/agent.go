// Package agent builds the LLM agent(s): model binding, generation config,
// system instruction, and tool/skill/workspace/MCP/knowledge wiring, plus the
// multi-agent team.
package agent

import (
	"trpc.group/trpc-go/trpc-agent-go/agent"
	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/codeexecutor"
	"trpc.group/trpc-go/trpc-agent-go/knowledge"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/skill"
	"trpc.group/trpc-go/trpc-agent-go/team"
	"trpc.group/trpc-go/trpc-agent-go/tool"

	"github.com/9Ashwin/trpc-agent-lab/backend/internal/config"
)

// Deps bundles the optional capabilities wired into the agent. Any field may
// be nil/empty to disable the corresponding capability.
type Deps struct {
	ExtraTools []tool.Tool
	ToolSets   []tool.ToolSet
	Repo       skill.Repository
	Exec       codeexecutor.CodeExecutor
	Knowledge  knowledge.Knowledge
}

// defaultInstruction is the system prompt for the single/coordinator agent.
const defaultInstruction = `你是一个乐于助人的中文助手，运行在一个有工具调用能力的 Agent 框架里。

能力约定：
- 计算用 calculator，取时间用 get_current_time。
- 你拥有长期记忆工具（memory_add / memory_search / memory_update / memory_delete）。
  用户告诉你值得记住的信息（偏好、事实、约定）时主动 memory_add；需要回忆时先 memory_search。
- 你拥有 Agent Skills（skill_load / skill_list_docs / skill_select_docs）。
  当任务匹配某个技能时，加载并按其说明执行。
- 你拥有一个隔离的 workspace（workspace_exec 运行命令/代码）。
  产出文件写到 $OUTPUT_DIR（out/），输入放 $WORK_DIR/inputs，技能文件经 $SKILLS_DIR 引用。
- 你可能还接入了外部 MCP 工具与知识库，按需调用。
- 回答简洁、准确，中文优先。`

// coordinatorInstruction is the coordinator's system prompt in team mode.
const coordinatorInstruction = `你是团队的协调者（coordinator）。你手下有一组专家成员：
- researcher：负责搜索、调研、总结资料，给出有依据的结论。
- coder：负责代码、技术架构、实现方案的编写与评审。
- writer：负责文案、写作、翻译、润色。

根据用户需求调度合适的专家，整合他们的产出，给出最终答案。
你同样拥有记忆、技能与 workspace 能力，可按需使用。回答中文优先。`

// Build constructs the single DeepSeek-backed LLM agent.
func Build(cfg *config.Config, deps Deps) agent.Agent {
	return buildLLMAgent(cfg, deps, "deepseek-agent", defaultInstruction, "A helpful Chinese assistant with tools, memory, skills and a workspace.")
}

// BuildTeam constructs a coordinator team: one coordinator plus specialist
// members (researcher / coder / writer). The coordinator carries the tools,
// memory, skills and workspace; members are lightweight specialists.
func BuildTeam(cfg *config.Config, deps Deps) agent.Agent {
	coordinator := buildLLMAgent(cfg, deps, "coordinator", coordinatorInstruction, "Coordinates a small team of specialists.")

	members := []agent.Agent{
		buildMember(cfg, "researcher", "Search, research and summarize with evidence.", "你是研究员，负责搜索、调研、总结资料，给出有依据的结论。中文优先。"),
		buildMember(cfg, "coder", "Write and review code, architecture and implementation plans.", "你是技术专家，负责代码、架构、实现方案的编写与评审。中文优先。"),
		buildMember(cfg, "writer", "Write, translate and polish copy and prose.", "你是写作者，负责文案、写作、翻译、润色。中文优先。"),
	}

	t, err := team.New(coordinator, members)
	if err != nil {
		// Fall back to the single coordinator if team assembly fails.
		return coordinator
	}
	return t
}

// buildLLMAgent is the shared constructor for the coordinator and single agent.
func buildLLMAgent(cfg *config.Config, deps Deps, name, instruction, description string) *llmagent.LLMAgent {
	modelInstance := openai.New(cfg.Model,
		openai.WithVariant(openai.VariantDeepSeek),
		openai.WithAPIKey(cfg.APIKey),
	)

	genConfig := model.GenerationConfig{
		MaxTokens:   intPtr(2048),
		Temperature: floatPtr(0.7),
		Stream:      true,
	}

	opts := []llmagent.Option{
		llmagent.WithModel(modelInstance),
		llmagent.WithGenerationConfig(genConfig),
		llmagent.WithTools(deps.ExtraTools),
		llmagent.WithInstruction(instruction),
		llmagent.WithDescription(description),
	}
	if len(deps.ToolSets) > 0 {
		opts = append(opts, llmagent.WithToolSets(deps.ToolSets))
	}
	if deps.Repo != nil {
		opts = append(opts, llmagent.WithSkills(deps.Repo))
	}
	if deps.Exec != nil {
		opts = append(opts,
			llmagent.WithCodeExecutor(deps.Exec),
			// Route code execution through workspace_exec instead of scraping
			// fenced ```sh blocks from prose.
			llmagent.WithEnableCodeExecutionResponseProcessor(false),
		)
	}
	if deps.Knowledge != nil {
		opts = append(opts, llmagent.WithKnowledge(deps.Knowledge))
	}

	return llmagent.New(name, opts...)
}

// buildMember constructs a lightweight specialist member (no tools/skills).
func buildMember(cfg *config.Config, name, description, instruction string) *llmagent.LLMAgent {
	modelInstance := openai.New(cfg.Model,
		openai.WithVariant(openai.VariantDeepSeek),
		openai.WithAPIKey(cfg.APIKey),
	)
	genConfig := model.GenerationConfig{
		MaxTokens:   intPtr(2048),
		Temperature: floatPtr(0.7),
		Stream:      true,
	}
	return llmagent.New(
		name,
		llmagent.WithModel(modelInstance),
		llmagent.WithGenerationConfig(genConfig),
		llmagent.WithDescription(description),
		llmagent.WithInstruction(instruction),
	)
}

func intPtr(i int) *int           { return &i }
func floatPtr(f float64) *float64 { return &f }
