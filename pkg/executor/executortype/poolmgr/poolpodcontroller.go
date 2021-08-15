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

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	k8sCache "k8s.io/client-go/tools/cache"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
)

type (
	PoolPodController struct {
		logger           *zap.Logger
		kubernetesClient *kubernetes.Clientset
		namespace        string
		enableIstio      bool
		funcInformer     *k8sCache.SharedIndexInformer
		pkgInformer      *k8sCache.SharedIndexInformer
		envInformer      *k8sCache.SharedIndexInformer
	}
)

func NewPoolPodController(logger *zap.Logger,
	kubernetesClient *kubernetes.Clientset,
	namespace string,
	enableIstio bool,
	funcInformer *k8sCache.SharedIndexInformer,
	pkgInformer *k8sCache.SharedIndexInformer,
	envInformer *k8sCache.SharedIndexInformer) *PoolPodController {
	return &PoolPodController{
		logger:           logger,
		kubernetesClient: kubernetesClient,
		namespace:        namespace,
		enableIstio:      enableIstio,
		funcInformer:     funcInformer,
		pkgInformer:      pkgInformer,
		envInformer:      envInformer,
	}
}

func (p PoolPodController) Run(gpm *GenericPoolManager) {
	(*p.funcInformer).AddEventHandler(PoolPodFunctionEventHandlers(p.logger, p.kubernetesClient, p.namespace, p.enableIstio))
	(*p.pkgInformer).AddEventHandler(PoolPodPackageEventHandlers(p.logger, p.kubernetesClient, p.namespace))
	(*p.envInformer).AddEventHandler(NewPoolPodEnvInformerHandlers(p.logger, gpm))
	p.logger.Info("pool pod controller handlers registered")
}

func NewPoolPodEnvInformerHandlers(logger *zap.Logger, gpm *GenericPoolManager) k8sCache.ResourceEventHandlerFuncs {
	return k8sCache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			env := obj.(*fv1.Environment)
			logger = logger.With(zap.String("env", env.Name), zap.String("namespace", env.Namespace))
			logger.Debug("environment created")
			poolsize := getEnvPoolsize(env)
			if poolsize == 0 {
				logger.Info("pool size is zero")
				return
			}
			pool, created, err := gpm.getPool(env)
			if err != nil {
				logger.Error("error getting pool", zap.Error(err))
				return
			}
			if created {
				logger.Info("created pool for the environment")
				return
			}
			err = pool.updatePoolDeployment(context.Background(), env)
			if err != nil {
				logger.Error("error updating pool", zap.Error(err))
			}
			logger.Info("Updated deployment for pool")
		},
		DeleteFunc: func(obj interface{}) {
			env := obj.(*fv1.Environment)
			logger = logger.With(zap.String("env", env.Name), zap.String("namespace", env.Namespace))
			logger.Debug("environment delete", zap.Any("env", env))
			gpm.cleanupPool(env)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldEnv := oldObj.(*fv1.Environment)
			newEnv := newObj.(*fv1.Environment)
			if oldEnv.ObjectMeta.ResourceVersion == newEnv.ObjectMeta.ResourceVersion {
				return
			}
			logger = logger.With(zap.String("env", oldEnv.Name), zap.String("namespace", oldEnv.Namespace))
			logger.Debug("environment update")
			poolsize := getEnvPoolsize(newEnv)
			if poolsize == 0 {
				gpm.cleanupPool(newEnv)
				return
			}
			pool, created, err := gpm.getPool(newEnv)
			if err != nil {
				logger.Error("error getting pool", zap.Error(err))
				return
			}
			if created {
				logger.Info("created pool for the environment", zap.Any("env", newEnv))
				return
			}
			err = pool.updatePoolDeployment(context.Background(), newEnv)
			if err != nil {
				logger.Error("error updating pool", zap.Error(err))
			}
			logger.Info("Updated deployment for pool")

		},
	}
}
