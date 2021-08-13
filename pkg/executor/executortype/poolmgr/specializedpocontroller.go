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

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
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
	(*spc.envInformer).AddEventHandler(NewSpeciaizedPodEnvInformerHandlers(spc.logger))
	spc.logger.Info("specialized pod controller started")
}

func NewSpeciaizedPodEnvInformerHandlers(logger *zap.Logger) k8sCache.ResourceEventHandlerFuncs {
	return k8sCache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			env := obj.(*fv1.Environment)
			logger.Debug("environment create", zap.Any("env", env))
		},
		DeleteFunc: func(obj interface{}) {
			env := obj.(*fv1.Environment)
			logger.Debug("environment delete", zap.Any("env", env))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldEnv := oldObj.(*fv1.Environment)
			newEnv := newObj.(*fv1.Environment)
			if oldEnv.ObjectMeta.ResourceVersion == newEnv.ObjectMeta.ResourceVersion {
				return
			}
			logger.Debug("environment update", zap.Any("oldEnv", oldEnv), zap.Any("newEnv", newEnv))
		},
	}
}
