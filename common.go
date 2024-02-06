package jarbles_framework

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"regexp"
	"strings"
	"time"
)

type ActionFunction func(payload string) (string, error)

//goland:noinspection GoUnusedExportedFunction
func MustCurrentUser() *user.User {
	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}

	return currentUser
}

//goland:noinspection GoUnusedExportedFunction
func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic("missing env variable: " + key)
	}

	return value
}

func PayloadParse(payload string) (map[string]any, error) {
	var request map[string]any
	err := json.Unmarshal([]byte(payload), &request)
	if err != nil {
		return nil, fmt.Errorf("error while unmarshaling payload: %w", err)
	}

	return request, nil
}

func PayloadMustParse(payload string) map[string]any {
	m, err := PayloadParse(payload)
	if err != nil {
		panic(err)
	}

	return m
}

func PayloadGetString(payload any, key, defaultValue string) (string, bool) {
	var payloadMap map[string]any
	switch v := payload.(type) {
	case string:
		var err error
		payloadMap, err = PayloadParse(v)
		if err != nil {
			return defaultValue, false // error while parsing
		}
	case map[string]any:
		payloadMap = v
	default:
		return defaultValue, false // wrong type
	}

	value, ok := payloadMap[key]
	if !ok {
		return defaultValue, false // missing key
	}

	s, ok := value.(string)
	if ok {
		return s, true
	}

	sarr, ok := value.([]any)
	if ok && len(sarr) > 0 {
		sv, ok := sarr[0].(string)
		if ok {
			return sv, true
		}
		return defaultValue, false
	}

	return defaultValue, false // wrong type
}

func SleepAtLeast(started time.Time, min time.Duration) {
	duration := time.Since(started)
	if duration < min {
		time.Sleep(min - duration)
	}
}

func slugify(str string) string {
	s := strings.ToLower(str)
	s = strings.ReplaceAll(s, " ", "-")

	reg, _ := regexp.Compile("[^a-zA-Z0-9\\-]+")
	s = reg.ReplaceAllString(s, "")

	return s
}
