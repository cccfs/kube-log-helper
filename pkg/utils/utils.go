package utils

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"text/template"
	"gopkg.in/yaml.v3"
)

func ToYAML(v interface{}) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		// Swallow errors inside of a template.
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}

func Indent(n int, text string) string {
	startOfLine := regexp.MustCompile(`(?m)^`)
	indentation := strings.Repeat(" ", n)
	return startOfLine.ReplaceAllLiteralString(text, indentation)
}

func ReadFile(path string, separator string) ([]string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(data), separator), nil
}

func Render(tmpl *template.Template, variables map[string]interface{}) (string, error) {

	var buf strings.Builder

	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", errors.Wrap(err, "Failed to render template")
	}
	return buf.String(), nil
}

type Data map[string]interface{}

func PutIfNotEmpty(store map[string]string, key, value string) {
	if key == "" || value == "" {
		return
	}
	store[key] = value
}

func Exists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func ParseBlocks(blocks string) (map[string]string, error) {
	blockMap := make(map[string]string)
	if blocks == "" {
		return blockMap, nil
	}

	kvArray := strings.Split(blocks, ",")
	for _, kv := range kvArray {
		arr := strings.Split(kv, "=")
		if len(arr) != 2 {
			return nil, fmt.Errorf("%s is not a valid k=v format", kv)
		}
		key := strings.TrimSpace(arr[0])
		value := strings.TrimSpace(arr[1])
		if key == "" || value == "" {
			return nil, fmt.Errorf("%s is not a valid k=v format", kv)
		}
		blockMap[key] = value
	}
	return blockMap, nil
}