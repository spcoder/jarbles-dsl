package jarbles_framework

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

type ExtensionResponse struct {
	HtmlTitle string `json:"html_title,omitempty"`
	HtmlHead  string `json:"html_head,omitempty"`
	HtmlBody  string `json:"html_body,omitempty"`
	Subject   string `json:"subject,omitempty"`
	TextBody  string `json:"text_body,omitempty"`
	NoLayout  bool   `json:"no_layout,omitempty"`
}

type ExtensionFunction func(payload string) (*ExtensionResponse, error)

type ExtensionAction struct {
	Id          string
	Index       int
	Name        string
	Description string
	Nav         bool
	Function    ActionFunction
	Extension   *Extension
	UrlPath     string
	Cron        string
}

type Extension struct {
	Id          string
	Name        string
	Description string
	actions     map[string]ExtensionAction
}

func NewExtension(name, description string) Extension {
	id := slugify(name)

	return Extension{
		Id:          id,
		Name:        name,
		Description: description,
	}
}

func NewExtensionResponse(htmlTitle, htmlHead, htmlBody, subject, textBody string) *ExtensionResponse {
	return &ExtensionResponse{
		HtmlTitle: htmlTitle,
		HtmlHead:  htmlHead,
		HtmlBody:  htmlBody,
		Subject:   subject,
		TextBody:  textBody,
	}
}

func (e *Extension) String() string {
	return fmt.Sprintf("(%s)", e.Id)
}

func (e *Extension) AddNavigation(id, name, description string, fn ExtensionFunction) {
	e.addAction(ExtensionAction{
		Id:          id,
		Index:       len(e.actions),
		Name:        name,
		Description: description,
		Nav:         true,
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
		UrlPath:   fmt.Sprintf("/extension/action/%s/%s", e.Id, id),
	})
}

func (e *Extension) AddAction(id string, fn ExtensionFunction) {
	e.addAction(ExtensionAction{
		Id:          slugify(id),
		Index:       len(e.actions),
		Name:        id,
		Description: id,
		Nav:         false,
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
		UrlPath:   fmt.Sprintf("/extension/action/%s/%s", e.Id, id),
	})
}

func (e *Extension) AddCron(id, cron string, fn ExtensionFunction) {
	e.addAction(ExtensionAction{
		Id:          slugify(id),
		Index:       -1,
		Name:        id,
		Description: id,
		Nav:         false,
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
		UrlPath:   fmt.Sprintf("/extension/action/%s/%s", e.Id, id),
		Cron:      cron,
	})
}

func (e *Extension) ActionById(id string) *ExtensionAction {
	for _, action := range e.actions {
		if action.Id == id {
			return &action
		}
	}
	return nil
}

func (e *Extension) NavigationByName(name string) *ExtensionAction {
	for _, action := range e.actions {
		if action.Name == name && action.Nav {
			return &action
		}
	}
	return nil
}

func (e *Extension) addAction(v ExtensionAction) {
	if e.actions == nil {
		e.actions = make(map[string]ExtensionAction)
	}
	e.actions[v.Id] = v
}

func (e *Extension) Respond() {
	output, err := e.Execute(os.Stdin)
	if err != nil {
		_, err := fmt.Fprintf(os.Stderr, err.Error())
		if err != nil {
			fmt.Printf(fmt.Sprintf("error while writing to stderr: %s", err.Error()))
		}
	} else {
		_, err := fmt.Fprintf(os.Stdout, "%s", output)
		if err != nil {
			fmt.Printf(fmt.Sprintf("error while writing to stdout: %s", err.Error()))
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
		return "", fmt.Errorf("error while scanning: %w", scanner.Err())
	}

	// add newlines back
	payload := strings.Join(lines, "\n")

	// route the request and output the response
	output, err := e.route(action, payload)
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
		for _, operation := range e.actions {
			if operation.Id == operationId {
				logger.Info("calling operation", "name", operationId)
				logger.Debug("calling operation", "payload", payload)
				return operation.Function(payload)
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
		Nav         bool   `json:"nav"`
		Cron        string `json:"cron"`
		CronSummary string `json:"cronSummary"`
	}

	type JarblesExtension struct {
		Id          string                            `json:"id"`
		Name        string                            `json:"name"`
		Description string                            `json:"description"`
		Action      map[string]JarblesExtensionAction `json:"actions"`
	}

	je := JarblesExtension{
		Id:          e.Id,
		Name:        e.Name,
		Description: e.Description,
		Action:      make(map[string]JarblesExtensionAction),
	}
	for _, op := range e.actions {
		je.Action[op.Id] = JarblesExtensionAction{
			Id:          op.Id,
			Index:       op.Index,
			Name:        op.Name,
			Description: op.Description,
			Nav:         op.Nav,
			Cron:        op.Cron,
		}
	}

	data, err := json.Marshal(je)
	if err != nil {
		return "", fmt.Errorf("error while marshaling: %w", err)
	}
	return string(data), nil
}
