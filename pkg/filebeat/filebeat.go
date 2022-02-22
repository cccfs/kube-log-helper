package filebeat

import (
	"encoding/json"
	"fmt"
	"github.com/cccfs/kube-log-helper/pkg/utils"
	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/yaml"
	"github.com/kris-nova/logger"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Ctr interface {
	Name()							string
	Start() 						error
	Stop()							error
	ConfPath(container string)		string
	GetBaseConf()					string
	DieEvent(container, configPaths string)	error
}

type filebeat struct {
	name           			string
	baseDir        			string
	watchDone      			chan bool
	watchDuration  			time.Duration
	watchContainer 			map[string]string  //map[container][container]
	destroyQueue
}

type destroyQueue struct {
	logSize int64
	logFile string
}

func NewFilebeat(baseDir string) (Ctr, error) {
	return &filebeat{
		name:           		"filebeat",
		baseDir:        		baseDir,
		watchDone:      		make(chan bool),
		watchContainer: 		make(map[string]string, 0),
		watchDuration:  		10 * time.Second,
	}, nil
}

func (f *filebeat) Name() string {
	return f.name
}

// 启动filebeat服务并持续接收容器事件处理
func (f *filebeat) Start() error {
	cmd := exec.Command(FILEBEAT_EXEC_CMD, "-c", FILEBEAT_CONF_FILE)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()

	go func() {
		logger.Info("filebeat startup, process pid is %v", cmd.Process.Pid)
		if err := cmd.Wait(); err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				// filebeat服务进程不存在将退出程序
				if err := f.Stop(); err != nil {
					logger.Critical("%v", err)
				}
				logger.Critical("filebeat exited, err: %v", err.(*exec.ExitError))
			}
		}
	}()
	go f.watchNewContainer()
	return err
}

func (f *filebeat) watchNewContainer() error  {
	for {
		select {
		case <-f.watchDone:
			logger.Critical("%s watcher stop", f.Name())
			os.Exit(-1)
			return nil
		case <-time.After(f.watchDuration):
			logger.Info("scan for container event")
			err := f.watchEventProcess()
			if err != nil {
				logger.Critical("%s watcher scan error: %v", f.Name(), err)
			}
		}
	}
}

func (f *filebeat) watchEventProcess() error {
	// 处理容器die事件
	for container := range f.watchContainer {
		logger.Debug("waiting destroy config event queue: [%v]", f.watchContainer[container])

		configPaths := f.ConfPath(container)
		if ok, _ := f.processRemoveConf(container); ok {
			logger.Info("remove container filebeat config %s", configPaths)
			if err := os.Remove(configPaths); err != nil {
				logger.Warning("remove container filebeat config %s failure", configPaths)
			}
			delete(f.watchContainer, container)
		} else {
			logger.Debug("waiting container log %v write done ...", configPaths)
		}
	}

	return nil
}

type config []struct {
	Paths []string `config:"paths"`
}

// 移除配置文件前需要先检测日志文件size与registry offset是否一致，这是判断日志完整性的策略.
func (f *filebeat) processRemoveConf(container string) (bool, error) {
	registryState, err := f.getRegistryState()
	if err != nil {
		return false, err
	}

	//if _, ok := registryState[f.logFile]; !ok {
	//	logger.Warning("[%s] %s registry not exist", container, f.logFile)
	//	return true, nil
	//}

	if registryState[f.logFile].Offset < f.logSize {
		logger.Debug("[%s] %s does not finish to read, current log offset vs registry state offset: [%v|%v],", container, f.logFile, f.logSize, registryState[f.logFile].Offset)
		return false, nil
	}

	logger.Info("logs finish to read")
	return true, nil
}

// RegistryState registry data.json field
type RegistryState struct {
	Source      string        `json:"source"`
	Offset      int64         `json:"offset"`
	Timestamp   time.Time     `json:"timestamp"`
	TTL         time.Duration `json:"ttl"`
	Type        string        `json:"type"`
	FileStateOS FileInode
}

type FileInode struct {
	Inode  uint64 `json:"inode,"`
	Device uint64 `json:"device,"`
}

func (f *filebeat) getRegistryState() (map[string]RegistryState, error) {
	file, err := os.Open(FILEBEAT_REGISTRY + "/data.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	states := make([]RegistryState, 0)
	err = decoder.Decode(&states)
	if err != nil {
		return nil, err
	}

	statesMap := make(map[string]RegistryState, 0)
	for _, state := range states {
		if _, ok := statesMap[state.Source]; !ok {
			statesMap[state.Source] = state
		}
	}
	return statesMap, nil
}

func (f *filebeat) Stop() error {
	f.watchDone <- true
	return nil
}

func (f *filebeat) DieEvent(container, configPaths string) error {
	// 获取容器die、destroy事件，追加到f.watchContainer当中
	if _, ok := f.watchContainer[container]; !ok {
		f.watchContainer[container] = container
	}

	if ok, _ := utils.Exists(configPaths); !ok {
		// 删除无配置文件的容器
		delete(f.watchContainer, container)
		logger.Debug("remove lost container: %v", container)
	} else {
		if err := f.parseConfigPaths(configPaths); err != nil {
			return err
		}
	}
	return nil
}

func (f *filebeat) parseConfigPaths(configPaths string) error {
	// 解析die事件容器的日志路径并记录总的日志数量
	parseConfig, err := yaml.NewConfigWithFile(configPaths, ucfg.PathSep("."))
	if err != nil {
		return err
	}
	var config config
	if err := parseConfig.Unpack(&config); err != nil {
		return err
	}

	var confPaths []string
	for _, conf := range config {
		confPaths = conf.Paths
	}

	for _, paths := range confPaths {
		// return all match container log config
		logFilePaths, _ := filepath.Glob(paths)
		for _, logFilePath := range logFilePaths {
			info, err := os.Stat(logFilePath)
			if err != nil && os.IsNotExist(err){
				continue
			}

			f.logSize = info.Size()
			f.logFile = logFilePath
		}
	}
	return nil
}

func (f *filebeat) ConfPath(container string) string {
	return fmt.Sprintf("%s/%s.yml", FILEBEAT_CONF_DIR, container)
}

func (f *filebeat) GetBaseConf() string {
	return FILEBEAT_BASE_CONF
}