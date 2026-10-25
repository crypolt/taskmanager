package models

const (
	StatusPending    = "pending"
	StatusPlanning   = "planning"
	StatusBlocked    = "blocked"
	StatusRunning    = "running"
	StatusVerifying  = "verifying"
	StatusRetrying   = "retrying"
	StatusNeedsHuman = "needs_human"
	StatusDone       = "done"
	StatusFailed     = "failed"
	StatusCancelled  = "cancelled"
)

var (
	// TerminalStatuses are final for the task lifecycle.
	TerminalStatuses = []string{StatusDone, StatusFailed, StatusCancelled}
	// RunningStatuses are statuses that occupy a worker or a verifier.
	RunningStatuses = []string{StatusPlanning, StatusRunning, StatusVerifying}
	// FailureStatuses block dependent tasks.
	FailureStatuses = []string{StatusFailed, StatusCancelled, StatusNeedsHuman}
)

func IsTerminalStatus(status string) bool {
	return contains(TerminalStatuses, status)
}

func IsRunningStatus(status string) bool {
	return contains(RunningStatuses, status)
}

func IsFailureStatus(status string) bool {
	return contains(FailureStatuses, status)
}

func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
