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

func (node *LogInfoNode) parseTagsContent() (map[string]string, error) {
	// prefix_logs_xxx_tags: "cluster=test"
	tags := node.get("tags")
	return ParseBlocks(tags)
}

func (node *LogInfoNode) parseConfigContent() (map[string]string, error) {
	// prefix_logs_xxx_config: "multiline.pattern='^[0-9]{3}.*',multiline.negate=true,multiline.match=after"
	config := node.get("config")
	return ParseBlocks(config)
}

func (node *LogInfoNode) parseCovertIndexContent(name string, tagsMap map[string]string) error {
	// prefix_logs_xxx_index: "project-demo-log"
	indexName := node.get("index")
	if _, ok := tagsMap["index"]; !ok {
		if indexName != "" {
			tagsMap["index"] = indexName
		} else {
			tagsMap["index"] = name
		}
	}

	// prefix_logs_xxx_topic: "project-demo-log"
	if _, ok := tagsMap["topic"]; !ok {
		if indexName != "" {
			tagsMap["topic"] = indexName
		} else {
			tagsMap["topic"] = name
		}
	}
	return nil
}

func (node *LogInfoNode) parseMetadataContent(podName, podNamespace, nodeName string, containerName []string) map[string]string {
	metadata := make(map[string]string)
	for _, container := range containerName {
		PutIfNotEmpty(metadata, "k8s_pod", podName)
		PutIfNotEmpty(metadata, "k8s_pod_namespace", podNamespace)
		PutIfNotEmpty(metadata, "k8s_pod_container_name", container)
		PutIfNotEmpty(metadata, "k8s_node_name", nodeName)
	}
	return metadata
}

func (node *LogInfoNode) parseLogFormatContent() error {
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

func (node *LogInfoNode) parseDefaultJavaLogContent() bool {
	// prefix_logs_xxx_java: "true"
	var multiLine bool
	java := node.get("java")
	if java == "true" {
		multiLine = true
	}
	return multiLine
}
