package controllers

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cccfs/kube-log-helper/pkg/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/kris-nova/logger"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 处理容器启动和删除事件
func (h *helper) processContainerEvent(msg events.Message) error {
	containerID := msg.Actor.ID
	//configPaths := h.controller.ConfPath(containerID)
	exists, configPaths := h.containerConfigPathsExists(containerID)

	//fmt.Println(msg.Action)
	switch msg.Action {
	case "start":
		// 容器配置文件存在则不处理下面生成容器配置逻辑
		if exists {
			return nil
		}
		logger.Debug("discover [%v] event container: [%v]", msg.Action, containerID)
		containerJSON, err := h.client.ContainerInspect(context.Background(), containerID)
		if err != nil {
			return err
		}
		return h.newContainerConfig(&containerJSON)
	case "die":
		logger.Debug("discover [%v] event container: [%v]", msg.Action, containerID)
		return h.controller.DieEvent(containerID, configPaths)
	}
	return nil
}

func (h *helper) newContainerConfig(containerJSON *types.ContainerJSON) error {
	containerID := containerJSON.ID
	containerLogPath := containerJSON.LogPath
	mounts := containerJSON.Mounts
	env := containerJSON.Config.Env
	labels := containerJSON.Config.Labels

	for _, e := range env {
		for _, prefix := range h.logPrefix {
			serviceLogs := fmt.Sprintf("%s_logs_", prefix)
			if !strings.HasPrefix(e, serviceLogs) {
				continue
			}

			envLabel := strings.SplitN(e, "=", 2)
			if len(envLabel) == 2 {
				labelKey := strings.Replace(envLabel[0], "_", ".", -1)
				labels[labelKey] = envLabel[1]
			}
		}
	}

	var labelNames []string
	//sort keys
	for k := range labels {
		labelNames = append(labelNames, k)
	}

	sort.Strings(labelNames)
	root := newLogInfoNode("")
	for _, k := range labelNames {
		for _, prefix := range h.logPrefix {
			serviceLogs := fmt.Sprintf("%s.logs.", prefix)
			if !strings.HasPrefix(k, serviceLogs) || strings.Count(k, ".") == 1 {
				continue
			}

			logLabel := strings.TrimPrefix(k, serviceLogs)
			if err := root.insert(strings.Split(logLabel, "."), labels[k]); err != nil {
				return err
			}
		}
	}

	var configList []*ConfigList

	for name, children := range root.children {
		logConfig, err := h.processLogConfig(name, containerLogPath, children, mounts)
		if err != nil {
			return err
		}
		logger.Info("discover new container logs collection requests: [%v] = %v", name, fmt.Sprintf("%v/%v", logConfig.HostDir, logConfig.File))
		configList = append(configList, logConfig)
	}

	if len(configList) == 0 {
		logger.Debug("%s has not log config, skipping", containerID)
		return nil
	}
	// format filebeat.tpl template
	var buffer bytes.Buffer
	data := map[string]interface{}{
		"configList":	configList,
		"container": 	getContainerLabels(containerJSON),
	}
	if err := h.template.Execute(&buffer, data); err != nil {
		return err
	}

	// generate container input config file
	if err := ioutil.WriteFile(h.controller.ConfPath(containerID), []byte(buffer.String()), os.FileMode(0644)); err != nil {
		return err
	}

	return nil
}

func getContainerLabels(containerJSON *types.ContainerJSON) map[string]string {
	labels := containerJSON.Config.Labels
	l := make(map[string]string)
	utils.PutIfNotEmpty(l, "k8s_pod", labels["io.kubernetes.pod.name"])
	utils.PutIfNotEmpty(l, "k8s_pod_namespace", labels["io.kubernetes.pod.namespace"])
	utils.PutIfNotEmpty(l, "k8s_pod_name", labels["io.kubernetes.container.name"])
	utils.PutIfNotEmpty(l, "k8s_node_name", os.Getenv("NODE_NAME"))
	//utils.PutIfNotEmpty(l, "docker_container", strings.TrimPrefix(containerJSON.Name, "/"))
	return l
}

