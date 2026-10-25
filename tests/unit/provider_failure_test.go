package unit

import (
	"strings"
	"testing"

	"taskmanager/internal/services/externals/llm"
)

const refusalLog = `timestamp=2026-10-24T10:47:53.078Z level=ERROR run=13a07341 message="stream error" providerID=opencode modelID=gpt-5.4-nano small=true agent=title mode=primary error.error="AI_APICallError: Upstream request failed: Insufficient account funds"
timestamp=2026-10-24T10:47:53.585Z level=ERROR run=13a07341 message="stream error" providerID=opencode modelID=deepseek-v4.1-flash-free small=false agent=task-runner mode=primary error.error="AI_APICallError: Free usage limit reached. Subscribe to OpenCode Go"
`

func TestProviderFailurePrefersRequestedModel(t *testing.T) {
	got := llm.ProviderFailure(refusalLog, "opencode/deepseek-v4.1-flash-free")
	if want := "Free usage limit reached"; !strings.Contains(got, want) {
		t.Fatalf("got %q, want it to contain %q", got, want)
	}
}

func TestProviderFailureFallsBackToOtherModel(t *testing.T) {
	got := llm.ProviderFailure(refusalLog, "opencode/nemotron-3-ultra-free")
	if !strings.Contains(got, "Free usage limit") {
		t.Fatalf("got %q, want the real refusal rather than the title helper error", got)
	}
	if strings.Contains(got, "small=true") {
		t.Fatalf("got %q, must never report the title helper line", got)
	}
}

func TestProviderFailureMatchesGenericAuthError(t *testing.T) {
	log := `level=ERROR message="stream error" error.error="AI_APICallError: Invalid API key"`
	if got := llm.ProviderFailure(log, "opencode/any"); got == "" {
		t.Fatal("generic auth failure must be reported")
	}
}
