package models

const (
	PipelineIDFree   = "free"
	PipelineIDDev    = "dev"
	PipelineIDSingle = "single"
)

// MaxPlanSubtasks caps how many subtasks a planner may produce.
const MaxPlanSubtasks = 8

// Stage is one fixed stage of a pipeline template. TitleKey is the i18n key
// used to render Title in the active language.
type Stage struct {
	Key      string `json:"key"`
	Role     string `json:"role"`
	Title    string `json:"title"`
	TitleKey string `json:"-"`
	Brief    string `json:"brief"`
}

// Pipeline is a named execution template. NameKey and AboutKey are i18n keys.
type Pipeline struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	NameKey  string  `json:"-"`
	About    string  `json:"about"`
	AboutKey string  `json:"-"`
	Stages   []Stage `json:"stages"`
}

var PipelineDev = Pipeline{
	ID:       PipelineIDDev,
	NameKey:  "pipeline.dev.name",
	AboutKey: "pipeline.dev.about",
	Stages: []Stage{
		{Key: RoleAnalyze, Role: RoleAnalyze, TitleKey: "stage.analyze",
			Brief: "Analyze the task: requirements, constraints, inputs/outputs, risks, definition of done. Do not write code."},
		{Key: RoleDesign, Role: RoleDesign, TitleKey: "stage.design",
			Brief: "Design the solution from the analysis: architecture, interfaces, file layout, step-by-step change plan."},
		{Key: RoleImplement, Role: RoleImplement, TitleKey: "stage.implement",
			Brief: "Implement the design. Return complete code (files and their content) and a short rationale for key decisions."},
		{Key: RoleTest, Role: RoleTest, TitleKey: "stage.test",
			Brief: "Write tests for the implementation: unit/integration scenarios, edge cases, run commands."},
		{Key: RoleReview, Role: RoleReview, TitleKey: "stage.review",
			Brief: "Review the implementation and tests: defects, security, performance. Findings with priority and a concrete fix."},
		{Key: RoleDocs, Role: RoleDocs, TitleKey: "stage.docs",
			Brief: "Document the outcome: what was built, how to run it, API/usage examples, known limitations."},
	},
}

var PipelineFreeTask = Pipeline{
	ID:       PipelineIDFree,
	NameKey:  "pipeline.free.name",
	AboutKey: "pipeline.free.about",
	Stages: []Stage{
		{Key: "plan", Role: RolePlanner, TitleKey: "stage.plan",
			Brief: "Split the task into 2-6 subtasks with roles analyze, design, implement, test, review, docs."},
	},
}

var PipelineSingleTask = Pipeline{
	ID:       PipelineIDSingle,
	NameKey:  "pipeline.single.name",
	AboutKey: "pipeline.single.about",
}

var Pipelines = []Pipeline{PipelineDev, PipelineFreeTask, PipelineSingleTask}

// FindPipeline resolves a pipeline template by id.
func FindPipeline(id string) (Pipeline, bool) {
	for _, pipeline := range Pipelines {
		if pipeline.ID == id {
			return pipeline, true
		}
	}
	return Pipeline{}, false
}
