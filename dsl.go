package jarbles_dsl

import (
	"bufio"
	_ "embed"
	"encoding/base64"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	mrand "math/rand"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed images/avatar1.jpeg
var image1 []byte

//go:embed images/avatar2.jpeg
var image2 []byte

//go:embed images/avatar3.jpeg
var image3 []byte

//go:embed images/avatar4.jpeg
var image4 []byte

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

type assistantDescription struct {
	Id          string                         `yaml:"id"`
	Name        string                         `yaml:"name"`
	Description string                         `yaml:"description"`
	Placeholder string                         `yaml:"placeholder"`
	Model       string                         `yaml:"model"`
	Messages    []assistantDescriptionMessage  `yaml:"messages"`
	Functions   []assistantDescriptionFunction `yaml:"functions"`
}

type Assistant struct {
	description assistantDescription
	avatarImage []byte
	actions     map[string]Action
}

func NewAssistant(name, description string) Assistant {
	id := slugify(name)

	return Assistant{
		avatarImage: randomImage(),
		description: assistantDescription{
			Id:          id,
			Name:        name,
			Description: description,
			Model:       string(ModelGPT35Turbo),
			Placeholder: "How can I help you?",
			Messages:    nil,
			Functions:   nil,
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

func (a *Assistant) ConfigDir() (string, error) {
	return a.userDir("config")
}

func (a *Assistant) StorageDir() (string, error) {
	return a.userDir("storage")
}

func (a *Assistant) Respond() {
	fmt.Printf(a.Execute(os.Stdin))
}

func (a *Assistant) Execute(r io.Reader) string {
	storageDir, err := a.StorageDir()
	if err != nil {
		return fmt.Sprintf("error while getting storage directory: %s: %s", storageDir, err)
	}

	logname := filepath.Join(storageDir, a.description.Id+".log")
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
		return fmt.Sprintf("error while trying to perform: %s: %s", action, err)
	}
	return fmt.Sprintf(output)
}

func (a *Assistant) Errorf(format string, v ...any) error {
	return fmt.Errorf(format, v...)
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

func (a *Assistant) userDir(dir string) (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("error while getting user home directory: %w", err)
	}

	name := filepath.Join(currentUser.HomeDir, ".jarbles", dir)
	return name, nil
}

func (a *Assistant) route(actionName, payload string) (string, error) {
	switch actionName {
	case "describe":
		return a.describe()
	case "image":
		return a.image()
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

func (a *Assistant) image() (string, error) {
	return base64.StdEncoding.EncodeToString(a.avatarImage), nil
}

func slugify(str string) string {
	s := strings.ToLower(str)
	s = strings.ReplaceAll(s, " ", "-")

	reg, _ := regexp.Compile("[^a-zA-Z0-9\\-]+")
	s = reg.ReplaceAllString(s, "")

	return s
}

func randomImage() []byte {
	images := [][]byte{image1, image2, image3, image4}
	randomIndex := mrand.Intn(len(images))
	return images[randomIndex]
}
