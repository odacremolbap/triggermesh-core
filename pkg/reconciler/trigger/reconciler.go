// Copyright 2022 TriggerMesh Inc.
// SPDX-License-Identifier: Apache-2.0

package trigger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
	"knative.dev/pkg/resolver"

	eventingv1alpha1 "github.com/triggermesh/triggermesh-core/pkg/apis/eventing/v1alpha1"
	eventingv1alpha1listers "github.com/triggermesh/triggermesh-core/pkg/client/generated/listers/eventing/v1alpha1"
	"github.com/triggermesh/triggermesh-core/pkg/reconciler/common"
)

type Reconciler struct {
	// TODO duck brokers
	rbLister    eventingv1alpha1listers.RedisBrokerLister
	mbLister    eventingv1alpha1listers.MemoryBrokerLister
	uriResolver *resolver.URIResolver
}

func (r *Reconciler) ReconcileKind(ctx context.Context, t *eventingv1alpha1.Trigger) pkgreconciler.Event {
	err := r.resolveBroker(ctx, t)
	if err != nil {
		return err
	}

	err = r.resolveTarget(ctx, t)
	if err != nil {
		return err
	}

	return r.resolveDLS(ctx, t)
}

func (r *Reconciler) resolveBroker(ctx context.Context, t *eventingv1alpha1.Trigger) pkgreconciler.Event {
	// TODO duck
	// TODO move to webhook
	switch {
	case t.Spec.Broker.Group == "":
		t.Spec.Broker.Group = eventingv1alpha1.SchemeGroupVersion.Group
	case t.Spec.Broker.Group != eventingv1alpha1.SchemeGroupVersion.Group:
		controller.NewPermanentError(fmt.Errorf("not supported Broker Group %q", t.Spec.Broker.Group))
	}

	var rb *eventingv1alpha1.RedisBroker
	if t.Spec.Broker.Kind == rb.GetGroupVersionKind().Kind {
		return r.resolveRedisBroker(ctx, t)
	}

	var mb *eventingv1alpha1.MemoryBroker
	if t.Spec.Broker.Kind != mb.GetGroupVersionKind().Kind {
		return controller.NewPermanentError(fmt.Errorf("not supported Broker Kind %q", t.Spec.Broker.Kind))
	}

	return r.resolveMemoryBroker(ctx, t)
}

func (r *Reconciler) resolveRedisBroker(ctx context.Context, t *eventingv1alpha1.Trigger) pkgreconciler.Event {
	rb, err := r.rbLister.RedisBrokers(t.Namespace).Get(t.Spec.Broker.Name)
	if err != nil {
		if apierrs.IsNotFound(err) {
			logging.FromContext(ctx).Errorw(fmt.Sprintf("Trigger %s/%s references non existing broker %q", t.Namespace, t.Name, t.Spec.Broker.Name))
			t.Status.MarkBrokerFailed(common.ReasonBrokerDoesNotExist, "Broker %q does not exist", t.Spec.Broker.Name)
			// No need to requeue, we will be notified when if broker is created.
			return controller.NewPermanentError(err)
		}

		t.Status.MarkBrokerFailed(common.ReasonFailedBrokerGet, "Failed to get broker %q : %s", t.Spec.Broker, err)
		return pkgreconciler.NewEvent(corev1.EventTypeWarning, common.ReasonFailedBrokerGet,
			"Failed to get broker for trigger %s/%s: %w", t.Namespace, t.Name, err)
	}

	t.Status.PropagateBrokerCondition(rb.Status.GetTopLevelCondition())

	// No need to requeue, we'll get requeued when broker changes status.
	if !rb.IsReady() {
		logging.FromContext(ctx).Errorw(fmt.Sprintf("Trigger %s/%s references non ready broker %q", t.Namespace, t.Name, t.Spec.Broker.Name))
	}

	return nil
}

