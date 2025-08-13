package jsonpointer

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type partType string

const (
	partTypeKey   partType = "key"
	partTypeIndex partType = "index"
)

type navigationPart struct {
	Type  partType
	Value string
}

func (n navigationPart) unescapeValue() string {
	val := strings.ReplaceAll(n.Value, "~1", "/")
	val = strings.ReplaceAll(val, "~0", "~")
	return val
}

func (n navigationPart) getIndex() int {
	index, _ := strconv.Atoi(n.Value)
	return index
}

var (
	tokenRegex     = regexp.MustCompile("^(?:[\x00-\x2E\x30-\x7D\x7F-\uffff]|~[01])+$")
	digitOnlyRegex = regexp.MustCompile("^[0-9]+$")
)

func (j JSONPointer) getNavigationStack() ([]navigationPart, error) {
	if len(j) == 0 {
		return nil, errors.New("jsonpointer must not be empty")
	}

	if len(j) == 1 && j[0] == '/' {
		return nil, nil
	}

	if !strings.HasPrefix(string(j), "/") {
		return nil, fmt.Errorf("jsonpointer must start with /: %s", string(j))
	}

	stack := []navigationPart{}

	strParts := strings.Split(strings.TrimPrefix(string(j), "/"), "/")

	for _, part := range strParts {
		if len(part) == 0 {
			return nil, fmt.Errorf("jsonpointer part must not be empty: %s", string(j))
		}

		if !tokenRegex.MatchString(part) {
			return nil, fmt.Errorf("jsonpointer part must be a valid token [%s]: %s", tokenRegex.String(), string(j))
		}

		if digitOnlyRegex.MatchString(part) && (len(part) == 1 || part[0] != '0') {
			stack = append(stack, navigationPart{
				Type:  partTypeIndex,
				Value: part,
			})
			continue
		}

		stack = append(stack, navigationPart{
			Type:  partTypeKey,
			Value: part,
		})
	}

	return stack, nil
}
