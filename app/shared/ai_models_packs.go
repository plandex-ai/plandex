package shared

var DailyDriverModelPack ModelPack
var ReasoningModelPack ModelPack
var StrongModelPack ModelPack

var Sonnet4ModelPack ModelPack
var Opus4PlannerModelPack ModelPack

var OSSModelPack ModelPack
var CheapModelPack ModelPack
var AnthropicModelPack ModelPack
var OpenAIModelPack ModelPack

var GeminiPreviewModelPack ModelPack
var GeminiExperimentalModelPack ModelPack
var R1PlannerModelPack ModelPack
var PerplexityPlannerModelPack ModelPack

var BuiltInModelPacks = []*ModelPack{
	&DailyDriverModelPack,
	&ReasoningModelPack,
	&StrongModelPack,
	&CheapModelPack,
	&OSSModelPack,

	&Sonnet4ModelPack,
	&Opus4PlannerModelPack,

	&AnthropicModelPack,
	&OpenAIModelPack,

	&GeminiPreviewModelPack,
	&GeminiExperimentalModelPack,

	&R1PlannerModelPack,
	&PerplexityPlannerModelPack,
}

var DefaultModelPack *ModelPack = &DailyDriverModelPack

func getModelRoleConfig(role ModelRole, modelId ModelId, fns ...func(*ModelRoleConfigSchema)) ModelRoleConfigSchema {
	c := ModelRoleConfigSchema{
		ModelId:     modelId,
		Temperature: DefaultConfigByRole[role].Temperature,
		TopP:        DefaultConfigByRole[role].TopP,
	}
	for _, f := range fns {
		f(&c)
	}
	return c
}

func getLargeContextFallback(role ModelRole, modelId ModelId, fns ...func(*ModelRoleConfigSchema)) func(*ModelRoleConfigSchema) {
	return func(c *ModelRoleConfigSchema) {
		n := getModelRoleConfig(role, modelId)
		for _, f := range fns {
			f(&n)
		}
		c.LargeContextFallback = &n
	}
}

func getErrorFallback(role ModelRole, modelId ModelId, fns ...func(*ModelRoleConfigSchema)) func(*ModelRoleConfigSchema) {
	return func(c *ModelRoleConfigSchema) {
		n := getModelRoleConfig(role, modelId)
		for _, f := range fns {
			f(&n)
		}
		c.ErrorFallback = &n
	}
}

func getStrongModelFallback(role ModelRole, modelId ModelId, fns ...func(*ModelRoleConfigSchema)) func(*ModelRoleConfigSchema) {
	return func(c *ModelRoleConfigSchema) {
		n := getModelRoleConfig(role, modelId)
		for _, f := range fns {
			f(&n)
		}
		c.StrongModel = &n
	}
}

var (
	DailyDriverPackSchema        ModelPackSchema
	ReasoningPackSchema          ModelPackSchema
	StrongPackSchema             ModelPackSchema
	OSSModelPackSchema           ModelPackSchema
	CheapModelPackSchema         ModelPackSchema
	AnthropicPackSchema          ModelPackSchema
	OpenAIPackSchema             ModelPackSchema
	GeminiPreviewPackSchema      ModelPackSchema
	GeminiExperimentalPackSchema ModelPackSchema
	R1PlannerPackSchema          ModelPackSchema
	PerplexityPlannerPackSchema  ModelPackSchema
)

// An ordered list you can iterate over if you need to display / validate.
var BuiltInModelPackSchemas = []*ModelPackSchema{
	&DailyDriverPackSchema,
	&ReasoningPackSchema,
	&StrongPackSchema,
	&CheapModelPackSchema,
	&OSSModelPackSchema,
	&AnthropicPackSchema,
	&OpenAIPackSchema,
	&GeminiPreviewPackSchema,
	&GeminiExperimentalPackSchema,
	&R1PlannerPackSchema,
	&PerplexityPlannerPackSchema,
}

