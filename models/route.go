package models

// Route is one entry of a role's model chain. Chains are ordered from cheap
// to strong: attempt N escalates to position N.
type Route struct {
	Role        string  `json:"role"`
	Position    int     `json:"position"`
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	PriceIn     float64 `json:"price_in"`
	PriceOut    float64 `json:"price_out"`
	DailyTokens int     `json:"daily_tokens"`
}

// ModelsResponse is the model routing read model grouped by role.
type ModelsResponse struct {
	Routes    map[string][]Route `json:"routes"`
	Providers []string           `json:"providers"`
}

// SetRouteInput creates or replaces one chain entry.
type SetRouteInput struct {
	Role        string  `json:"role"`
	Position    int     `json:"position"`
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	PriceIn     float64 `json:"price_in"`
	PriceOut    float64 `json:"price_out"`
	DailyTokens int     `json:"daily_tokens"`
}

// MoveRouteInput swaps two chain entries of one role, which is how the UI
// reorders a fallback chain.
type MoveRouteInput struct {
	Role string `json:"role"`
	From int    `json:"from"`
	To   int    `json:"to"`
}

// DeleteRouteInput removes one entry from a role's chain.
type DeleteRouteInput struct {
	Role     string `json:"role"`
	Position int    `json:"position"`
}
