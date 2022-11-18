// Copyright 2022 TriggerMesh Inc.
// SPDX-License-Identifier: Apache-2.0
// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	internalinterfaces "github.com/triggermesh/triggermesh-core/pkg/client/generated/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// MemoryBrokers returns a MemoryBrokerInformer.
	MemoryBrokers() MemoryBrokerInformer
	// RedisBrokers returns a RedisBrokerInformer.
	RedisBrokers() RedisBrokerInformer
	// Triggers returns a TriggerInformer.
	Triggers() TriggerInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// MemoryBrokers returns a MemoryBrokerInformer.
func (v *version) MemoryBrokers() MemoryBrokerInformer {
	return &memoryBrokerInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// RedisBrokers returns a RedisBrokerInformer.
func (v *version) RedisBrokers() RedisBrokerInformer {
	return &redisBrokerInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Triggers returns a TriggerInformer.
func (v *version) Triggers() TriggerInformer {
	return &triggerInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
