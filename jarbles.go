package framework

type functionProperty struct {
	Type        string   `json:"type" toml:"type"`
	Description string   `json:"description" toml:"description"`
	Enum        []string `json:"enum,omitempty" toml:"enum,omitempty"`
}

type functionParameters struct {
	Type       string                      `json:"type,omitempty" toml:"type,omitempty"`
	Required   []string                    `json:"required,omitempty" toml:"required,omitempty"`
	Properties map[string]functionProperty `json:"properties,omitempty" toml:"properties,omitempty"`
}

type toolFunction struct {
	Name        string              `json:"name" toml:"name"`
	Description string              `json:"description" toml:"description"`
	Parameters  *functionParameters `json:"parameters,omitempty" toml:"parameters,omitempty"`
}

type tool struct {
	Type     string        `json:"type" toml:"type"`
	Function *toolFunction `json:"function" toml:"function"`
}

type quicklink struct {
	Title   string `json:"title" toml:"title"`
	Content string `json:"content" toml:"content"`
}

type message struct {
	Role    string `json:"role" toml:"role"`
	Content string `json:"content" toml:"content"`
	Visible bool   `json:"visible,omitempty" toml:"visible,omitempty"`
}

type initiate struct {
	Silence  int     `json:"silence,omitempty" toml:"silence,omitempty" yaml:"silence,omitempty"`
	Chance   float64 `json:"chance,omitempty" toml:"chance,omitempty" yaml:"chance,omitempty"`
	Retries  int     `json:"retries,omitempty" toml:"retries,omitempty" yaml:"retries,omitempty"`
	Cooldown int     `json:"cooldown,omitempty" toml:"cooldown,omitempty" yaml:"cooldown,omitempty"`
	Prompt   string  `json:"prompt,omitempty" toml:"prompt,omitempty" yaml:"prompt,omitempty"`
	Model    string  `json:"model,omitempty" toml:"model,omitempty" yaml:"model,omitempty"`
}

type frameworkAssistant struct {
	StaticID     string      `json:"static_id" toml:"static_id"`
	Name         string      `json:"name" toml:"name"`
	Description  string      `json:"description" toml:"description"`
	Model        string      `json:"model" toml:"model"`
	Instructions string      `json:"instructions" toml:"instructions"`
	Tools        []tool      `json:"tools,omitempty" toml:"tools,omitempty"`
	Version      string      `json:"version,omitempty" toml:"version,omitempty"`
	BinaryName   string      `json:"binary_name,omitempty" toml:"binary_name,omitempty"`
	Placeholder  string      `json:"placeholder,omitempty" toml:"placeholder,omitempty"`
	Initiate     initiate    `json:"initiate,omitempty" toml:"initiate,omitempty"`
	Quicklinks   []quicklink `json:"quicklinks,omitempty" toml:"quicklinks,omitempty"`
	Messages     []message   `json:"messages,omitempty" toml:"messages,omitempty"`
}
