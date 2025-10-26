package test_children

import (
	"context"

	ctrlfwk "github.com/u-ctf/controller-fwk"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	testv1 "operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewConfigMapResource creates a new Resource representing a ConfigMap
func NewConfigMapResource(reconciler ctrlfwk.Reconciler[*testv1.Test]) *ctrlfwk.Resource[*corev1.ConfigMap] {
	cr := reconciler.GetCustomResource()

	return ctrlfwk.NewResource(
		&corev1.ConfigMap{},

		ctrlfwk.ResourceSkipAndDeleteOnCondition(&corev1.ConfigMap{}, func() bool {
			return !cr.Spec.ConfigMap.Enabled
		}),

		ctrlfwk.ResourceWithKeyFunc(&corev1.ConfigMap{}, func() types.NamespacedName {
			if !cr.Spec.ConfigMap.Enabled && cr.Status.ConfigMapStatus != nil && cr.Status.ConfigMapStatus.Name != "" {
				// Use the name from status if the ConfigMap is disabled but still exists
				return types.NamespacedName{
					Name:      cr.Status.ConfigMapStatus.Name,
					Namespace: cr.Namespace,
				}
			}

			return types.NamespacedName{
				Name:      cr.Spec.ConfigMap.Name,
				Namespace: cr.Namespace,
			}
		}),

		ctrlfwk.ResourceWithMutator(func(resource *corev1.ConfigMap) (err error) {
			resource.Data = make(map[string]string)
			for k, v := range cr.Spec.ConfigMap.Data {
				resource.Data[k] = v
			}

			return controllerutil.SetOwnerReference(cr, resource, reconciler.Scheme())
		}),

		ctrlfwk.ResourceWithReadinessCondition(func(_ *corev1.ConfigMap) bool { return true }),

		// Update the Status condition on ConfigMap creation
		ctrlfwk.ResourceAfterReconcile(&corev1.ConfigMap{}, func(ctx context.Context, resource *corev1.ConfigMap) error {
			// This is the following state: The ConfigMap has been disabled
			if !cr.Spec.ConfigMap.Enabled {
				return CleanupStatusOnConfigMapDeletion(ctx, reconciler)
			}

			// This would happen on a change from disabled to enabled (or initial creation)
			if cr.Status.ConfigMapStatus == nil {
				cr.Status.ConfigMapStatus = &testv1.ConfigMapStatus{}
			}

			// This is the following state: The ConfigMap has been renamed
			if cr.Status.ConfigMapStatus.Name != cr.Spec.ConfigMap.Name {
				if err := CleanupConfigMapOnDeletion(ctx, reconciler); err != nil {
					return err
				}
				if err := CleanupStatusOnConfigMapDeletion(ctx, reconciler); err != nil {
					return err
				}
			}

			// This is the following state: The ConfigMap is up to date
			return SetStatusConfigMapIsUpToDate(ctx, reconciler)
		}),
	)
}

func CleanupStatusOnConfigMapDeletion(
	ctx context.Context,
	reconciler ctrlfwk.Reconciler[*testv1.Test],
) error {
	cr := reconciler.GetCustomResource()

	changed := meta.RemoveStatusCondition(&cr.Status.Conditions, "ConfigMap")
	if changed || cr.Status.ConfigMapStatus != nil {
		cr.Status.ConfigMapStatus = nil
		return ctrlfwk.PatchCustomResourceStatus(ctx, reconciler)
	}
	return nil
}

func CleanupConfigMapOnDeletion(
	ctx context.Context,
	reconciler ctrlfwk.Reconciler[*testv1.Test],
) error {
	cr := reconciler.GetCustomResource()

	if cr.Status.ConfigMapStatus != nil && cr.Status.ConfigMapStatus.Name != "" {
		cm := &corev1.ConfigMap{}
		cm.SetName(cr.Status.ConfigMapStatus.Name)
		cm.SetNamespace(cr.Namespace)
		if err := reconciler.Delete(ctx, cm); client.IgnoreNotFound(err) != nil {
			return err
		}
	}
	return nil
}

func SetStatusConfigMapIsUpToDate(
	ctx context.Context,
	reconciler ctrlfwk.Reconciler[*testv1.Test],
) error {
	cr := reconciler.GetCustomResource()

	cond := meta.FindStatusCondition(cr.Status.Conditions, "ConfigMap")
	if cond == nil {
		cond = &metav1.Condition{
			Type:               "ConfigMap",
			Status:             metav1.ConditionTrue,
			ObservedGeneration: cr.Generation,
			Reason:             "UpToDate",
		}
	}
	newCond := *cond

	newCond.Status = metav1.ConditionTrue
	newCond.Reason = "UpToDate"
	newCond.ObservedGeneration = cr.Generation
	cr.Status.ConfigMapStatus = &testv1.ConfigMapStatus{
		Name: cr.Spec.ConfigMap.Name,
	}

	changed := meta.SetStatusCondition(&cr.Status.Conditions, newCond)
	if changed {
		return ctrlfwk.PatchCustomResourceStatus(ctx, reconciler)
	}
	return nil
}
