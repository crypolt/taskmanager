package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Request struct {
	Model       string
	System      string
	Prompt      string
	MaxTokens   int
	Temperature float64
}

type Response struct {
	Text         string
	Provider     string
	Model        string
	InputTokens  int
	OutputTokens int
	// CacheReadTokens/CacheWriteTokens are the prompt-cache parts of the
	// conversation, counted by the provider separately from fresh input.
	CacheReadTokens  int
	CacheWriteTokens int
	CostUSD          float64
	Latency          time.Duration
}

type Provider interface {
	Name() string
	Chat(ctx context.Context, req Request) (Response, error)
}

type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{providers: map[string]Provider{}}
}

func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q is not configured", name)
	}
	return p, nil
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.providers))
	for k := range r.providers {
		out = append(out, k)
	}
	return out
}

// Cost returns the USD price of tokens at the given per-million rates.
func Cost(in, out int, priceInPerM, priceOutPerM float64) float64 {
	return float64(in)/1_000_000*priceInPerM + float64(out)/1_000_000*priceOutPerM
}

// EstimateTokens is a coarse fallback when a provider reports no usage.
func EstimateTokens(s string) int {
	n := len([]rune(s))
	return n/4 + 1
}

func StripModelPrefix(model string) string {
	if i := strings.Index(model, "/"); i >= 0 {
		return model[i+1:]
	}
	return model
}

var ErrEmptyResponse = errors.New("model returned an empty response")

func CleanOutput(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// opencode may wrap the answer in a markdown fence; keep the text as is,
	// only trim surrounding fences when they cover the whole response.
	if strings.HasPrefix(s, "```") && strings.HasSuffix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) >= 2 {
			s = strings.Join(lines[1:len(lines)-1], "\n")
			s = strings.TrimSpace(s)
		}
	}
	return s
}
