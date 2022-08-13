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

	if watchLogInstance.Status.Phase == "Pending" {
		return ctrl.Result{}, nil
	}

	clp := &ContainerLogOptions{
		podName:           watchLogInstance.Name,
		namespace:         watchLogInstance.Namespace,
		containerID:       "",
		containerName:     make([]string, 0),
		containerLogPaths: make([]string, 0),
		containerStatus:   watchLogInstance.Status.Phase,
	}
	statusContainerStatuses := watchLogInstance.Status.ContainerStatuses
	specContainers := watchLogInstance.Spec.Containers
	clp.GetContainerLogPath(helper.indexPrefix, statusContainerStatuses, specContainers)

	fmt.Println(clp)
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
}

func LogHelperInit() (*LogHelperOptions, error) {
	// default log index prefix is 'k8s', define LOGGING_INDEX_PREFIX environment variables custom multi index prefix
	prefix := []string{"k8s_logs_"}
	envLoggingPrefix := os.Getenv(EnvLoggingPrefix)
	if envLoggingPrefix != "" {
		prefix = strings.Split(envLoggingPrefix, ",")
	}
	return &LogHelperOptions{
		indexPrefix: prefix,
	}, nil
}

type ContainerLogOptions struct {
	podName           string
	namespace         string
	containerID       string
	containerName     []string
	containerLogPaths []string
	containerStatus   corev1.PodPhase
}

func (clp *ContainerLogOptions) GetContainerEnv(indexPrefix []string, envVar []corev1.EnvVar) error {
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
		tagsMap, err := root.parseTags()
		if err != nil {
			return err
		}
		// e.g: k8s_logs_xxx-xxx-xxx_tags: "env=test"
		if os.Getenv(EnvClusterEnvName) != "" {
			clusterName := os.Getenv(EnvClusterEnvName)
			if tagsMap["env"] != clusterName {
				klog.Warning("cluster env with logs tag not match, skipping logs collection")
				break
			}
		}

		fmt.Println("children:", name, children)
	}
	return nil
}

func (clp *ContainerLogOptions) GetContainerLogPath(indexPrefix []string, status []corev1.ContainerStatus, container []corev1.Container) error {
	// csi-driver-d2t4w_gds-csi_csi-driver-4ea36377d2c0dbab0b02a5ffb350b64b4297993394b00e30629c61cd659accfc.log
	// /var/log/containers log format: [pod name]_[namespace]_[container name]-[container id]
	for _, containerList := range container {
		for _, statusList := range status {
			if statusList.Name == containerList.Name {
				clp.containerID = strings.Split(statusList.ContainerID, "//")[1]
				clp.containerName = append(clp.containerName, statusList.Name)
				logFormat := fmt.Sprintf("%s/%s_%s_%s-%s.log", EnvLoggingPath, clp.podName, clp.namespace, statusList.Name, clp.containerID)
				clp.containerLogPaths = append(clp.containerLogPaths, logFormat)
			}
		}
		if err := clp.GetContainerEnv(indexPrefix, containerList.Env); err != nil {
			return err
		}
	}
	return nil
}
