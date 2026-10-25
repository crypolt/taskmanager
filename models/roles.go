package models

const (
	RoleAnalyze   = "analyze"
	RoleDesign    = "design"
	RoleImplement = "implement"
	RoleTest      = "test"
	RoleReview    = "review"
	RoleDocs      = "docs"
	RoleGeneric   = "generic"
	RolePlanner   = "planner"
	RoleAggregate = "aggregate"
	RoleVerify    = "verify"
)

// PlanRoles are the roles a planner is allowed to assign to subtasks.
// AllRoles is every role the UI may bind models to.
var AllRoles = []string{RoleAnalyze, RoleDesign, RoleImplement, RoleTest, RoleReview, RoleDocs, RoleGeneric, RolePlanner, RoleAggregate, RoleVerify}

var PlanRoles = []string{RoleAnalyze, RoleDesign, RoleImplement, RoleTest, RoleReview, RoleDocs, RoleGeneric}

func IsPlanRole(role string) bool {
	for _, allowed := range PlanRoles {
		if allowed == role {
			return true
		}
	}
	return false
}
