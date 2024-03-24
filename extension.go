package jarbles_framework

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

func NewExtension(name, description string) Extension {
	id := slugify(name)

	return Extension{
		ID:          id,
		Name:        name,
		Description: description,
	}
}

func NewExtensionResponse(htmlTitle, htmlHead, htmlBody, subject, textBody string) *ExtensionResponse {
	return &ExtensionResponse{
		HTMLTitle: htmlTitle,
		HTMLHead:  htmlHead,
		HTMLBody:  htmlBody,
		Subject:   subject,
		TextBody:  textBody,
	}
}

func (e *Extension) String() string {
	return fmt.Sprintf("(%s)", e.ID)
}

func (e *Extension) AddCard(title, description, id string) {
	e.Cards = append(e.Cards, ExtensionCard{
		ID:   id,
		HTML: lib.CardDefault(e.Name, title, description, e.ActionUrl(id)),
	})
}

func (e *Extension) AddCardCustom(card ExtensionCard) {
	e.Cards = append(e.Cards, card)
}

func (e *Extension) AddAction(id string, fn ExtensionFunction) {
	e.addAction(ExtensionAction{
		ID:          slugify(id),
		Index:       len(e.actions),
		Name:        id,
		Description: id,
		Function: func(payload string) (string, error) {
			response, err := fn(payload)
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
		URLPath:   fmt.Sprintf("/extension/action/%s/%s", e.ID, id),
	})
}

func (e *Extension) AddCommand(id string, fn CommandFunction) {
	e.addCommand(ExtensionCommand{
		ID: slugify(id),
		Function: func(payload string) error {
			err := fn(payload)
			if err != nil {
				return err
			}
			return nil
		},
		Extension: e,
	})
}

func (e *Extension) AddCron(id, cron string, fn ExtensionFunction) {
	e.addAction(ExtensionAction{
		ID:          slugify(id),
		Index:       -1,
		Name:        id,
		Description: id,
		Function: func(payload string) (string, error) {
			response, err := fn(payload)
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
		URLPath:   fmt.Sprintf("/extension/action/%s/%s", e.ID, id),
		Cron:      cron,
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
	output, err := e.Execute(os.Stdin)
	if err != nil {
		_, err := fmt.Fprintf(os.Stderr, err.Error())
		if err != nil {
			fmt.Printf(fmt.Sprintf("error while writing to stderr: %s", err.Error()))
		}
	} else {
		if output != "" {
			_, err := fmt.Fprintf(os.Stdout, "%s", output)
			if err != nil {
				fmt.Printf(fmt.Sprintf("error while writing to stdout: %s", err.Error()))
			}
		}
	}
}

func (e *Extension) Execute(r io.Reader) (string, error) {
	var err error
	logger, err = NewLibLogger(e, "extensions.log")
	if err != nil {
		return "", fmt.Errorf("error while creating logger: %w", err)
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
		return "", fmt.Errorf("error while scanning: %w", scanner.Err())
	}

	// add newlines back
	payload := strings.Join(lines, "\n")

	// route the request and output the response
	output, err := e.route(operationId, payload)
	if err != nil {
		logger.Log(context.Background(), slog.LevelDebug-1, "operation response", "error", err.Error())
		return "", err
	}

	logger.Log(context.Background(), slog.LevelDebug-1, "operation response", "output", output)
	return output, nil
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
