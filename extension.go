package framework

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/spcoder/jarbles-framework/lib"
	"io"
	"log/slog"
	"os"
	"strings"
)

type ExtensionResponse struct {
	HTMLTitle string `json:"html_title,omitempty"`
	HTMLHead  string `json:"html_head,omitempty"`
	HTMLBody  string `json:"html_body,omitempty"`
	Subject   string `json:"subject,omitempty"`
	TextBody  string `json:"text_body,omitempty"`
	NoLayout  bool   `json:"no_layout,omitempty"`
}

type ExtensionFunction func(payload string) (*ExtensionResponse, error)

type ExtensionAction struct {
	ID          string
	Index       int
	Name        string
	Description string
	Function    ActionFunction
	Extension   *Extension
	URLPath     string
	Cron        string
}

type ExtensionCommand struct {
	ID        string
	Extension *Extension
	Function  CommandFunction
}

type ExtensionCard struct {
	ID   string `json:"id"`
	HTML string `json:"html"`
}

type Extension struct {
	ID          string
	Name        string
	Description string
	Cards       []ExtensionCard
	actions     map[string]ExtensionAction
	commands    map[string]ExtensionCommand
}

type NewExtensionOptions struct {
	Name        string
	Description string
}

func NewExtension(options NewExtensionOptions) Extension {
	id := slugify(options.Name)

	return Extension{
		ID:          id,
		Name:        options.Name,
		Description: options.Description,
	}
}

func (e *Extension) String() string {
	return fmt.Sprintf("(%s)", e.ID)
}

type AddCardOptions struct {
	ID          string
	ActionID    string
	Title       string
	Description string
}

func (e *Extension) AddCard(options AddCardOptions) {
	e.Cards = append(e.Cards, ExtensionCard{
		ID: options.ID,
		HTML: lib.CardDefault(lib.CardDefaultOptions{
			ExtensionName: e.Name,
			Title:         options.Title,
			Description:   options.Description,
			Href:          e.ActionUrl(options.ActionID),
		}),
	})
}

func (e *Extension) AddCardCustom(card ExtensionCard) {
	e.Cards = append(e.Cards, card)
}

type AddActionOptions struct {
	ID       string
	Function ExtensionFunction
}

func (e *Extension) AddAction(options AddActionOptions) {
	e.addAction(ExtensionAction{
		ID:          slugify(options.ID),
		Index:       len(e.actions),
		Name:        options.ID,
		Description: options.ID,
		Function: func(payload string) (string, error) {
			response, err := options.Function(payload)
			if err != nil {
				return "", err
			}
			data, err := json.Marshal(response)
			if err != nil {
				return "", fmt.Errorf("error while marshaling response: %w", err)
			}
			return string(data), nil
		},
		Extension: e,
		URLPath:   fmt.Sprintf("/extension/action/%s/%s", e.ID, options.ID),
	})
}

type AddCommandOptions struct {
	ID       string
	Function CommandFunction
}

func (e *Extension) AddCommand(options AddCommandOptions) {
	e.addCommand(ExtensionCommand{
		ID: slugify(options.ID),
		Function: func(payload string) error {
			err := options.Function(payload)
			if err != nil {
				return err
			}
			return nil
		},
		Extension: e,
	})
}

type AddCronOptions struct {
	ID       string
	Cron     string
	Function ExtensionFunction
}

func (e *Extension) AddCron(options AddCronOptions) {
	e.addAction(ExtensionAction{
		ID:          slugify(options.ID),
		Index:       -1,
		Name:        options.ID,
		Description: options.ID,
		Function: func(payload string) (string, error) {
			response, err := options.Function(payload)
			if err != nil {
				return "", err
			}
			data, err := json.Marshal(response)
			if err != nil {
				return "", fmt.Errorf("error while marshaling response: %w", err)
			}
			return string(data), nil
		},
		Extension: e,
		URLPath:   fmt.Sprintf("/extension/action/%s/%s", e.ID, options.ID),
		Cron:      options.Cron,
	})
}

