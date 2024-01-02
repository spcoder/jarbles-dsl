package jarbles_framework

import (
	"os/user"
	"regexp"
	"strings"
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

func slugify(str string) string {
	s := strings.ToLower(str)
	s = strings.ReplaceAll(s, " ", "-")

	reg, _ := regexp.Compile("[^a-zA-Z0-9\\-]+")
	s = reg.ReplaceAllString(s, "")

	return s
}