type LogInfoNode struct {
	value    string
	children map[string]*LogInfoNode
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
			return fmt.Errorf("%s has no parent node", key)
		}
	} else {
		child := newLogInfoNode(value)
		node.children[key] = child
	}
	return nil
}

func newLogInfoNode(value string) *LogInfoNode {
	return &LogInfoNode{
		value:    value,
		children: make(map[string]*LogInfoNode),
	}
}

type ConfigList struct {
	Stdout       	bool
	Multiline	 	bool
	HostDir      	string
	File         	string
	Format 			string
	Tags         	map[string]string
	CustomConfigs 	map[string]string
}

func (h *helper) processLogConfig(name, containerLogPath string, info *LogInfoNode, mounts []types.MountPoint) (*ConfigList, error) {
	// prefix_logs_xxx_tags: "env=test,cluster=test"
	tags := info.get("tags")
	tagMap, err := utils.ParseBlocks(tags)
	if err != nil {
		return nil, errors.Wrap(err, "parse tags for error")
	}

	// prefix_logs_xxx_config: ""
	customConfigs := info.get("config")
	customConfigMap, err := utils.ParseBlocks(customConfigs)
	if err != nil {
		return nil, errors.Wrap(err, "parse custom configs for error")
	}

	target := info.get("target")
	// add default index and topic
	if _, ok := tagMap["index"]; !ok {
		if target != "" {
			tagMap["index"] = target
		} else {
			tagMap["index"] = name
		}
	}

	if _, ok := tagMap["topic"]; !ok {
		if target != "" {
			tagMap["topic"] = target
		} else {
			tagMap["topic"] = name
		}
	}

	format := info.children["format"]
	if format == nil || format.value == "none" {
		format = newLogInfoNode("none")
	}
	formatConfig, err := Convert(format)
	if err != nil {
		return nil, errors.Wrap(err, "log format for error")
	}

	//process regex
	if format.value == "regexp" {
		format.value = fmt.Sprintf("/%s/", formatConfig["pattern"])
		delete(formatConfig, "pattern")
	}

	//enable multi line process java logs
	var multiLine bool
	java := info.get("java")
	if java == "true" {
		multiLine = true
	}

	logPath := strings.TrimSpace(info.value)
	if logPath == "" {
		return nil, errors.Wrap(err, "log path is empty")
	}

	// collection container stdout type logs
	if logPath == "stdout" {
		logFile := filepath.Base(containerLogPath) + "*"

		return &ConfigList{
			HostDir:      	filepath.Join(h.baseDir, filepath.Dir(containerLogPath)),
			File:         	logFile,
			Format:       	format.value,
			Tags:         	tagMap,
			CustomConfigs: 	customConfigMap,
			Stdout:       	true,
			Multiline:    	multiLine,
		}, nil
	}

	// collection container custom path logs
	if !filepath.IsAbs(logPath) {
		return nil, errors.Wrap(err, "logs path must be absolute path")
	}

	containerDir := filepath.Dir(logPath)
	file := filepath.Base(logPath)
	if file == "" {
		return nil, errors.Wrap(err, "must be a file path, not directory")
	}

	mountsMap := make(map[string]types.MountPoint)
	for _, mount := range mounts {
		mountsMap[mount.Destination] = mount
	}

	hostDir := h.hostDirOf(containerDir, mountsMap)
	if hostDir == "" {
		return nil, errors.Wrap(err, "log is not mount on host")
	}

	return &ConfigList{
		File:         file,
		Format:       format.value,
		Tags:         tagMap,
		CustomConfigs: customConfigMap,
		HostDir:      filepath.Join(h.baseDir, hostDir),
		Multiline:    multiLine,
	}, nil
}

func (h *helper) hostDirOf(path string, mounts map[string]types.MountPoint) string {
	confPath := path
	for {
		if point, ok := mounts[path]; ok {
			if confPath == path {
				return point.Source
			}

			relPath, err := filepath.Rel(path, confPath)
			if err != nil {
				panic(err)
			}
			return fmt.Sprintf("%s/%s", point.Source, relPath)
		}
		path = filepath.Dir(path)
		if path == "/" || path == "." {
			break
		}
	}
	return ""
}