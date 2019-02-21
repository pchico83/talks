package model

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"bitbucket.org/okteto/okteto/backend/logger"

	yaml "gopkg.in/yaml.v2"
)

var cannotUnmarshall = regexp.MustCompile(`line (\d*): cannot unmarshal (!!.*) into (.*)`)

func parseManifest(m []byte) (*Service, *AppError) {
	var service Service
	err := yaml.Unmarshal(m, &service)

	if err != nil {
		if v, ok := err.(*yaml.TypeError); ok {
			return nil, translateYamlTypeError(v)
		}

		return nil, &AppError{Status: 400, Code: InvalidYAML, Message: err.Error()}
	}

	return &service, service.Validate()
}

func translateYamlTypeError(err *yaml.TypeError) *AppError {
	if len(err.Errors) > 0 {
		if matches := cannotUnmarshall.FindStringSubmatch(err.Errors[0]); matches != nil {
			received := "unknown"

			// the error for a string includes the value (e.g. !!str `hello`), we ignore it.
			recievedType := strings.Split(matches[2], " ")[0]
			switch recievedType {
			case "!!seq":
				received = "sequence"

			case "!!map":
				received = "map"

			case "!!str":
				received = "string"

			case "!!bool":
				received = "boolean"
			default:
				logger.Error(fmt.Errorf("unknown received type: '%s' \n yaml error: %s", matches[2], err.Errors[0]))
			}

			expected := "map"
			if strings.Contains(matches[3], "[]") {
				expected = "sequence"
			} else {
				switch matches[3] {
				case "string":
					expected = "string"
				case "bool":
					expected = "boolean"
				default:
					expected = "map"
				}
			}

			return &AppError{
				Status:  400,
				Code:    InvalidYAMLWithInfo,
				Message: err.Error(),
				Data: map[string]string{
					"line":     matches[1],
					"expected": expected,
					"received": received,
				},
			}
		}

	}

	return &AppError{Status: 400, Code: InvalidYAML, Message: err.Error()}
}

// ParseEncodedManifest decodes m and returns a instance of Service
func ParseEncodedManifest(m string) (*Service, *AppError) {
	decodedManifest, err := base64.StdEncoding.DecodeString(m)
	if err != nil {
		return nil, &AppError{Status: 400, Code: InvalidBase64}
	}

	return parseManifest(decodedManifest)
}
