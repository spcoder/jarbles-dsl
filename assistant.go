package framework

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"io"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

var (
	logger *slog.Logger
)

const ModelGPT35Turbo string = "gpt-3.5-turbo-1106"

//goland:noinspection GoUnusedConst
const (
	ModelGPT4Turbo string = "gpt-4-1106-preview"
	RoleSystem     string = "system"
	RoleUser       string = "user"
	RoleAssistant  string = "assistant"
)

type ActionArguments struct {
	Name        string
	Type        string
	Description string
	Enum        []string
}

type Action struct {
	Name              string
	Description       string
	Arguments         []ActionArguments
	RequiredArguments []string
	Function          ActionFunction
}

type Assistant struct {
	description frameworkAssistant
	actions     map[string]Action
}

func userDir(dir ...string) string {
	currentUser, err := user.Current()
	if err != nil {
		panic(fmt.Errorf("error while getting user home directory: %w", err))
	}

	paths := []string{currentUser.HomeDir, ".jarbles"}
	paths = append(paths, dir...)

	return filepath.Clean(strings.Join(paths, string(filepath.Separator)))
}

func AssistantsDir() string {
	return userDir("assistants")
}

func LogDir() string {
	return userDir("log")
}

type NewAssistantOptions struct {
	StaticID    string
	Name        string
	Description string
}

//goland:noinspection GoUnusedExportedFunction
func NewAssistant(options NewAssistantOptions) Assistant {
	return Assistant{
		description: frameworkAssistant{
			StaticID:    options.StaticID,
			Name:        options.Name,
			Description: options.Description,
			Model:       ModelGPT35Turbo,
			Placeholder: "How can I help you?",
			Tools:       nil,
			Quicklinks:  nil,
		},
	}
}

func NewAssistantFromTOML(data []byte) (Assistant, error) {
	var fa frameworkAssistant
	err := toml.Unmarshal(data, &fa)
	if err != nil {
		return Assistant{}, fmt.Errorf("error while unmarshaling toml: %w", err)
	}

	return Assistant{description: fa}, nil
}

func (a *Assistant) String() string {
	return fmt.Sprintf("(%s) {%s}", a.description.StaticID, a.description.Model)
}

func (a *Assistant) Model(v string) {
	a.description.Model = v
}

func (a *Assistant) Placeholder(v string) {
	a.description.Placeholder = v
}

func (a *Assistant) AddInstructions(v string) {
	a.description.Instructions = v
}

type AddQuicklinkOptions struct {
	Title   string
	Content string
}

func (a *Assistant) AddQuicklink(options AddQuicklinkOptions) {
	a.description.Quicklinks = append(a.description.Quicklinks, quicklink{
		Title:   options.Title,
		Content: options.Content,
	})
}

func (a *Assistant) AddTool(v Action) {
	if a.actions == nil {
		a.actions = make(map[string]Action)
	}
	a.actions[v.Name] = v

	t := tool{
		Type: "function",
		Function: &toolFunction{
			Name:        v.Name,
			Description: v.Description,
		},
	}

	if v.Arguments != nil {
		t.Function.Parameters = &functionParameters{
			Type:       "object",
			Required:   v.RequiredArguments,
			Properties: make(map[string]functionProperty),
		}
		for _, argument := range v.Arguments {
			t.Function.Parameters.Properties[argument.Name] = functionProperty{
				Type:        argument.Type,
				Description: argument.Description,
				Enum:        argument.Enum,
			}
		}
	}

	a.description.Tools = append(a.description.Tools, t)
}

func (a *Assistant) Respond() {
	fmt.Printf(a.execute(os.Stdin))
}

func (a *Assistant) Test(r io.Reader) string {
	return a.execute(r)
}

func (a *Assistant) execute(r io.Reader) string {
	var err error
	logger, err = NewLibLogger(a, "assistants.log")
	if err != nil {
		return fmt.Sprintf("error while creating logger: %s", err.Error())
	}
	defer func(l *slog.Logger) {
		h, ok := logger.Handler().(LibLogger)
		if ok {
			_ = h.Close()
		}
	}(logger)

	slog.SetDefault(logger)

	scanner := bufio.NewScanner(r)

	// grab the action
	scanner.Scan()
	action := scanner.Text()

	// skip payload delimiter
	scanner.Scan()

	// read the json payload
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if scanner.Err() != nil {
		return fmt.Sprintf("error while scanning: %s", scanner.Err())
	}

	// add newlines back
	payload := strings.Join(lines, "\n")

	// route the request and output the response
	output, err := a.route(action, payload)
	if err != nil {
		logger.Error("action response", "error", err.Error())
		return err.Error()
	}

	logger.Debug("action response", "output", output)
	return output
}

func (a *Assistant) Payload(action, data string) io.Reader {
	return strings.NewReader(action + "\n\n" + data)
}

func (a *Assistant) route(actionName, payload string) (string, error) {
	switch actionName {
	case "describe":
		return a.describe()
	default:
		for _, action := range a.actions {
			if action.Name == actionName {
				logger.Info("calling action", "name", actionName)
				logger.Debug("calling action", "payload", payload)
				return action.Function(payload)
			}
		}
		return "", fmt.Errorf("unknown action: %s", actionName)
	}
}

func (a *Assistant) describe() (string, error) {
	logger.Debug("describe called")
	data, err := json.Marshal(a.description)
	if err != nil {
		return "", fmt.Errorf("error while marshaling json: %w", err)
	}
	return string(data), nil
}
