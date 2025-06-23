package shared

var DailyDriverModelPack ModelPack
var ReasoningModelPack ModelPack
var StrongModelPack ModelPack
var OSSModelPack ModelPack
var CheapModelPack ModelPack

var OpusPlannerModelPack ModelPack
var StrongModelOpus ModelPack

var AnthropicModelPack ModelPack
var OpenAIModelPack ModelPack

var GeminiPreviewModelPack ModelPack
var GeminiExperimentalModelPack ModelPack
var R1PlannerModelPack ModelPack
var PerplexityPlannerModelPack ModelPack

var OllamaExperimentalModelPack ModelPack
var OllamaAdaptiveModelPack ModelPack

var BuiltInModelPacks = []*ModelPack{
	&DailyDriverModelPack,
	&ReasoningModelPack,
	&StrongModelPack,
	&CheapModelPack,
	&OSSModelPack,
	&OllamaExperimentalModelPack,
	&OllamaAdaptiveModelPack,
	&OpusPlannerModelPack,
	&StrongModelOpus,
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
		ModelId: modelId,
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
	DailyDriverSchema        ModelPackSchema
	ReasoningSchema          ModelPackSchema
	StrongSchema             ModelPackSchema
	OssSchema                ModelPackSchema
	CheapSchema              ModelPackSchema
	OllamaExperimentalSchema ModelPackSchema
	OllamaAdaptiveSchema     ModelPackSchema
	AnthropicSchema          ModelPackSchema
	OpenAISchema             ModelPackSchema
	GeminiPreviewSchema      ModelPackSchema
	GeminiExperimentalSchema ModelPackSchema
	R1PlannerSchema          ModelPackSchema
	PerplexityPlannerSchema  ModelPackSchema
)

var BuiltInModelPackSchemas = []*ModelPackSchema{
	&DailyDriverSchema,
	&ReasoningSchema,
	&StrongSchema,
	&CheapSchema,
	&OssSchema,
	&OllamaExperimentalSchema,
	&OllamaAdaptiveSchema,
	&AnthropicSchema,
	&OpenAISchema,
	&GeminiPreviewSchema,
	&GeminiExperimentalSchema,
	&R1PlannerSchema,
	&PerplexityPlannerSchema,
}