func init() {
	defaultBuilder := getModelRoleConfig(ModelRoleBuilder, "openai/o4-mini-medium",
		getStrongModelFallback(ModelRoleBuilder, "openai/o4-mini-high"),
	)

	DailyDriverPackSchema = ModelPackSchema{
		Name:        "daily-driver",
		Description: "A mix of models from Anthropic, OpenAI, and Google that balances speed, quality, and cost. Supports up to 2M context.",
		Planner: getModelRoleConfig(ModelRolePlanner, "anthropic/claude-3.7-sonnet",
			getLargeContextFallback(ModelRolePlanner, "google/gemini-2.5-pro-preview",
				getLargeContextFallback(ModelRolePlanner, "google/gemini-pro-1.5"),
			),
		),
		Architect: Pointer(getModelRoleConfig(ModelRoleArchitect, "anthropic/claude-3.7-sonnet",
			getLargeContextFallback(ModelRoleArchitect, "google/gemini-2.5-pro-preview",
				getLargeContextFallback(ModelRoleArchitect, "google/gemini-pro-1.5"),
			),
		)),
		Coder: Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-3.7-sonnet",
			getLargeContextFallback(ModelRoleCoder, "openai/gpt-4.1"),
		)),
		PlanSummary:      getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
		Builder:          defaultBuilder,
		WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder, "openai/o4-mini-medium")),
		Namer:            getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
		CommitMsg:        getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
		ExecStatus:       getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
	}

	ReasoningPackSchema = ModelPackSchema{
		Name:             "reasoning",
		Description:      "Like the daily driver, but uses 3.7-sonnet:thinking with reasoning enabled for planning and coding. Supports up to 160k input context.",
		Planner:          getModelRoleConfig(ModelRolePlanner, "anthropic/claude-3.7-sonnet-thinking-hidden"),
		Coder:            Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-3.7-sonnet-thinking-hidden")),
		PlanSummary:      getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
		Builder:          defaultBuilder,
		WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder, "openai/o4-mini-medium")),
		Namer:            getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
		CommitMsg:        getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
		ExecStatus:       getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
	}

	StrongPackSchema = ModelPackSchema{
		Name:             "strong",
		Description:      "For difficult tasks where slower responses and builds are ok. Uses o3-high for architecture and planning, claude-3.7-sonnet thinking for implementation, prioritizes reliability over speed for builds. Supports up to 160k input context.",
		Planner:          getModelRoleConfig(ModelRolePlanner, "openai/o3-high"),
		Architect:        Pointer(getModelRoleConfig(ModelRoleArchitect, "openai/o3-high")),
		Coder:            Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-3.7-sonnet-thinking-hidden")),
		PlanSummary:      getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
		Builder:          getModelRoleConfig(ModelRoleBuilder, "openai/o4-mini-high"),
		WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder, "openai/o4-mini-high")),
		Namer:            getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
		CommitMsg:        getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
		ExecStatus:       getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-medium"),
	}

	CheapModelPackSchema = ModelPackSchema{
		Name:             "cheap",
		Description:      "Cost-effective models that can still get the job done for easier tasks. Supports up to 160k context. Uses OpenAI's o4-mini model for planning, GPT-4.1 for coding, and GPT-4.1 Mini for lighter tasks.",
		Planner:          getModelRoleConfig(ModelRolePlanner, "openai/o4-mini-medium"),
		Coder:            Pointer(getModelRoleConfig(ModelRoleCoder, "openai/gpt-4.1")),
		PlanSummary:      getModelRoleConfig(ModelRolePlanSummary, "openai/gpt-4.1-mini"),
		Builder:          getModelRoleConfig(ModelRoleBuilder, "openai/o4-mini-low"),
		WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder, "openai/o4-mini-low")),
		Namer:            getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
		CommitMsg:        getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
		ExecStatus:       getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
	}

	OSSModelPackSchema = ModelPackSchema{
		Name:             "oss",
		Description:      "An experimental mix of the best open source models for coding. Supports up to 56k context, 8k per file. Works best with smaller projects and files. Includes reasoning.",
		Planner:          getModelRoleConfig(ModelRolePlanner, "deepseek/r1-reasoning-visible"),
		Coder:            Pointer(getModelRoleConfig(ModelRoleCoder, "deepseek/v3-0324")),
		PlanSummary:      getModelRoleConfig(ModelRolePlanSummary, "deepseek/r1-reasoning-hidden"),
		Builder:          getModelRoleConfig(ModelRoleBuilder, "deepseek/r1-reasoning-hidden"),
		WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder, "deepseek/r1-reasoning-hidden")),
		Namer:            getModelRoleConfig(ModelRoleName, "qwen/qwen-2.5-coder-32b-instruct"),
		CommitMsg:        getModelRoleConfig(ModelRoleCommitMsg, "qwen/qwen-2.5-coder-32b-instruct"),
		ExecStatus:       getModelRoleConfig(ModelRoleExecStatus, "deepseek/r1-reasoning-hidden"),
	}

	OpenAIPackSchema = ModelPackSchema{
		Name:             "openai",
		Description:      "OpenAI blend. Supports up to 1M context. Uses OpenAI's GPT-4.1 model for heavy lifting, GPT-4.1 Mini for lighter tasks.",
		Planner:          getModelRoleConfig(ModelRolePlanner, "openai/gpt-4.1"),
		PlanSummary:      getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
		Builder:          defaultBuilder,
		WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder, "openai/o4-mini-medium")),
		Namer:            getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
		CommitMsg:        getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
		ExecStatus:       getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
	}

	AnthropicPackSchema = ModelPackSchema{
		Name:             "anthropic",
		Description:      "Anthropic blend. Supports up to 180k context. Uses Claude 3.5 Sonnet for heavy lifting, Claude 3 Haiku for lighter tasks.",
		Planner:          getModelRoleConfig(ModelRolePlanner, "anthropic/claude-3.7-sonnet"),
		Coder:            Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-3.7-sonnet")),
		PlanSummary:      getModelRoleConfig(ModelRolePlanSummary, "anthropic/claude-3.5-haiku"),
		Builder:          getModelRoleConfig(ModelRoleBuilder, "anthropic/claude-3.7-sonnet"),
		WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder, "anthropic/claude-3.7-sonnet")),
		Namer:            getModelRoleConfig(ModelRoleName, "anthropic/claude-3.5-haiku"),
		CommitMsg:        getModelRoleConfig(ModelRoleCommitMsg, "anthropic/claude-3.5-haiku"),
		ExecStatus:       getModelRoleConfig(ModelRoleExecStatus, "anthropic/claude-3.7-sonnet"),
	}

	GeminiPreviewPackSchema = ModelPackSchema{
		Name:             "gemini-preview",
		Description:      "Uses Gemini 2.5 Pro Preview for planning and coding, default models for other roles. Supports up to 1M input context.",
		Planner:          getModelRoleConfig(ModelRolePlanner, "google/gemini-2.5-pro-preview"),
		Coder:            Pointer(getModelRoleConfig(ModelRoleCoder, "google/gemini-2.5-pro-preview")),
		PlanSummary:      getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
		Builder:          defaultBuilder,
		WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder, "openai/o4-mini-medium")),
		Namer:            getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
		CommitMsg:        getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
		ExecStatus:       getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
	}

	GeminiExperimentalPackSchema = ModelPackSchema{
		Name:             "gemini-exp",
		Description:      "Uses Gemini 2.5 Pro Experimental (free) for planning and coding, default models for other roles. Supports up to 1M input context.",
		Planner:          getModelRoleConfig(ModelRolePlanner, "google/gemini-2.5-pro-exp"),
		Coder:            Pointer(getModelRoleConfig(ModelRoleCoder, "google/gemini-2.5-pro-exp")),
		PlanSummary:      getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
		Builder:          defaultBuilder,
		WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder, "openai/o4-mini-medium")),
		Namer:            getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
		CommitMsg:        getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
		ExecStatus:       getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
	}

	R1PlannerPackSchema = ModelPackSchema{
		Name:             "r1-planner",
		Description:      "Uses DeepSeek R1 for planning, Qwen for light tasks, and default models for implementation. Supports up to 56k input context.",
		Planner:          getModelRoleConfig(ModelRolePlanner, "deepseek/r1-reasoning-visible"),
		Coder:            Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-3.7-sonnet")),
		PlanSummary:      getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
		Builder:          getModelRoleConfig(ModelRoleBuilder, "openai/o4-mini-medium"),
		WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder, "openai/o4-mini-low")),
		Namer:            getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
		CommitMsg:        getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
		ExecStatus:       getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-medium"),
	}

	PerplexityPlannerPackSchema = ModelPackSchema{
		Name:             "perplexity-planner",
		Description:      "Uses Perplexity Sonar for planning, Qwen for light tasks, and default models for implementation. Supports up to 97k input context.",
		Planner:          getModelRoleConfig(ModelRolePlanner, "perplexity/sonar-reasoning-visible"),
		Coder:            Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-3.7-sonnet")),
		PlanSummary:      getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
		Builder:          getModelRoleConfig(ModelRoleBuilder, "openai/o4-mini-medium"),
		WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder, "openai/o4-mini-low")),
		Namer:            getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
		CommitMsg:        getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
		ExecStatus:       getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-medium"),
	}

	DailyDriverModelPack = DailyDriverPackSchema.ToModelPack()
	ReasoningModelPack = ReasoningPackSchema.ToModelPack()
	StrongModelPack = StrongPackSchema.ToModelPack()
	CheapModelPack = CheapModelPackSchema.ToModelPack()
	OSSModelPack = OSSModelPackSchema.ToModelPack()
	AnthropicModelPack = AnthropicPackSchema.ToModelPack()
	OpenAIModelPack = OpenAIPackSchema.ToModelPack()
	GeminiPreviewModelPack = GeminiPreviewPackSchema.ToModelPack()
	GeminiExperimentalModelPack = GeminiExperimentalPackSchema.ToModelPack()
	R1PlannerModelPack = R1PlannerPackSchema.ToModelPack()
	PerplexityPlannerModelPack = PerplexityPlannerPackSchema.ToModelPack()

	// Preserve the old slices/vars
	BuiltInModelPacks = []*ModelPack{
		&DailyDriverModelPack,
		&ReasoningModelPack,
		&StrongModelPack,
		&CheapModelPack,
		&OSSModelPack,
		&AnthropicModelPack,
		&OpenAIModelPack,
		&GeminiPreviewModelPack,
		&GeminiExperimentalModelPack,
		&R1PlannerModelPack,
		&PerplexityPlannerModelPack,
	}

	DefaultModelPack = &DailyDriverModelPack

}
