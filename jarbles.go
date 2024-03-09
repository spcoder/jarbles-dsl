package jarbles_framework

type functionProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum"`
}

type functionParameters struct {
	Type       string                      `json:"type,omitempty"`
	Required   []string                    `json:"required,omitempty"`
	Properties map[string]functionProperty `json:"properties,omitempty"`
}

type toolFunction struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Parameters  *functionParameters `json:"parameters,omitempty"`
}

type tool struct {
	Type     string        `json:"type"`
	Function *toolFunction `json:"function"`
}

type quicklink struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type frameworkAssistant struct {
	StaticID     string      `json:"static_id"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Model        string      `json:"model"`
	Instructions string      `json:"instructions"`
	Tools        []tool      `json:"tools,omitempty"`
	Version      string      `json:"version,omitempty"`
	BinaryName   string      `json:"binary_name,omitempty"`
	Placeholder  string      `json:"placeholder,omitempty"`
	Quicklinks   []quicklink `json:"quicklinks,omitempty"`
}
