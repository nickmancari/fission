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
	"k8s.io/client-go/kubernetes"
	k8sCache "k8s.io/client-go/tools/cache"
)

type (
	PoolPodController struct {
		logger           *zap.Logger
		kubernetesClient *kubernetes.Clientset
		namespace        string
		enableIstio      bool
		funcInformer     *k8sCache.SharedIndexInformer
		pkgInformer      *k8sCache.SharedIndexInformer
	}
)

func NewPoolPodController(logger *zap.Logger,
	kubernetesClient *kubernetes.Clientset,
	namespace string,
	enableIstio bool,
	funcInformer *k8sCache.SharedIndexInformer,
	pkgInformer *k8sCache.SharedIndexInformer) *PoolPodController {
	return &PoolPodController{
		logger:           logger,
		kubernetesClient: kubernetesClient,
		namespace:        namespace,
		enableIstio:      enableIstio,
		funcInformer:     funcInformer,
		pkgInformer:      pkgInformer}
}

func (p PoolPodController) Run() {
	(*p.funcInformer).AddEventHandler(FunctionEventHandlers(p.logger, p.kubernetesClient, p.namespace, p.enableIstio))
	(*p.pkgInformer).AddEventHandler(PackageEventHandlers(p.logger, p.kubernetesClient, p.namespace))
	p.logger.Info("pool pod controller started")
}