func init() {
	defaultBuilder := getModelRoleConfig(ModelRoleBuilder, "openai/o4-mini-medium",
		getStrongModelFallback(ModelRoleBuilder, "openai/o4-mini-high"),
	)

	DailyDriverSchema = ModelPackSchema{
		Name:        "daily-driver",
		Description: "A mix of models from Anthropic, OpenAI, and Google that balances speed, quality, and cost. Supports up to 2M context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner: getModelRoleConfig(ModelRolePlanner, "anthropic/claude-sonnet-4",
				getLargeContextFallback(ModelRolePlanner, "google/gemini-2.5-pro-preview",
					getLargeContextFallback(ModelRolePlanner, "google/gemini-pro-1.5"),
				),
			),
			Architect: Pointer(getModelRoleConfig(ModelRoleArchitect, "anthropic/claude-sonnet-4",
				getLargeContextFallback(ModelRoleArchitect, "google/gemini-2.5-pro-preview",
					getLargeContextFallback(ModelRoleArchitect, "google/gemini-pro-1.5"),
				),
			)),
			Coder: Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4",
				getLargeContextFallback(ModelRoleCoder, "openai/gpt-4.1"),
			)),
			PlanSummary:      getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:          defaultBuilder,
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder, "openai/o4-mini-medium")),
			Namer:            getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:        getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus:       getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
		},
	}

	ReasoningSchema = ModelPackSchema{
		Name:        "reasoning",
		Description: "Like the daily driver, but uses sonnet-4-thinking with reasoning enabled for planning and coding. Supports up to 160k input context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "anthropic/claude-sonnet-4-thinking-hidden"),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4-thinking-hidden")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     defaultBuilder,
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-medium")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
		},
	}

	StrongSchema = ModelPackSchema{
		Name:        "strong",
		Description: "For difficult tasks where slower responses and builds are ok. Uses o3-high for architecture and planning, claude-sonnet-4 thinking for implementation. Supports up to 160k input context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "openai/o3-high"),
			Architect:   Pointer(getModelRoleConfig(ModelRoleArchitect, "openai/o3-high")),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4-thinking-hidden")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     getModelRoleConfig(ModelRoleBuilder, "openai/o4-mini-high"),
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-high")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-medium"),
		},
	}

	CheapSchema = ModelPackSchema{
		Name:        "cheap",
		Description: "Cost-effective models that can still get the job done for easier tasks. Supports up to 160k context. Uses OpenAI's o4-mini model for planning, GPT-4.1 for coding, and GPT-4.1 Mini for lighter tasks.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "openai/o4-mini-medium"),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "openai/gpt-4.1")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/gpt-4.1-mini"),
			Builder:     getModelRoleConfig(ModelRoleBuilder, "openai/o4-mini-low"),
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-low")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
		},
	}

	OssSchema = ModelPackSchema{
		Name:        "oss",
		Description: "An experimental mix of the best open source models for coding. Supports up to 56k context, 8k per file. Works best with smaller projects and files. Includes reasoning.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "deepseek/r1"),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "deepseek/v3-0324")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "deepseek/r1-hidden"),
			Builder:     getModelRoleConfig(ModelRoleBuilder, "deepseek/r1-hidden"),
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"deepseek/r1-hidden")),
			Namer:      getModelRoleConfig(ModelRoleName, "qwen/qwen-2.5-coder-32b-instruct"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "qwen/qwen-2.5-coder-32b-instruct"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "deepseek/r1-hidden"),
		},
	}

	OllamaExperimentalSchema = ModelPackSchema{
		Name:        "ollama-experimental",
		Description: "Ollama experimental local blend. Supports up to 110k context. Uses Qwen3-32b model for heavy lifting, Qwen3-14b for summaries, Qwen3-8b for lighter tasks.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			LocalProvider: ModelProviderOllama,
			Planner:       getModelRoleConfig(ModelRolePlanner, "mistral/devstral-small"),
			PlanSummary:   getModelRoleConfig(ModelRolePlanSummary, "qwen/qwen3-14b"),
			Builder:       getModelRoleConfig(ModelRoleBuilder, "mistral/devstral-small"),
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"mistral/devstral-small")),
			Namer:      getModelRoleConfig(ModelRoleName, "qwen/qwen3-8b"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "qwen/qwen3-8b"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "mistral/devstral-small"),
		},
	}

	OllamaAdaptiveSchema = ModelPackSchema{
		Name:        "ollama-adaptive",
		Description: "Ollama adaptive blend. Uses local models for planning and context selection, cloud models for implementation and file edits. Supports up to 110k context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			LocalProvider: ModelProviderOllama,
			Planner:       getModelRoleConfig(ModelRolePlanner, "mistral/devstral-small"),
			PlanSummary:   getModelRoleConfig(ModelRolePlanSummary, "qwen/qwen3-14b"),
			Builder:       getModelRoleConfig(ModelRoleBuilder, "mistral/devstral-small"),
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"mistral/devstral-small")),
			Namer:      getModelRoleConfig(ModelRoleName, "qwen/qwen3-8b"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "qwen/qwen3-8b"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "mistral/devstral-small"),
		},
	}

	OpenAISchema = ModelPackSchema{
		Name:        "openai",
		Description: "OpenAI blend. Supports up to 1M context. Uses OpenAI's GPT-4.1 model for heavy lifting, GPT-4.1 Mini for lighter tasks.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "openai/gpt-4.1"),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     defaultBuilder,
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-medium")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
		},
	}

	AnthropicSchema = ModelPackSchema{
		Name:        "anthropic",
		Description: "Anthropic blend. Supports up to 180k context. Uses Claude Sonnet 4 for heavy lifting, Claude 3 Haiku for lighter tasks.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "anthropic/claude-sonnet-4"),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "anthropic/claude-3.5-haiku"),
			Builder:     getModelRoleConfig(ModelRoleBuilder, "anthropic/claude-sonnet-4"),
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"anthropic/claude-sonnet-4")),
			Namer:      getModelRoleConfig(ModelRoleName, "anthropic/claude-3.5-haiku"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "anthropic/claude-3.5-haiku"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "anthropic/claude-sonnet-4"),
		},
	}

	GeminiPreviewSchema = ModelPackSchema{
		Name:        "gemini-preview",
		Description: "Uses Gemini 2.5 Pro Preview for planning and coding, default models for other roles. Supports up to 1M input context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "google/gemini-2.5-pro-preview"),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "google/gemini-2.5-pro-preview")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     defaultBuilder,
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-medium")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
		},
	}

	GeminiExperimentalSchema = ModelPackSchema{
		Name:        "gemini-exp",
		Description: "Uses Gemini 2.5 Pro Experimental (free) for planning and coding, default models for other roles. Supports up to 1M input context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "google/gemini-2.5-pro-exp"),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "google/gemini-2.5-pro-exp")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     defaultBuilder,
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-medium")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
		},
	}

	R1PlannerSchema = ModelPackSchema{
		Name:        "r1-planner",
		Description: "Uses DeepSeek R1 for planning, Qwen for light tasks, and default models for implementation. Supports up to 56k input context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "deepseek/r1"),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     getModelRoleConfig(ModelRoleBuilder, "openai/o4-mini-medium"),
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-low")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-medium"),
		},
	}

	PerplexityPlannerSchema = ModelPackSchema{
		Name:        "perplexity-planner",
		Description: "Uses Perplexity Sonar for planning, Qwen for light tasks, and default models for implementation. Supports up to 97k input context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "perplexity/sonar-reasoning"),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     getModelRoleConfig(ModelRoleBuilder, "openai/o4-mini-medium"),
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-low")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-medium"),
		},
	}

	DailyDriverModelPack = DailyDriverSchema.ToModelPack()
	ReasoningModelPack = ReasoningSchema.ToModelPack()
	StrongModelPack = StrongSchema.ToModelPack()
	CheapModelPack = CheapSchema.ToModelPack()
	OSSModelPack = OssSchema.ToModelPack()
	OllamaExperimentalModelPack = OllamaExperimentalSchema.ToModelPack()
	OllamaAdaptiveModelPack = OllamaAdaptiveSchema.ToModelPack()
	AnthropicModelPack = AnthropicSchema.ToModelPack()
	OpenAIModelPack = OpenAISchema.ToModelPack()
	GeminiPreviewModelPack = GeminiPreviewSchema.ToModelPack()
	GeminiExperimentalModelPack = GeminiExperimentalSchema.ToModelPack()
	R1PlannerModelPack = R1PlannerSchema.ToModelPack()
	PerplexityPlannerModelPack = PerplexityPlannerSchema.ToModelPack()

	BuiltInModelPacks = []*ModelPack{
		&DailyDriverModelPack,
		&ReasoningModelPack,
		&StrongModelPack,
		&CheapModelPack,
		&OSSModelPack,
		&OllamaExperimentalModelPack,
		&OllamaAdaptiveModelPack,
		&AnthropicModelPack,
		&OpenAIModelPack,
		&GeminiPreviewModelPack,
		&GeminiExperimentalModelPack,
		&R1PlannerModelPack,
		&PerplexityPlannerModelPack,
	}

	DefaultModelPack = &DailyDriverModelPack

	for _, mp := range BuiltInModelPacks {
		for _, id := range mp.ToModelPackSchema().AllModelIds() {
			if BuiltInBaseModelsById[id] == nil {
				panic("missing base model: " + id)
			}
		}
	}

}
