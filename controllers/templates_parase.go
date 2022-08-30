package controllers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func filebeatConfigParse() (string, error) {
	return Render(FilebeatConfTemplate, Data{
		"FilebeatLogLevel":              os.Getenv(EnvFilebeatLogLevel),
		"FilebeatMetricsEnabled":        os.Getenv(EnvFilebeatMetricsEnabled),
		"FilebeatFilesRotateeverybytes": os.Getenv(EnvFilebeatFilesRotateeverybytes),
		"FilebeatMaxProcs":              os.Getenv(EnvFilebeatMaxProcs),
		"FilebeatSetupIlmEnabled":       os.Getenv(EnvFilebeatSetupIlmEnabled),
	})
}

func filebeatInputConfigParse(logFile string, tagMap map[string]string, name string, podName, podNamespace, nodeName string, containerName []string, node *LogInfoNode) (string, error) {
	node.parseCovertIndexContent(name, tagMap)
	customConfig, _ := node.parseConfigContent()

	// collection container stdout type logs
	// prefix_logs_xxx: "stdout"
	logOutputType := strings.TrimSpace(node.value)
	if logOutputType == "stdout" {
		return Render(FilebeatInputConfTemplate, Data{
			"Stdout":        true,
			"Multiline":     node.parseDefaultJavaLogContent(),
			"File":          logFile,
			"Format":        node.parseLogFormatContent(),
			"Tags":          tagMap,
			"Metadata":      node.parseMetadataContent(podName, podNamespace, nodeName, containerName),
			"CustomConfigs": customConfig,
		})
	}

	// collection container is absolute path logs
	if !filepath.IsAbs(logFile) {
		return fmt.Sprintf("log file must be absolute path"), nil
	}
	return Render(FilebeatInputConfTemplate, Data{
		"Multiline":     node.parseDefaultJavaLogContent(),
		"File":          logFile,
		"Format":        node.parseLogFormatContent(),
		"Tags":          tagMap,
		"Metadata":      node.parseMetadataContent(podName, podNamespace, nodeName, containerName),
		"CustomConfigs": customConfig,
	})
}
