package jarbles_dsl

import (
	"bufio"
	_ "embed"
	"encoding/base64"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed avatar.jpeg
var avatar []byte

var logger *log.Logger

const (
	ModelGPT35Turbo model = "gpt-3.5-turbo-1106"
	ModelGPT4Turbo  model = "gpt-4-1106-preview"
)

const (
	RoleSystem    role = "system"
	RoleUser      role = "user"
	RoleAssistant role = "assistant"
)

type role string

type model string

type ActionFunction func(payload string) (string, error)

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

func NewAssistant(name, description string) Assistant {
	id := slugify(name)

	return Assistant{
		avatarImage: avatar,
		description: assistantDescription{
			Id:          id,
			Name:        name,
			Description: description,
			Model:       string(ModelGPT35Turbo),
			Placeholder: "How can I help you?",
			Messages:    nil,
			Functions:   nil,
			Quicklinks:  nil,
		},
	}
}

func (a *Assistant) Model(v model) {
	a.description.Model = string(v)
}

func (a *Assistant) Placeholder(v string) {
	a.description.Placeholder = v
}

func (a *Assistant) AddMessage(role role, content string) {
	a.description.Messages = append(a.description.Messages, assistantDescriptionMessage{
		Role:    string(role),
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

func (a *Assistant) AssistantsDir() (string, error) {
	return a.userDir("assistants")
}

func (a *Assistant) LogDir() (string, error) {
	return a.userDir("log")
}

func (a *Assistant) ConfigFilename() (string, error) {
	assistantsDir, err := a.AssistantsDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(assistantsDir, a.description.Id+".config"), nil
}

func (a *Assistant) ConfigGet(key, defaultValue string) (string, error) {
	configFilename, err := a.ConfigFilename()
	if err != nil {
		return "", err
	}

	file, err := os.Open(configFilename)
	if err != nil {
		return "", err
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
		return "", err
	}

	return defaultValue, nil
}

func (a *Assistant) ConfigSet(key string, value string) error {
	configFilename, err := a.ConfigFilename()
	if err != nil {
		return err
	}

	file, err := os.OpenFile(configFilename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
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
		return err
	}

	if !updated {
		lines = append(lines, key+"="+value)
	}

	_, err = file.Seek(0, 0) // move the cursor to the start
	if err != nil {
		return err
	}

	err = file.Truncate(0) // clear the file
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}

func (a *Assistant) ConfigMap() (map[string]string, error) {
	configFilename, err := a.ConfigFilename()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(configFilename)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return keyValues, nil
}

func (a *Assistant) Respond() {
	fmt.Printf(a.Execute(os.Stdin))
}

func (a *Assistant) Execute(r io.Reader) string {
	logDir, err := a.LogDir()
	if err != nil {
		return fmt.Sprintf("error while getting log directory: %s: %s", logDir, err)
	}

	logname := filepath.Join(logDir, a.description.Id+".log")
	logfile, err := os.OpenFile(logname, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return fmt.Sprintf("error while creating log file: %s: %s", logname, err)
	}
	logger = log.New(logfile, "", log.LstdFlags)
	defer func(file *os.File) {
		_ = file.Close()
	}(logfile)

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
		return err.Error()
	}
	return fmt.Sprintf(output)
}

func (a *Assistant) Log(message string) {
	logger.Println(message)
}

func (a *Assistant) Logf(format string, v ...any) {
	logger.Printf(format, v...)
}

func (a *Assistant) LogError(message string) {
	logger.Println("ERROR: " + message)
}

func (a *Assistant) LogErrorf(format string, v ...any) {
	logger.Printf("ERROR: "+format, v...)
}

func (a *Assistant) userDir(dir ...string) (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("error while getting user home directory: %w", err)
	}

	paths := []string{currentUser.HomeDir, ".jarbles"}
	paths = append(paths, dir...)

	return filepath.Clean(strings.Join(paths, string(filepath.Separator))), nil
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
				return action.Function(payload)
			}
		}
		return "", fmt.Errorf("unknown action: %s", actionName)
	}
}

func (a *Assistant) describe() (string, error) {
	data, err := yaml.Marshal(a.description)
	if err != nil {
		return "", fmt.Errorf("error while marshaling yaml: %w", err)
	}
	return string(data), nil
}

func (a *Assistant) image() string {
	return base64.StdEncoding.EncodeToString(a.avatarImage)
}

func slugify(str string) string {
	s := strings.ToLower(str)
	s = strings.ReplaceAll(s, " ", "-")

	reg, _ := regexp.Compile("[^a-zA-Z0-9\\-]+")
	s = reg.ReplaceAllString(s, "")

	return s
}
