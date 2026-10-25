package internals

import (
	"encoding/json"
	"fmt"
	"strings"

	"taskmanager/models"
	"taskmanager/skills"
)

// SystemPrompt returns the role instruction for an agent. Prompts live in
// skills/agents/*.md so they can be edited without touching the engine.
// sharedRules is prepended to every agent prompt: the rules of a shared
// workspace where several agents work on one project at the same time.
const sharedRules = `## Shared workspace rules
- You are one of several agents working on the same project, often in parallel.
- Stay inside your task: never rewrite, delete or reformat files owned by other tasks.
- Your scope (when given) lists the files and directories you own; do not touch anything outside it.
- Treat the shared context as ground truth: do not redo finished work and do not contradict earlier decisions.
- If a change outside your scope is required, describe it in the result instead of making it.
- Budget your steps: read only what the task needs, then produce the result. When the task names files, create them first and refine afterwards; an unfinished file is worse than a short one.
- Answer with the final result only: no restating the task, no filler, no repetition of the context.
`

func SystemPrompt(role string) string {
	if prompt, ok := skills.Agent(role); ok {
		return sharedRules + "\n" + prompt
	}
	if fallback, ok := skills.Agent(models.RoleGeneric); ok {
		return sharedRules + "\n" + fallback
	}
	return sharedRules + "\nYou are an executor. Complete the task at the highest quality and return the finished result."
}

// VerifySystem returns the verifier instruction.
func VerifySystem() string {
	if prompt, ok := skills.Agent(models.RoleVerify); ok {
		return prompt
	}
	return `You are an independent verifier. Reply with VALID JSON only: {"pass": true|false, "score": 0-100, "issues": ["..."]}`
}

// ExecPrompt builds the executor prompt for one stage. Feedback carries the
// previous rejection verbatim: without it a retry starts blind and repeats the
// same mistakes, which is how a task burns every attempt without converging.
func ExecPrompt(title, description, context, feedback string) string {
	var builder strings.Builder
	builder.WriteString("## Task\n")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	builder.WriteString(description)
	if strings.TrimSpace(context) != "" {
		builder.WriteString("\n\n## Context from previous stages\n")
		builder.WriteString(context)
	}
	if strings.TrimSpace(feedback) != "" {
		builder.WriteString("\n\n## Previous attempt was rejected\n")
		builder.WriteString(strings.TrimSpace(feedback))
		builder.WriteString("\n\nFix every point above and inspect the workspace for what already exists. " +
			"Do not return the same result again.")
	}
	builder.WriteString("\n\nReturn only the stage result, without introductions or apologies.")
	return builder.String()
}

// PlanPrompt asks the planner for a JSON plan.
func PlanPrompt(title, description, context string) string {
	extra := ""
	if strings.TrimSpace(context) != "" {
		extra = "\n\n## Shared context\n" + context
	}
	return fmt.Sprintf(`## Task
%s

%s%s

Split the task into 2-6 subtasks for the pipeline. Allowed roles: analyze, design, implement, test, review, docs.
Every subtask must be self-contained and verifiable. Give each subtask a scope: the files or directories it owns (an empty list when the subtask is not a code change).
Reply with VALID JSON only, no markdown:
{"subtasks":[{"title":"...","description":"...","role":"analyze","scope":["internal/foo"]}]}`, title, description, extra)
}

// VerifyPrompt asks the verifier for a verdict JSON. Changes lists what the
// executor actually touched: the files are the contract, the message is only a
// claim, so the verifier is told to read them.
func VerifyPrompt(title, description, output, context, changes string) string {
	extra := ""
	if strings.TrimSpace(context) != "" {
		extra = "\n\n## Shared context\n" + context
	}
	workspace := ""
	if strings.TrimSpace(changes) != "" {
		workspace = "\n\n## Workspace changes\n" + changes +
			"\n\nRead the changed files before judging: the workspace is ground truth, " +
			"the executor message is only a claim. Cite file paths in every issue."
	}
	return fmt.Sprintf(`## Original task
%s

%s%s%s

## Executor output
%s

Assess the output against the task, the shared context and the workspace, then return the verdict JSON.`, title, description, extra, workspace, output)
}

// ParsePlan decodes a planner response into normalized subtasks.
func ParsePlan(raw string) ([]models.Subtask, error) {
	block, ok := pickJSON(raw, `"subtasks"`)
	if !ok {
		return nil, fmt.Errorf("planner response contains no JSON")
	}
	var envelope models.PlanEnvelope
	if err := json.Unmarshal([]byte(block), &envelope); err != nil {
		return nil, fmt.Errorf("decode plan: %w", err)
	}

	out := make([]models.Subtask, 0, len(envelope.Subtasks))
	for _, subtask := range envelope.Subtasks {
		subtask.Title = strings.TrimSpace(subtask.Title)
		if subtask.Title == "" {
			continue
		}
		if !models.IsPlanRole(subtask.Role) {
			subtask.Role = models.RoleGeneric
		}
		subtask.Scope = models.NormalizeScope(subtask.Scope)
		out = append(out, subtask)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("planner returned an empty subtask list")
	}
	if len(out) > models.MaxPlanSubtasks {
		out = out[:models.MaxPlanSubtasks]
	}
	return out, nil
}

// ParseVerdict decodes a verifier response. A malformed reply is a failed
// verdict, not a transport error.
func ParseVerdict(raw string) (models.Verdict, error) {
	block, ok := pickJSON(raw, `"pass"`)
	if !ok {
		return models.Verdict{Pass: false, Issues: []string{"verifier response contains no JSON"}}, nil
	}
	var verdict models.Verdict
	if err := json.Unmarshal([]byte(block), &verdict); err != nil {
		return models.Verdict{Pass: false, Issues: []string{"invalid verdict JSON: " + err.Error()}}, nil
	}
	return models.NormalizeVerdict(verdict), nil
}

// pickJSON returns a JSON object from the model reply, tolerating markdown
// fences and prose around it. Every balanced {...} span is validated, so braces
// mentioned in prose (paths like `{en,ru}.json`) are skipped, and a span
// carrying the marker (a verdict has "pass", a plan has "subtasks") wins.
func pickJSON(text, marker string) (string, bool) {
	if trimmed := strings.TrimSpace(text); json.Valid([]byte(trimmed)) {
		return trimmed, true
	}
	var fallback string
	for i := 0; i < len(text); i++ {
		if text[i] != '{' {
			continue
		}
		end, ok := matchBrace(text, i)
		if !ok {
			continue
		}
		candidate := text[i : end+1]
		if !json.Valid([]byte(candidate)) {
			continue
		}
		if marker != "" && strings.Contains(candidate, marker) {
			return candidate, true
		}
		if len(candidate) > len(fallback) {
			fallback = candidate
		}
	}
	return fallback, fallback != ""
}

// matchBrace returns the index of the brace closing the object opened at start,
// ignoring braces inside JSON strings.
func matchBrace(text string, start int) (int, bool) {
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(text); i++ {
		c := text[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}
