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
	"go.uber.org/zap"
	k8sCache "k8s.io/client-go/tools/cache"
)

type (
	SpecializedPodController struct {
		logger      *zap.Logger
		envInformer *k8sCache.SharedIndexInformer
	}
)

func NewSpecializedPodController(logger *zap.Logger, envInformer *k8sCache.SharedIndexInformer) *SpecializedPodController {
	return &SpecializedPodController{
		logger:      logger,
		envInformer: envInformer,
	}
}

func (spc *SpecializedPodController) Run() {
	(*spc.envInformer).AddEventHandler(NewSpeciaizedPodEnvInformerHandlers())
	spc.logger.Info("specialized pod controller started")
}

func NewSpeciaizedPodEnvInformerHandlers() k8sCache.ResourceEventHandlerFuncs {
	return k8sCache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
		},
		DeleteFunc: func(obj interface{}) {
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
		},
	}
}
