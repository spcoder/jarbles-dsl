package jarbles_framework

import (
	"bufio"
	_ "embed"
	"encoding/base64"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

var (
	//go:embed avatar.jpeg
	avatar []byte
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

type assistantDescriptionMessage struct {
	Role    string `yaml:"role"`
	Hidden  bool   `yaml:"hidden"`
	Content string `yaml:"content"`
}

type assistantDescriptionFunction struct {
	Name        string                                 `yaml:"name"`
	Description string                                 `yaml:"description"`
	Properties  []assistantDescriptionFunctionProperty `yaml:"properties"`
	Required    []string                               `yaml:"required"`
}

type assistantDescriptionFunctionProperty struct {
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type"`
	Description string   `yaml:"description"`
	Enum        []string `yaml:"enum"`
}

type quicklink struct {
	Title   string `yaml:"title"`
	Content string `yaml:"content"`
}

type assistantDescription struct {
	Id          string                         `yaml:"id"`
	Name        string                         `yaml:"name"`
	Description string                         `yaml:"description"`
	Placeholder string                         `yaml:"placeholder"`
	Model       string                         `yaml:"model"`
	Messages    []assistantDescriptionMessage  `yaml:"messages"`
	Functions   []assistantDescriptionFunction `yaml:"functions"`
	Quicklinks  []quicklink                    `yaml:"quicklinks"`
}

type Assistant struct {
	description assistantDescription
	avatarImage []byte
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

//goland:noinspection GoUnusedExportedFunction
func NewAssistant(name, description string) Assistant {
	id := slugify(name)

	return Assistant{
		avatarImage: avatar,
		description: assistantDescription{
			Id:          id,
			Name:        name,
			Description: description,
			Model:       ModelGPT35Turbo,
			Placeholder: "How can I help you?",
			Messages:    nil,
			Functions:   nil,
			Quicklinks:  nil,
		},
	}
}

func (a *Assistant) String() string {
	return fmt.Sprintf("(%s) {%s}", a.description.Id, a.description.Model)
}

func (a *Assistant) Model(v string) {
	a.description.Model = v
}

func (a *Assistant) Placeholder(v string) {
	a.description.Placeholder = v
}

// AddMessages adds messages from a string.
//
// The format of the string is:
//
// { some text; this usually begins with the system message }
//
// @@@ {ROLE} =====
//
// { text; usually the second message is a user message }
//
// { text }
//
// @@@ {ROLE} =====
//
// { text }
//
// { text }
//
// { text }
func (a *Assistant) AddMessages(content string) {
	r := strings.NewReader(content)
	scanner := bufio.NewScanner(r)
	roleStr := "system"
	var lines string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "@@@") {
			// add the message
			a.AddMessageHidden(roleStr, lines)
			// reset the lines
			lines = ""
			// get the roleStr
			line = strings.ReplaceAll(line, "  ", " ")
			parts := strings.Split(line, " ")
			roleStr = parts[1]
		} else {
			lines += line + "\n"
		}
	}
	if lines != "" {
		a.AddMessageHidden(roleStr, lines)
	}

	err := scanner.Err()
	if err != nil {
		panic(err)
	}
}

func (a *Assistant) AddMessage(role string, content string) {
	a.description.Messages = append(a.description.Messages, assistantDescriptionMessage{
		Role:    strings.ToLower(role),
		Hidden:  false,
		Content: strings.TrimSpace(content),
	})
}

func (a *Assistant) AddMessageHidden(role string, content string) {
	a.description.Messages = append(a.description.Messages, assistantDescriptionMessage{
		Role:    strings.ToLower(role),
		Hidden:  true,
		Content: strings.TrimSpace(content),
	})
}

func (a *Assistant) Image(v []byte) {
	a.avatarImage = v
}

func (a *Assistant) AddQuicklink(title, content string) {
	a.description.Quicklinks = append(a.description.Quicklinks, quicklink{
		Title:   title,
		Content: content,
	})
}

func (a *Assistant) AddAction(v Action) {
	if a.actions == nil {
		a.actions = make(map[string]Action)
	}
	a.actions[v.Name] = v

	properties := make([]assistantDescriptionFunctionProperty, 0)
	for _, argument := range v.Arguments {
		properties = append(properties, assistantDescriptionFunctionProperty{
			Name:        argument.Name,
			Type:        argument.Type,
			Description: argument.Description,
			Enum:        argument.Enum,
		})
	}

	a.description.Functions = append(a.description.Functions, assistantDescriptionFunction{
		Name:        v.Name,
		Description: v.Description,
		Properties:  properties,
		Required:    v.RequiredArguments,
	})
}

func (a *Assistant) ConfigFilename() (string, error) {
	return filepath.Join(AssistantsDir(), a.description.Id+".config"), nil
}

func (a *Assistant) ConfigGet(key, defaultValue string) (string, error) {
	configFilename, err := a.ConfigFilename()
	if err != nil {
		logger.Error("error getting config filename: %s", err.Error())
		return "", fmt.Errorf("error while getting config filename: %w", err)
	}

	file, err := os.Open(configFilename)
	if err != nil {
		logger.Error("error while opening config file: %s", err.Error())
		return "", fmt.Errorf("error while opening config file: %w", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		k, v, found := strings.Cut(line, "=")
		if found && k == key {
			return v, nil
		}
	}

	err = scanner.Err()
	if err != nil {
		logger.Error("error while scanning config file: %s", err.Error())
		return "", fmt.Errorf("error while scanning config file: %w", err)
	}

	return defaultValue, nil
}

func (a *Assistant) ConfigSet(key string, value string) error {
	configFilename, err := a.ConfigFilename()
	if err != nil {
		logger.Error("error while getting config filename: %s", err.Error())
		return fmt.Errorf("error while getting config filename: %w", err)
	}

	file, err := os.OpenFile(configFilename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		logger.Error("error while opening config file: %s", err.Error())
		return fmt.Errorf("error while opening config file: %w", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	var lines []string
	updated := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		k, _, found := strings.Cut(line, "=")
		if found && k == key {
			updated = true
			line = key + "=" + value
		}
		lines = append(lines, line)
	}

	err = scanner.Err()
	if err != nil {
		logger.Error("error while scanning config file: %s", err.Error())
		return fmt.Errorf("error while scanning config file: %w", err)
	}

	if !updated {
		lines = append(lines, key+"="+value)
	}

	_, err = file.Seek(0, 0) // move the cursor to the start
	if err != nil {
		logger.Error("error while seeking config file: %s", err.Error())
		return fmt.Errorf("error while seeking config file: %w", err)
	}

	err = file.Truncate(0) // clear the file
	if err != nil {
		logger.Error("error while truncating config file: %s", err.Error())
		return fmt.Errorf("error while truncating config file: %w", err)
	}

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			logger.Error("error while writing to config file: %s", err.Error())
			return fmt.Errorf("error while writing to config file: %w", err)
		}
	}

	err = writer.Flush()
	if err != nil {
		logger.Error("error while flushing config file: %s", err.Error())
		return fmt.Errorf("error while flushing config file: %w", err)
	}

	return nil
}

func (a *Assistant) ConfigMap() (map[string]string, error) {
	configFilename, err := a.ConfigFilename()
	if err != nil {
		logger.Error("error while getting config filename: %s", err.Error())
		return nil, fmt.Errorf("error while getting config filename: %w", err)
	}

	file, err := os.Open(configFilename)
	if err != nil {
		logger.Error("error while opening config file: %s", err.Error())
		return nil, fmt.Errorf("error while opening config file: %w", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	keyValues := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		k, v, found := strings.Cut(line, "=")
		if found {
			keyValues[k] = v
		}
	}

	err = scanner.Err()
	if err != nil {
		logger.Error("error while scanning config file: %s", err.Error())
		return nil, fmt.Errorf("error while scanning config file: %w", err)
	}

	return keyValues, nil
}

func (a *Assistant) Respond() {
	fmt.Printf(a.Execute(os.Stdin))
}

func (a *Assistant) Execute(r io.Reader) string {
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
	return fmt.Sprintf(output)
}

func (a *Assistant) Payload(action, data string) io.Reader {
	return strings.NewReader(action + "\n\n" + data)
}

func (a *Assistant) route(actionName, payload string) (string, error) {
	switch actionName {
	case "describe":
		return a.describe()
	case "image":
		return a.image(), nil
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
	data, err := yaml.Marshal(a.description)
	if err != nil {
		return "", fmt.Errorf("error while marshaling yaml: %w", err)
	}
	return string(data), nil
}

func (a *Assistant) image() string {
	logger.Debug("image called")
	return base64.StdEncoding.EncodeToString(a.avatarImage)
}
