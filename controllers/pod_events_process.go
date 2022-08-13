package controllers

import (
	"fmt"
	"k8s.io/klog/v2"
)

type LogInfoNode struct {
	value    string
	children map[string]*LogInfoNode
}

func newLogInfoNode(value string) *LogInfoNode {
	return &LogInfoNode{
		value:    value,
		children: make(map[string]*LogInfoNode),
	}
}

func (node *LogInfoNode) get(key string) string {
	if child, ok := node.children[key]; ok {
		return child.value
	}
	return ""
}

func (node *LogInfoNode) insert(keys []string, value string) error {
	if len(keys) == 0 {
		return nil
	}
	key := keys[0]
	if len(keys) > 1 {
		if child, ok := node.children[key]; ok {
			child.insert(keys[1:], value)
		} else {
			klog.Warningf("[%s] has no parent index name", key)
		}
	} else {
		child := newLogInfoNode(value)
		node.children[key] = child
	}
	return nil
}

func (node *LogInfoNode) parseTags() (map[string]string, error) {
	// prefix_logs_xxx_tags: "cluster=test"
	tags := node.get("tags")
	return ParseBlocks(tags)
}

func (node *LogInfoNode) parseCustomConfig() (map[string]string, error) {
	// prefix_logs_xxx_config: "multiline.pattern='^[0-9]{3}.*',multiline.negate=true,multiline.match=after"
	config := node.get("config")
	return ParseBlocks(config)
}

func (node *LogInfoNode) parseCovertIndex(tagsMap map[string]string) error {
	// prefix_logs_xxx_index: "project-demo-log"
	indexName := node.get("index")
	if _, ok := tagsMap["index"]; !ok {
		if indexName != "" {
			tagsMap["index"] = indexName
		} else {
			tagsMap["index"] = node.value
		}
	}

	// prefix_logs_xxx_topic: "project-demo-log"
	if _, ok := tagsMap["topic"]; !ok {
		if indexName != "" {
			tagsMap["topic"] = indexName
		} else {
			tagsMap["topic"] = node.value
		}
	}
	return nil
}

func (node *LogInfoNode) parseLogFormat(tagsMap map[string]string) error {
	// prefix_logs_xxx_format: "none|json|csv|nginx|apache2|regexp"
	format := node.children["format"]
	if format == nil || format.value == "none" {
		format = newLogInfoNode("none")
	}
	formatConfig, err := Convert(format)
	if err != nil {
		return err
	}

	if format.value == "regexp" {
		format.value = fmt.Sprintf("/%s/", formatConfig["pattern"])
		delete(formatConfig, "pattern")
	}
	return nil
}

func (node *LogInfoNode) parseDefaultJavaLog() bool {
	// prefix_logs_xxx_java: "true"
	var multiLine bool
	java := node.get("java")
	if java == "true" {
		multiLine = true
	}
	return multiLine
}

//func (node *LogInfoNode) parseLogOutputType(logPath string) (string, error) {
//	// prefix_logs_xxx: "stdout"
//	path := strings.TrimSpace(node.value)
//	if path == "" {
//		return ""
//	}
//
//	// collection container stdout type logs
//	if logPath == "stdout" {
//		logFile := filepath.Base(logPath) + "*"
//	}
//	return Render(FilebeatInputConfTemplate, Data{
//		"Stdout": true,
//		"Multiline": node.parseDefaultJavaLog(),
//		"HostDir":
//	})
//}
