package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// syncBuffer collects the CLI's stderr so the caller can read it while the
// process is still running.
type syncBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// providerFailurePatterns are log lines that mean the provider refused the
// call; retrying the same model is pointless.
var providerFailurePatterns = []string{
	"free usage limit",
	"insufficient account funds",
	"rate limit",
	"quota",
	"unauthorized",
	"invalid api key",
}

// providerFailure returns the stderr line that explains a refusal, preferring
// the one about the model actually requested (the CLI also logs a helper
// "title" model, whose error is not the reason the run failed).
func ProviderFailure(msg, model string) string { return providerFailure(msg, model) }

func providerFailure(msg, model string) string {
	short := model
	if i := strings.LastIndexByte(short, '/'); i >= 0 {
		short = short[i+1:]
	}
	var fallback string
	for _, line := range strings.Split(msg, "\n") {
		low := strings.ToLower(line)
		matched := false
		for _, p := range providerFailurePatterns {
			if strings.Contains(low, p) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		// The CLI also runs a small helper model for the session title; its
		// failure is not the reason the run stopped.
		if strings.Contains(line, "small=true") {
			continue
		}
		if short != "" && strings.Contains(line, short) {
			return firstLine(strings.TrimSpace(line))
		}
		fallback = firstLine(strings.TrimSpace(line))
	}
	return fallback
}

type OpenCode struct {
	Bin     string
	Timeout time.Duration
	// Auto approves tool permissions. Without it a headless run that asks for
	// approval (e.g. reading /tmp) is aborted by the CLI and yields no text.
	Auto bool
	// Agent selects the opencode agent; Config points at a generated config
	// that defines it (and disables MCP servers for task runs).
	Agent  string
	Config string
	// Pure skips external plugins.
	Pure bool
	// Idle aborts a run that produced no event for that long, so a stalled or
	// rate-limited provider fails fast instead of holding a worker for the
	// whole timeout.
	Idle time.Duration
}

func (o *OpenCode) Name() string { return "opencode" }

func firstLine(text string) string {
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		return text[:i]
	}
	if len(text) > 300 {
		return text[:300]
	}
	return text
}

type ocEvent struct {
	Type string `json:"type"`
	Part struct {
		Type   string `json:"type"`
		Text   string `json:"text"`
		Tokens struct {
			Input  int `json:"input"`
			Output int `json:"output"`
			Cache  struct {
				Read  int `json:"read"`
				Write int `json:"write"`
			} `json:"cache"`
		} `json:"tokens"`
		Cost  float64 `json:"cost"`
		Error string  `json:"error"`
	} `json:"part"`
}