func (r *Reconciler) resolveMemoryBroker(ctx context.Context, t *eventingv1alpha1.Trigger) pkgreconciler.Event {
	mb, err := r.mbLister.MemoryBrokers(t.Namespace).Get(t.Spec.Broker.Name)
	if err != nil {
		if apierrs.IsNotFound(err) {
			logging.FromContext(ctx).Errorw(fmt.Sprintf("Trigger %s/%s references non existing broker %q", t.Namespace, t.Name, t.Spec.Broker.Name))
			t.Status.MarkBrokerFailed(common.ReasonBrokerDoesNotExist, "Broker %q does not exist", t.Spec.Broker.Name)
			// No need to requeue, we will be notified when if broker is created.
			return controller.NewPermanentError(err)
		}

		t.Status.MarkBrokerFailed(common.ReasonFailedBrokerGet, "Failed to get broker %q : %s", t.Spec.Broker, err)
		return pkgreconciler.NewEvent(corev1.EventTypeWarning, common.ReasonFailedBrokerGet,
			"Failed to get broker for trigger %s/%s: %w", t.Namespace, t.Name, err)
	}

	t.Status.PropagateBrokerCondition(mb.Status.GetTopLevelCondition())

	// No need to requeue, we'll get requeued when broker changes status.
	if !mb.IsReady() {
		logging.FromContext(ctx).Errorw(fmt.Sprintf("Trigger %s/%s references non ready broker %q", t.Namespace, t.Name, t.Spec.Broker.Name))
	}

	return nil
}

func (r *Reconciler) resolveTarget(ctx context.Context, t *eventingv1alpha1.Trigger) pkgreconciler.Event {
	if t.Spec.Target.Ref != nil && t.Spec.Target.Ref.Namespace == "" {
		// To call URIFromDestinationV1(ctx context.Context, dest v1.Destination, parent interface{}), dest.Ref must have a Namespace
		// If Target.Ref.Namespace is nil, We will use the Namespace of Trigger as the Namespace of dest.Ref
		t.Spec.Target.Ref.Namespace = t.Namespace
	}

	targetURI, err := r.uriResolver.URIFromDestinationV1(ctx, t.Spec.Target, t)
	if err != nil {
		logging.FromContext(ctx).Errorw("Unable to get the target's URI", zap.Error(err))
		t.Status.MarkTargetResolvedFailed("Unable to get the target's URI", "%v", err)
		t.Status.TargetURI = nil
		return pkgreconciler.NewEvent(corev1.EventTypeWarning, common.ReasonFailedResolveReference,
			"Failed to get target's URI: %w", err)
	}

	t.Status.TargetURI = targetURI
	t.Status.MarkTargetResolvedSucceeded()

	return nil
}

func (r *Reconciler) resolveDLS(ctx context.Context, t *eventingv1alpha1.Trigger) pkgreconciler.Event {
	if t.Spec.Delivery == nil || t.Spec.Delivery.DeadLetterSink == nil {
		t.Status.DeadLetterSinkURI = nil
		t.Status.MarkDeadLetterSinkNotConfigured()
		return nil
	}

	if t.Spec.Delivery.DeadLetterSink.Ref != nil && t.Spec.Delivery.DeadLetterSink.Ref.Namespace == "" {
		// To call URIFromDestinationV1(ctx context.Context, dest v1.Destination, parent interface{}), dest.Ref must have a Namespace
		// If Target.Ref.Namespace is nil, We will use the Namespace of Trigger as the Namespace of dest.Ref
		t.Spec.Delivery.DeadLetterSink.Ref.Namespace = t.Namespace
	}

	dlsURI, err := r.uriResolver.URIFromDestinationV1(ctx, *t.Spec.Delivery.DeadLetterSink, t)
	if err != nil {
		logging.FromContext(ctx).Errorw("Unable to get the dead letter sink's URI", zap.Error(err))
		t.Status.MarkDeadLetterSinkResolvedFailed("Unable to get the dead letter sink's URI", "%v", err)
		t.Status.TargetURI = nil
		return pkgreconciler.NewEvent(corev1.EventTypeWarning, common.ReasonFailedResolveReference,
			"Failed to get dead letter sink's URI: %w", err)
	}

	t.Status.DeadLetterSinkURI = dlsURI
	t.Status.MarkDeadLetterSinkResolvedSucceeded()

	return nil
}
