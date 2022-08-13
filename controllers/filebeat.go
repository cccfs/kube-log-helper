package controllers

import (
	"io/ioutil"
	"k8s.io/klog/v2"
	"os"
	"os/exec"
	"time"
)

type LogHelperEntry struct {
	filebeatCtrl FilebeatCtrlInterface
}

func Run() error {
	_, err := BeforeRun()
	//logHelper, err := BeforeRun()
	if err != nil {
		return err
	}

	// development mode annotations the feature
	//err = logHelper.filebeatCtrl.StartFilebeat()
	//if err != nil && AlreadyStartedError != err.Error() {
	//	return err
	//}
	return nil
}

func BeforeRun() (*LogHelperEntry, error) {
	ctrl, err := InitFilebeat()
	if err != nil {
		return nil, err
	}
	return &LogHelperEntry{
		filebeatCtrl: ctrl,
	}, nil
}

type FilebeatCtrlInterface interface {
	StartFilebeat() error
	StopFilebeat() error
}

type FilebeatCtrlOptions struct {
	watchDone      chan bool
	watchDuration  time.Duration
	watchContainer map[string]string
}

func InitFilebeat() (FilebeatCtrlInterface, error) {
	config, err := filebeatConfigParse()
	if err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(FilebeatConf, []byte(config), 0600); err != nil {
		klog.Exitf("unable to write %s: %v", FilebeatConf, err)
	}
	return &FilebeatCtrlOptions{
		watchDone:      make(chan bool),
		watchDuration:  10 * time.Second,
		watchContainer: make(map[string]string, 0),
	}, nil
}

func (f *FilebeatCtrlOptions) StopFilebeat() error {
	f.watchDone <- true
	return nil
}

func (f *FilebeatCtrlOptions) StartFilebeat() error {
	cmd := exec.Command(FilebeatBin, "-c", FilebeatConf)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()

	// wait filebeat start success
	go func() {
		if err := cmd.Wait(); err != nil {
			// filebeat start failed program exit
			if _, ok := err.(*exec.ExitError); ok {
				f.StopFilebeat()
			}
			klog.Exitf("filebeat exited: %v", err.(*exec.ExitError))
		}
	}()

	go f.watchContainerLoop()

	return err
}

func (f *FilebeatCtrlOptions) watchContainerLoop() error {
	for {
		select {
		case <-f.watchDone:
			klog.Exitf("filebeat watcher stop")
			return nil
		case <-time.After(f.watchDuration):
			// get for queue container events
		}
	}
}