func (o *OpenCode) Chat(ctx context.Context, req Request) (Response, error) {
	start := time.Now()
	bin := o.Bin
	if bin == "" {
		bin = "opencode"
	}
	prompt := req.Prompt
	if req.System != "" {
		prompt = req.System + "\n\n---\n\n" + prompt
	}
	timeout := o.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// --print-logs puts provider refusals on stderr; without it a quota error
	// is invisible and the run looks like a silent stall.
	args := []string{"run", "-m", req.Model, "--format", "json", "--print-logs", "--log-level", "ERROR"}
	if o.Auto {
		args = append(args, "--auto")
	}
	if o.Agent != "" {
		args = append(args, "--agent", o.Agent)
	}
	if o.Pure {
		args = append(args, "--pure")
	}
	args = append(args, prompt)
	cmd := exec.CommandContext(cctx, bin, args...)
	if o.Config != "" {
		cmd.Env = append(os.Environ(), "OPENCODE_CONFIG="+o.Config)
	}
	cmd.Stdin = nil
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Response{}, err
	}
	stderr := &syncBuffer{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return Response{}, fmt.Errorf("start opencode: %w", err)
	}

	var text strings.Builder
	var inTokens, outTokens, cacheRead, cacheWrite int
	var cost float64
	var runErr string
	events := make(chan ocEvent)
	go func() {
		defer close(events)
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || !strings.HasPrefix(line, "{") {
				continue
			}
			var ev ocEvent
			if err := json.Unmarshal([]byte(line), &ev); err != nil {
				continue
			}
			events <- ev
		}
	}()
	idle := o.Idle
	if idle <= 0 {
		idle = 2 * time.Minute
	}
	idleTimer := time.NewTimer(idle)
	probe := time.NewTicker(500 * time.Millisecond)
	defer probe.Stop()
	defer idleTimer.Stop()
	open := true
	for open {
		select {
		case ev, ok := <-events:
			if !ok {
				open = false
				break
			}
			if !idleTimer.Stop() {
				select {
				case <-probe.C:
					// The provider may refuse before any event reaches stdout, so the
					// refusal is only visible on stderr.
					if runErr == "" {
						if msg := providerFailure(stderr.String(), req.Model); msg != "" {
							runErr = msg
							open = false
							cancel()
						}
					}
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(idle)
			switch ev.Type {
			case "text":
				text.WriteString(ev.Part.Text)
			case "step_finish":
				if ev.Part.Tokens.Output == 0 && text.Len() == 0 {
					// The provider refused (quota, rate limit, upstream error)
					// and the CLI keeps retrying silently. Fail now instead of
					// waiting for the idle timeout on every model in the chain.
					runErr = "provider returned an empty completion (quota, rate limit or upstream error)"
					open = false
					cancel()
					continue
				}
				// Each agent step re-sends the whole conversation: input is the
				// fresh part, cache.read is the cached prefix. Summing cache
				// reads into input made usage look hundreds of times larger.
				inTokens += ev.Part.Tokens.Input
				outTokens += ev.Part.Tokens.Output
				cacheRead += ev.Part.Tokens.Cache.Read
				cacheWrite += ev.Part.Tokens.Cache.Write
				cost += ev.Part.Cost
			case "error":
				runErr = ev.Part.Error
				open = false
				cancel()
			}
			if runErr == "" {
				if msg := providerFailure(stderr.String(), req.Model); msg != "" {
					runErr = msg
					open = false
					cancel()
				}
			}
		case <-probe.C:
			// The provider may refuse before any event reaches stdout, so the
			// refusal is only visible on stderr.
			if runErr == "" {
				if msg := providerFailure(stderr.String(), req.Model); msg != "" {
					runErr = msg
					open = false
					cancel()
				}
			}
		case <-idleTimer.C:
			cancel()
			_ = cmd.Wait()
			msg := strings.TrimSpace(stderr.String())
			if msg != "" {
				msg = ": " + firstLine(msg)
			}
			return Response{}, fmt.Errorf("opencode produced no events for %s%s", idle, msg)
		}
	}
	// Release the reader goroutine if we stopped consuming early.
	go func() {
		for range events {
		}
	}()
	waitErr := cmd.Wait()
	if runErr != "" {
		return Response{}, fmt.Errorf("opencode error: %s", runErr)
	}
	if cctx.Err() == context.DeadlineExceeded {
		return Response{}, fmt.Errorf("opencode timed out after %s", timeout)
	}
	if waitErr != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = waitErr.Error()
		}
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		return Response{}, fmt.Errorf("opencode failed: %s", msg)
	}
	out := CleanOutput(text.String())
	if out == "" {
		return Response{}, ErrEmptyResponse
	}
	if inTokens == 0 && outTokens == 0 && cacheRead == 0 {
		inTokens = EstimateTokens(prompt)
		outTokens = EstimateTokens(out)
	}
	return Response{
		Text:             out,
		Provider:         o.Name(),
		Model:            req.Model,
		InputTokens:      inTokens,
		OutputTokens:     outTokens,
		CacheReadTokens:  cacheRead,
		CacheWriteTokens: cacheWrite,
		CostUSD:          cost,
		Latency:          time.Since(start),
	}, nil
}
