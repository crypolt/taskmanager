package models

import "taskmanager/utils/i18n"

// AllStatuses returns every task status code the API can emit, in lifecycle
// order. Enum-driven: every entry must have a task.state.<code> key in the
// backend catalogue (guarded by a test).
func AllStatuses() []string {
	return []string{
		StatusPending,
		StatusPlanning,
		StatusBlocked,
		StatusRunning,
		StatusVerifying,
		StatusRetrying,
		StatusNeedsHuman,
		StatusDone,
		StatusFailed,
		StatusCancelled,
	}
}

// StatusKey maps a status code to its backend catalogue key
// ("task.state.<code>"). Unknown codes return "" so callers keep the raw
// value instead of rendering a missing-key placeholder.
func StatusKey(status string) i18n.Key {
	switch status {
	case StatusPending:
		return i18n.TaskStatePending
	case StatusPlanning:
		return i18n.TaskStatePlanning
	case StatusBlocked:
		return i18n.TaskStateBlocked
	case StatusRunning:
		return i18n.TaskStateRunning
	case StatusVerifying:
		return i18n.TaskStateVerifying
	case StatusRetrying:
		return i18n.TaskStateRetrying
	case StatusNeedsHuman:
		return i18n.TaskStateNeedsHuman
	case StatusDone:
		return i18n.TaskStateDone
	case StatusFailed:
		return i18n.TaskStateFailed
	case StatusCancelled:
		return i18n.TaskStateCancelled
	}
	return ""
}
