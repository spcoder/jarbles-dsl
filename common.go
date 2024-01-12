package jarbles_framework

import (
	"crypto/rand"
	"encoding/binary"
	"os"
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

//goland:noinspection GoUnusedExportedFunction
func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic("missing env variable: " + key)
	}

	return value
}

func slugify(str string) string {
	s := strings.ToLower(str)
	s = strings.ReplaceAll(s, " ", "-")

	reg, _ := regexp.Compile("[^a-zA-Z0-9\\-]+")
	s = reg.ReplaceAllString(s, "")

	return s
}

func randInt(n int) int {
	if n < 1 {
		n = 1
	}

	var v uint64
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return -1
	}

	v = binary.BigEndian.Uint64(b)

	return int(v % uint64(n))
}

func randomId() string {
	n := 20
	lettersAndNumbers := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	var sb strings.Builder
	k := len(lettersAndNumbers)

	for i := 0; i < n; i++ {
		c := lettersAndNumbers[randInt(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}
