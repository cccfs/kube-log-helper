package controllers

import (
	"context"
	"github.com/cccfs/kube-log-helper/pkg/filebeat"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	docker "github.com/docker/docker/client"
	"github.com/kris-nova/logger"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"
)

func Run(tplName, baseDir string) error {
	tplByte, err := ioutil.ReadFile(tplName)
	if err != nil {
		log.Fatal(err)
	}
	h, err := newRun(string(tplByte), baseDir)
	if err != nil {
		log.Fatal(err)
	}
	return h.watch()
}

type helper struct {
	mutex      		sync.Mutex
	template   		*template.Template
	client     		*docker.Client
	lastReload 		time.Time
	stopChan   		chan bool
	baseDir    		string
	logPrefix  		[]string
	controller 		filebeat.Ctr
}

// 初始化默认参数
func newRun(tplStr, baseDir string) (*helper, error)  {
	// parse filebeat.tpl files
	tpl, err := template.New("filebeatTemplate").Parse(tplStr)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := docker.NewClientWithOpts(docker.FromEnv)
	if err != nil {
		return nil, err
	}
	client.NegotiateAPIVersion(ctx)

	ctr, err := filebeat.NewFilebeat(baseDir)
	if err != nil {
		return nil, err
	}

	logPrefix := []string{"k8s"}
	if os.Getenv(ENV_LOGGING_PREFIX) != "" {
		envLogPrefix := os.Getenv(ENV_LOGGING_PREFIX)
		logPrefix = strings.Split(envLogPrefix, ",")
	}
	return &helper{
		client: 			client,
		template: 			tpl,
		baseDir: 			baseDir,
		stopChan: 			make(chan bool),
		logPrefix: 			logPrefix,
		controller: 		ctr,
	}, nil
}

func (h *helper) watch() error {
	err := h.controller.Start()
	if err != nil && ERR_ALREADY_STARTED != err.Error() {
		return err
	}

	ctx := context.Background()
	filter := filters.NewArgs()
	filter.Add("type", "container")
	options := types.EventsOptions{
		Filters: filter,
	}

	if err := h.processAllContainerEvent(); err != nil {
		return err
	}

	// found container events
	msgs, errs := h.client.Events(ctx, options)
	go func() {
		defer func() {
			logger.Warning("program exception exit ...")
			h.stopChan <- true
		}()

		for {
			select {
			case msg := <-msgs:

				//logger.Debug("print container event: %v", msg)
				// 监听容器的事件响应
				if err := h.processContainerEvent(msg); err != nil {
					logger.Critical("failure to process event: %v", msg, err)
				}
			case err := <-errs:
				logger.Warning("error: %v", err)
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					return
				}
				msgs, errs = h.client.Events(ctx, options)
			}
		}
	}()

	<-h.stopChan
	close(h.stopChan)
	return nil
}

func (h *helper) processAllContainerEvent() error {
	logger.Info("begin to collection node container logs event")

	opts := types.ContainerListOptions{}
	containerList, err := h.client.ContainerList(context.Background(), opts)
	if err != nil {
		return err
	}

	for _, container := range containerList {
		//logger.Debug("container status: %v %v", container.ID, container.State)
		// 跳过removing状态的容器
		if container.State == "removing" {
			continue
		}

		// 跳过已经有配置文件的容器
		if ok, _ := h.containerConfigPathsExists(container.ID); ok {
			logger.Debug("%v config is already exists", container.ID)
			continue
		}

		containerJSON, err := h.client.ContainerInspect(context.Background(), container.ID)
		if err != nil {
			continue
		}

		//logger.Debug("container name with id: %v %v", containerJSON.Name, containerJSON.ID)

		if err = h.newContainerConfig(&containerJSON); err != nil {
			return err
		}
	}
	return nil
}

func (h *helper) containerConfigPathsExists(container string) (bool, string) {
	configPaths := h.controller.ConfPath(container)
	if _, err := os.Stat(configPaths); os.IsNotExist(err) {
		return false, ""
	}
	return true, configPaths
}