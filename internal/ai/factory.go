package ai

// NewClient creates a new AI client for the specified provider.
func NewClient(provider Provider, cfg *Config) Client {
	switch provider {
	case ProviderClaude:
		return NewClaudeClient(cfg.Claude)
	case ProviderCodex:
		return NewCodexClient(cfg.Codex)
	case ProviderGemini:
		return NewGeminiClient(cfg.Gemini)
	default:
		return nil
	}
}

// AutoDetect finds the first available AI CLI tool.
// Priority: claude > codex > gemini
func AutoDetect(cfg *Config) (Client, error) {
	for _, provider := range AllProviders() {
		client := NewClient(provider, cfg)
		if client != nil && client.IsAvailable() {
			return client, nil
		}
	}
	return nil, ErrNoProvider
}

// GetClient returns a client based on config.Default setting.
// If "auto", it uses AutoDetect. Otherwise, it creates the specified provider.
func GetClient(cfg *Config) (Client, error) {
	if cfg.Default == "" || cfg.Default == "auto" {
		return AutoDetect(cfg)
	}

	provider, ok := ParseProvider(cfg.Default)
	if !ok {
		return nil, ErrNoProvider
	}

	client := NewClient(provider, cfg)
	if client == nil {
		return nil, ErrNoProvider
	}

	if !client.IsAvailable() {
		return nil, ErrNoProvider
	}

	return client, nil
}

// ListAvailable returns all available AI CLI tools.
func ListAvailable(cfg *Config) []Client {
	var available []Client
	for _, provider := range AllProviders() {
		client := NewClient(provider, cfg)
		if client != nil && client.IsAvailable() {
			available = append(available, client)
		}
	}
	return available
}
