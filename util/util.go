package util

import (
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

func StringToUpperAndSnake(str string) string {
	return strings.ReplaceAll(strings.ToUpper(str), "-", "_")
}

func MD5Checksum(buf []byte) ([]byte, error) {
	hash := md5.New()
	_, err := hash.Write(buf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write to hash")
	}

	return hash.Sum(nil), nil
}

func Prompt(msg string) bool {
	// check for yes and assume no on any other input to avoid annoyance
	fmt.Print(msg)
	var resp string
	_, err := fmt.Scanln(&resp)
	if err != nil {
		return false
	}
	if strings.ToLower(string(resp[0])) == "y" {
		return true
	}
	return false
}

func ExpandVars(str string, vars map[string]string) string {
	// Regex to match variable substitution of the form ${VAR}
	regex := regexp.MustCompile(`\$\{([\w-]+)\}`)
	indices := regex.FindAllStringSubmatchIndex(str, -1)

	// Go through the string in reverse order and replace all variables with their value
	expandedStr := str
	for i := len(indices) - 1; i >= 0; i-- {
		match := indices[i]
		// match[0] is the start index of the whole match
		startIndex := match[0]
		// match[1] is the end index of the whole match (exclusive)
		endIndex := match[1]
		// match[2] is start index of group
		startIndexGroup := match[2]
		// match[3] is end index of group (exclusive)
		endIndexGroup := match[3]

		varName := str[startIndexGroup:endIndexGroup]
		expandedStr = expandedStr[:startIndex] + vars[varName] + expandedStr[endIndex:]
	}

	return expandedStr
}
