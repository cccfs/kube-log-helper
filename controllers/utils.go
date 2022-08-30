package controllers

import (
	"fmt"
	"github.com/pkg/errors"
	"strings"
	"text/template"
)

func Render(tmpl *template.Template, variables map[string]interface{}) (string, error) {

	var buf strings.Builder

	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", errors.Wrap(err, "Failed to render template")
	}
	return buf.String(), nil
}

type Data map[string]interface{}

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

func PutIfNotEmpty(store map[string]string, key, value string) {
	if key == "" || value == "" {
		return
	}
	store[key] = value
}
