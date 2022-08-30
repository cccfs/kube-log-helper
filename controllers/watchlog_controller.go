/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

// WatchLogReconciler reconciles a WatchLog object
type WatchLogReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=crd.k8s.deeproute.cn,resources=watchlogs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crd.k8s.deeproute.cn,resources=watchlogs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crd.k8s.deeproute.cn,resources=watchlogs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WatchLog object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *WatchLogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	watchLogInstance := &corev1.Pod{}
	err := r.Client.Get(ctx, req.NamespacedName, watchLogInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		klog.Error(err, "unable to fetch pod")
		return ctrl.Result{}, err
	}

	helper, err := LogHelperInit()
	if err != nil {
		return ctrl.Result{}, err
	}

	clp := &ContainerLogOptions{
		podName:           watchLogInstance.Name,
		podNamespace:      watchLogInstance.Namespace,
		podContainerID:    "",
		containerName:     make([]string, 0),
		containerLogFiles: make([]string, 0),
		containerStatus:   watchLogInstance.Status.Phase,
		nodeName:          watchLogInstance.Spec.NodeName,
	}

	reason := string(watchLogInstance.Status.Phase)
	//skipping for status 'Pending' events
	if reason == string(corev1.PodPending) {
		return ctrl.Result{}, nil
	}

	//var restarts int32 = 0
	//readyContainers := 0
	//
	//if reason != "" {
	//	reason = watchLogInstance.Status.Reason
	//}
	//for i := len(watchLogInstance.Status.ContainerStatuses) - 1; i >= 0; i-- {
	//	container := watchLogInstance.Status.ContainerStatuses[i]
	//
	//	restarts += container.RestartCount
	//	if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
	//		reason = container.State.Waiting.Reason
	//	} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
	//		reason = container.State.Terminated.Reason
	//	} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
	//		if container.State.Terminated.Signal != 0 {
	//			reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
	//		} else {
	//			reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
	//		}
	//	} else if container.Ready && container.State.Running != nil {
	//		readyContainers++
	//	}
	//	fmt.Println("ReasonPod:", container.Name, reason)
	//}
	//if watchLogInstance.DeletionTimestamp != nil {
	//	reason = "Terminating"
	//}
	//fmt.Println("Reason:", reason)
	//switch reason {
	//// skip process 'Pending' status pod
	//case string(corev1.PodPending):
	//	return ctrl.Result{}, nil
	//case string(corev1.PodRunning):
	//	statusContainerStatuses := watchLogInstance.Status.ContainerStatuses
	//	specContainers := watchLogInstance.Spec.Containers
	//	clp.GetContainerLogPath(helper.indexPrefix, statusContainerStatuses, specContainers)
	//case "Terminating":
	//	fmt.Println("status", watchLogInstance.Name)
	//}

	// container current state
	statusContainerStatuses := watchLogInstance.Status.ContainerStatuses
	// container desired state
	specContainers := watchLogInstance.Spec.Containers

	clp.JoinContainerLogPath(helper.indexPrefix, helper.indexSuffix, statusContainerStatuses, specContainers)

	//fmt.Println(clp)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WatchLogReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}

type LogHelperOptions struct {
	indexPrefix []string
	indexSuffix string
}

func LogHelperInit() (*LogHelperOptions, error) {
	// default log index prefix is 'k8s', define LOGGING_INDEX_PREFIX environment variables custom multi index prefix
	prefix := []string{"k8s_logs_"}
	envLoggingPrefix := os.Getenv(EnvLoggingPrefix)
	if envLoggingPrefix != "" {
		prefix = strings.Split(envLoggingPrefix, ",")
	}
	// default log index suffix is 'log', define LOGGING_INDEX_SUFFIX environment variables custom index suffix, e.g: %{+yyyy.MM.dd}
	suffix := "log"
	envLoggingSuffix := os.Getenv(EnvLoggingSuffix)
	if envLoggingSuffix != "" {
		suffix = envLoggingSuffix
	}
	return &LogHelperOptions{
		indexPrefix: prefix,
		indexSuffix: suffix,
	}, nil
}

type ContainerLogOptions struct {
	podName           string
	podNamespace      string
	podContainerID    string
	containerName     []string
	containerLogFiles []string
	containerStatus   corev1.PodPhase
	nodeName          string
}

func (clp *ContainerLogOptions) GetContainerEnv(indexPrefix []string, indexSuffix string, envVar []corev1.EnvVar) error {
	// get all container envVar
	root := newLogInfoNode("")
	for _, env := range envVar {
		// skip envVar that match custom prefix
		for _, prefix := range indexPrefix {
			if !strings.HasPrefix(env.Name, prefix) {
				continue
			}

			trimLogIndexPrefix := strings.TrimPrefix(env.Name, prefix)
			if err := root.insert(strings.Split(trimLogIndexPrefix, "_"), env.Name); err != nil {
				return err
			}
		}
	}

	for name, children := range root.children {
		// cluster multi kube-log-helper env support
		tagsMapContent, err := root.parseTagsContent()
		if err != nil {
			return err
		}
		// e.g: k8s_logs_xxx-xxx-xxx_tags: "env=test"
		if os.Getenv(EnvClusterEnvName) != "" {
			clusterName := os.Getenv(EnvClusterEnvName)
			if tagsMapContent["env"] != clusterName {
				klog.Warning("cluster env with logs tag not match, skipping logs collection")
				break
			}
		}
		// parse filebeat input config
		for _, logFile := range clp.containerLogFiles {
			fmt.Println(logFile)
			joinName := fmt.Sprintf("%s-%s", name, indexSuffix)
			inputConfig, _ := filebeatInputConfigParse(logFile, tagsMapContent, joinName, clp.podName, clp.podNamespace, clp.nodeName, clp.containerName, children)

			fmt.Println(inputConfig)
			joinConfigFile := fmt.Sprintf("%s/%s.yml", FilebeatConfDir, clp.podContainerID)
			if _, err := os.Stat(FilebeatConfDir); os.IsNotExist(err) {
				if err := os.Mkdir(FilebeatConfDir, 0755); err != nil {
					klog.Warningf("%v", err)
				}

			}
			if err := ioutil.WriteFile(joinConfigFile, []byte(inputConfig), 0600); err != nil {
				klog.Exitf("unable to write %s: %v", FilebeatConfDir, err)
			}
		}
	}
	return nil
}

func (clp *ContainerLogOptions) JoinContainerLogPath(indexPrefix []string, indexSuffix string, status []corev1.ContainerStatus, container []corev1.Container) error {
	// csi-driver-d2t4w_gds-csi_csi-driver-4ea36377d2c0dbab0b02a5ffb350b64b4297993394b00e30629c61cd659accfc.log
	// /var/log/containers log format: [pod name]_[namespace]_[container name]-[container id]
	// if container name desired state with current state equal, join the container name log path
	for _, containerList := range container {
		for _, statusList := range status {
			if statusList.Name == containerList.Name {
				clp.podContainerID = strings.Split(statusList.ContainerID, "//")[1]
				clp.containerName = append(clp.containerName, statusList.Name)
				logFormat := fmt.Sprintf("%s/%s_%s_%s-%s.log", EnvLoggingPath, clp.podName, clp.podNamespace, statusList.Name, clp.podContainerID)
				clp.containerLogFiles = append(clp.containerLogFiles, logFormat)
			}
		}
		if err := clp.GetContainerEnv(indexPrefix, indexSuffix, containerList.Env); err != nil {
			return err
		}
	}
	return nil
}
