/*
Copyright The Kubernetes Authors.

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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	time "time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	nfssharecontroller_v1alpha1 "github.com/piersharding/nfsshare-controller/pkg/apis/nfssharecontroller/v1alpha1"
	versioned "github.com/piersharding/nfsshare-controller/pkg/client/clientset/versioned"
	internalinterfaces "github.com/piersharding/nfsshare-controller/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/piersharding/nfsshare-controller/pkg/client/listers/nfssharecontroller/v1alpha1"
)

// NfsshareInformer provides access to a shared informer and lister for
// Nfsshares.
type NfsshareInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.NfsshareLister
}

type nfsshareInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewNfsshareInformer constructs a new informer for Nfsshare type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory nfssharetprint and number of connections to the server.
func NewNfsshareInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredNfsshareInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredNfsshareInformer constructs a new informer for Nfsshare type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory nfssharetprint and number of connections to the server.
func NewFilteredNfsshareInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NfssharecontrollerV1alpha1().Nfsshares(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NfssharecontrollerV1alpha1().Nfsshares(namespace).Watch(options)
			},
		},
		&nfssharecontroller_v1alpha1.Nfsshare{},
		resyncPeriod,
		indexers,
	)
}

func (f *nfsshareInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredNfsshareInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *nfsshareInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&nfssharecontroller_v1alpha1.Nfsshare{}, f.defaultInformer)
}

func (f *nfsshareInformer) Lister() v1alpha1.NfsshareLister {
	return v1alpha1.NewNfsshareLister(f.Informer().GetIndexer())
}
