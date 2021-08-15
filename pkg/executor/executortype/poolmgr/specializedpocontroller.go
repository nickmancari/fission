/*
Copyright 2021 The Fission Authors.

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
package poolmgr

import (
	"context"
	"strings"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sCache "k8s.io/client-go/tools/cache"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/executor/fscache"
)

type (
	SpecializedPodController struct {
		logger      *zap.Logger
		envInformer *k8sCache.SharedIndexInformer
	}
)

func getSpecializedPodLabels(env *fv1.Environment) map[string]string {
	specialPodLabels := make(map[string]string)
	specialPodLabels[fv1.EXECUTOR_TYPE] = string(fv1.ExecutorTypePoolmgr)
	specialPodLabels[fv1.ENVIRONMENT_NAME] = env.ObjectMeta.Name
	specialPodLabels[fv1.ENVIRONMENT_NAMESPACE] = env.ObjectMeta.Namespace
	specialPodLabels[fv1.ENVIRONMENT_UID] = string(env.ObjectMeta.UID)
	specialPodLabels["managed"] = "false"
	return specialPodLabels
}

func NewSpecializedPodController(logger *zap.Logger, envInformer *k8sCache.SharedIndexInformer) *SpecializedPodController {
	return &SpecializedPodController{
		logger:      logger,
		envInformer: envInformer,
	}
}

func (spc *SpecializedPodController) Run(gpm *GenericPoolManager) {
	(*spc.envInformer).AddEventHandler(NewSpeciaizedPodEnvInformerHandlers(spc.logger, gpm))
	spc.logger.Info("specialized pod controller handlers registered")
}

func NewSpeciaizedPodEnvInformerHandlers(logger *zap.Logger, gpm *GenericPoolManager) k8sCache.ResourceEventHandlerFuncs {
	return k8sCache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			env := obj.(*fv1.Environment)
			logger.Debug("environment create", zap.Any("env", env))
		},
		DeleteFunc: func(obj interface{}) {
			env := obj.(*fv1.Environment)
			logger = logger.With(zap.String("env", env.Name), zap.String("namespace", env.Namespace))
			logger.Debug("environment delete")
			selectorLabels := getSpecializedPodLabels(env)
			listOptions := metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
			}
			ctx := context.Background()
			podList, err := gpm.kubernetesClient.CoreV1().Pods(gpm.namespace).List(ctx, listOptions)
			if err != nil {
				logger.Error("failed to list pods", zap.Error(err))
				return
			}
			logger.Info("pods identified for cleanup after env delete", zap.Int("numPods", len(podList.Items)))
			for _, pod := range podList.Items {
				podName := strings.SplitAfter(pod.GetName(), ".")
				if fsvc, ok := gpm.fsCache.PodToFsvc.Load(strings.TrimSuffix(podName[0], ".")); ok {
					fsvc, ok := fsvc.(*fscache.FuncSvc)
					if !ok {
						logger.Error("could not covert item from PodToFsvc")
					}
					gpm.fsCache.DeleteFunctionSvc(fsvc)
					gpm.fsCache.DeleteEntry(fsvc)
				}
				err = gpm.kubernetesClient.CoreV1().Pods(gpm.namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
				if err != nil {
					logger.Error("failed to delete pod", zap.Error(err))
					continue
				}
				logger.Info("cleaned specialized pod as environment deleted",
					zap.String("pod", pod.ObjectMeta.Name), zap.String("pod_namespace", pod.ObjectMeta.Namespace),
					zap.String("address", pod.Status.PodIP))
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldEnv := oldObj.(*fv1.Environment)
			newEnv := newObj.(*fv1.Environment)
			if oldEnv.ObjectMeta.ResourceVersion == newEnv.ObjectMeta.ResourceVersion {
				return
			}
			logger = logger.With(zap.String("env", oldEnv.Name), zap.String("namespace", oldEnv.Namespace))
			logger.Debug("environment update")

			// TODO:
			// wait for pool to start using new env and complete update of the deployment
			// depending on deployment rollout we should be able to delete the specialized pods
			// and start using the new env
			// since below code has dependency on poolpodcontroller update handler
			// probaby we should move below code to poolpodcontroller update handler

			selectorLabels := getSpecializedPodLabels(oldEnv)
			listOptions := metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
			}
			ctx := context.Background()
			podList, err := gpm.kubernetesClient.CoreV1().Pods(gpm.namespace).List(ctx, listOptions)
			if err != nil {
				logger.Error("failed to list pods", zap.Error(err))
				return
			}
			logger.Info("pods identified for cleanup after env update", zap.Int("numPods", len(podList.Items)))

			for _, pod := range podList.Items {
				podName := strings.SplitAfter(pod.GetName(), ".")
				if fsvc, ok := gpm.fsCache.PodToFsvc.Load(strings.TrimSuffix(podName[0], ".")); ok {
					fsvc, ok := fsvc.(*fscache.FuncSvc)
					if ok {
						gpm.fsCache.DeleteFunctionSvc(fsvc)
						gpm.fsCache.DeleteEntry(fsvc)
					} else {
						logger.Error("could not covert item from PodToFsvc")
					}
				}
				err = gpm.kubernetesClient.CoreV1().Pods(pod.ObjectMeta.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
				if err != nil {
					logger.Error("failed to delete pod", zap.Error(err))
					continue
				}
				logger.Info("cleaned specialized pod as environment updated",
					zap.String("pod", pod.ObjectMeta.Name), zap.String("pod_namespace", pod.ObjectMeta.Namespace),
					zap.String("address", pod.Status.PodIP))
			}
		},
	}
}