func (e *Extension) ActionById(id string) *ExtensionAction {
	for _, action := range e.actions {
		if action.ID == id {
			return &action
		}
	}
	return nil
}

func (e *Extension) ActionUrl(id string) string {
	action := e.ActionById(id)
	if action != nil {
		return action.URLPath
	}
	return ""
}

func (e *Extension) addAction(v ExtensionAction) {
	if e.actions == nil {
		e.actions = make(map[string]ExtensionAction)
	}
	e.actions[v.ID] = v
}

func (e *Extension) addCommand(v ExtensionCommand) {
	if e.commands == nil {
		e.commands = make(map[string]ExtensionCommand)
	}
	e.commands[v.ID] = v
}

func (e *Extension) Respond() {
	fmt.Printf(e.execute(os.Stdin))
}

func (e *Extension) Test(r io.Reader) string {
	return e.execute(r)
}

func (e *Extension) execute(r io.Reader) string {
	var err error
	logger, err = NewLibLogger(e, "extensions.log")
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

	// grab the operation id
	scanner.Scan()
	operationId := scanner.Text()

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
	output, err := e.route(operationId, payload)
	if err != nil {
		logger.Log(context.Background(), slog.LevelDebug-1, "operation response", "error", err.Error())
		return err.Error()
	}

	logger.Log(context.Background(), slog.LevelDebug-1, "operation response", "output", output)
	return output
}

// Payload builds a payload from an action and data. This is useful for testing.
func (e *Extension) Payload(action, data string) io.Reader {
	return strings.NewReader(action + "\n\n" + data)
}

func (e *Extension) route(operationId, payload string) (string, error) {
	switch operationId {
	case "describe":
		return e.describe()
	default:
		for _, action := range e.actions {
			if action.ID == operationId {
				logger.Info("calling action", "name", action.ID)
				logger.Debug("calling action", "payload", payload)
				return action.Function(payload)
			}
		}
		for _, command := range e.commands {
			if command.ID == operationId {
				logger.Info("calling command", "name", command.ID)
				logger.Debug("calling command", "payload", payload)
				return "", command.Function(payload)
			}
		}
		return "", fmt.Errorf("unknown operation: %s", operationId)
	}
}

// transforms the extension struct to a jarbles compatible one, and then returns the marshaled json
func (e *Extension) describe() (string, error) {
	logger.Debug("describe called")

	type JarblesExtensionAction struct {
		Id          string `json:"id"`
		Index       int    `json:"index"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Cron        string `json:"cron"`
		CronSummary string `json:"cronSummary"`
	}

	type JarblesExtensionCommand struct {
		Id string `json:"id"`
	}

	type JarblesExtensionCard struct {
		Id   string `json:"id"`
		Html string `json:"html"`
	}

	type JarblesExtension struct {
		Id          string                             `json:"id"`
		Name        string                             `json:"name"`
		Description string                             `json:"description"`
		Actions     map[string]JarblesExtensionAction  `json:"actions"`
		Commands    map[string]JarblesExtensionCommand `json:"commands"`
		Cards       []JarblesExtensionCard             `json:"cards"`
	}

	je := JarblesExtension{
		Id:          e.ID,
		Name:        e.Name,
		Description: e.Description,
		Actions:     make(map[string]JarblesExtensionAction),
		Commands:    make(map[string]JarblesExtensionCommand),
		Cards:       make([]JarblesExtensionCard, 0),
	}
	for _, op := range e.actions {
		je.Actions[op.ID] = JarblesExtensionAction{
			Id:          op.ID,
			Index:       op.Index,
			Name:        op.Name,
			Description: op.Description,
			Cron:        op.Cron,
		}
	}
	for _, op := range e.commands {
		je.Commands[op.ID] = JarblesExtensionCommand{
			Id: op.ID,
		}
	}
	for _, card := range e.Cards {
		je.Cards = append(je.Cards, JarblesExtensionCard{
			Id:   card.ID,
			Html: card.HTML,
		})
	}

	data, err := json.Marshal(je)
	if err != nil {
		return "", fmt.Errorf("error while marshaling: %w", err)
	}
	return string(data), nil
}
