package i18n

// Key identifies a message in the locale catalogue. Keys are flat dot paths:
// the JSON catalogue is nested, but code always uses the flattened form.
type Key string

const (
	// Task lifecycle messages.
	MsgTaskAccepted         Key = "task.accepted"
	MsgTaskAcceptedPipeline Key = "task.acceptedPipeline"
	MsgTaskAcceptedPlanner  Key = "task.acceptedPlanner"
	MsgTaskCancelled        Key = "task.cancelled"
	MsgTaskCancelledParent  Key = "task.cancelledParent"
	MsgTaskRetried          Key = "task.retried"
	MsgTaskMoved            Key = "task.moved"
	MsgTaskAnswered         Key = "task.answered"
	MsgTaskStagesDone       Key = "task.stagesDone"
	MsgTaskStagesPending    Key = "task.stagesPending"
	MsgTaskResumed          Key = "task.resumed"
	MsgTaskNeedsHuman       Key = "task.needsHuman"
	MsgTaskRetryFailed      Key = "task.retryFailed"
	MsgTaskDependencyGone   Key = "task.dependencyMissing"
	MsgTaskStageFailed      Key = "task.stageFailed"
	MsgTaskStageNeedsHuman  Key = "task.stageNeedsHuman"
	MsgTaskStageTitle       Key = "task.stageTitle"
	MsgTaskStageError       Key = "task.stageError"

	// Planning and execution messages.
	MsgTaskPlanReady       Key = "task.planReady"
	MsgTaskPlanFallback    Key = "task.planFallback"
	MsgTaskExecuting       Key = "task.executing"
	MsgTaskExecutionDone   Key = "task.executionDone"
	MsgTaskVerdictPass     Key = "task.verdictPass"
	MsgTaskVerified        Key = "task.verified"
	MsgTaskVerdictRejected Key = "task.verdictRejected"
	MsgTaskReviewRejected  Key = "task.reviewRejected"
	MsgTaskAggregated      Key = "task.aggregated"
	MsgTaskAggregatedBy    Key = "task.aggregatedBy"
	MsgTaskAggregatedNoLLM Key = "task.aggregatedNoModel"
	MsgTaskAggregatedGlued Key = "task.aggregatedFallback"
	MsgPlanCreated         Key = "task.planCreated"

	// Task state labels: models.StatusKey maps a status code to
	// "task.state.<code>". Enum-driven — one key per models.Status* constant.
	TaskStatePending    Key = "task.state.pending"
	TaskStatePlanning   Key = "task.state.planning"
	TaskStateBlocked    Key = "task.state.blocked"
	TaskStateRunning    Key = "task.state.running"
	TaskStateVerifying  Key = "task.state.verifying"
	TaskStateRetrying   Key = "task.state.retrying"
	TaskStateNeedsHuman Key = "task.state.needs_human"
	TaskStateDone       Key = "task.state.done"
	TaskStateFailed     Key = "task.state.failed"
	TaskStateCancelled  Key = "task.state.cancelled"

	// Shared-context entry titles.
	MsgContextHumanNote Key = "context.entry.humanNote"
	MsgContextPlan      Key = "context.entry.plan"

	// Errors.
	ErrTitleRequired    Key = "error.titleRequired"
	ErrRoleModelNeeded  Key = "error.roleModelRequired"
	ErrUnknownPipeline  Key = "error.unknownPipeline"
	ErrNoModelForRole   Key = "error.noModelForRole"
	ErrEmptyChain       Key = "error.emptyChain"
	ErrBudgetExhausted  Key = "error.budgetExhausted"
	ErrVerification     Key = "error.verification"
	ErrTaskNotFound     Key = "error.taskNotFound"
	ErrRouteNotFound    Key = "error.routeNotFound"
	ErrTaskMoveStatus   Key = "error.taskMoveStatus"
	ErrInvalidJSON      Key = "error.invalidJson"
	ErrInternal         Key = "error.internal"
	ErrInvalidRoute     Key = "error.invalidRoute"
	ErrContextRequired  Key = "error.contextRequired"
	ErrPlanEmpty        Key = "error.planEmpty"
	ErrPlanFailed       Key = "error.planFailed"
	ErrMCPNotFound      Key = "error.mcpNotFound"
	ErrMCPNameRequired  Key = "error.mcpNameRequired"
	ErrMCPConfig        Key = "error.mcpConfig"
	ErrEndpointNotFound Key = "error.endpointNotFound"
	ErrBadRequest       Key = "error.badRequest"
	ErrMethodNotAllowed Key = "error.methodNotAllowed"

	// Pipeline templates.
	PipelineDevName     Key = "pipeline.dev.name"
	PipelineDevAbout    Key = "pipeline.dev.about"
	PipelineFreeName    Key = "pipeline.free.name"
	PipelineFreeAbout   Key = "pipeline.free.about"
	PipelineSingleName  Key = "pipeline.single.name"
	PipelineSingleAbout Key = "pipeline.single.about"
	StageAnalyze        Key = "stage.analyze"
	StageDesign         Key = "stage.design"
	StageImplement      Key = "stage.implement"
	StageTest           Key = "stage.test"
	StageReview         Key = "stage.review"
	StageDocs           Key = "stage.docs"
	StagePlan           Key = "stage.plan"
	StageGeneric        Key = "stage.generic"

	// Placeholder page served when the frontend has not been built yet.
	StaticBrand           Key = "static.brand"
	StaticUINotBuiltTitle Key = "static.uiNotBuilt.title"
	StaticUINotBuiltBody  Key = "static.uiNotBuilt.body"
	StaticUINotBuiltAPI   Key = "static.uiNotBuilt.api"
)
