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

// NewConfigMapChild creates a new ChildResource representing a ConfigMap
func NewConfigMapChild(reconciler ctrlfwk.Reconciler[*testv1.Test]) *ctrlfwk.ChildResource[*corev1.ConfigMap] {
	cr := reconciler.GetCustomResource()

	return ctrlfwk.NewChildResource(
		&corev1.ConfigMap{},

		ctrlfwk.WithChildShouldDelete(&corev1.ConfigMap{}, func() bool {
			return !cr.Spec.ConfigMap.Enabled
		}),

		ctrlfwk.WithChildKeyFunc(&corev1.ConfigMap{}, func() types.NamespacedName {
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

		ctrlfwk.WithChildMutator(func(child *corev1.ConfigMap) (err error) {
			child.Data = make(map[string]string)
			for k, v := range cr.Spec.ConfigMap.Data {
				child.Data[k] = v
			}

			return controllerutil.SetOwnerReference(cr, child, reconciler.Scheme())
		}),

		ctrlfwk.WithChildReadyCheck(func(_ *corev1.ConfigMap) bool { return true }),

		// Update the Status condition on ConfigMap creation
		ctrlfwk.WithChildOnReconcile(&corev1.ConfigMap{}, func(ctx context.Context) error {
			if cr.Status.ConfigMapStatus == nil {
				return nil
			}

			if !cr.Spec.ConfigMap.Enabled {
				changed := meta.RemoveStatusCondition(&cr.Status.Conditions, "ConfigMap")
				if changed || cr.Status.ConfigMapStatus != nil {
					cr.Status.ConfigMapStatus = nil
					return ctrlfwk.PatchCustomResourceStatus(ctx, reconciler)
				}
				return nil
			}

			if cr.Spec.ConfigMap.Enabled && cr.Status.ConfigMapStatus.Name != cr.Spec.ConfigMap.Name {
				oldCM := &corev1.ConfigMap{}
				oldCM.SetName(cr.Status.ConfigMapStatus.Name)
				oldCM.SetNamespace(cr.Namespace)
				if err := reconciler.Delete(ctx, oldCM); client.IgnoreNotFound(err) != nil {
					return err
				}
				cr.Status.ConfigMapStatus = nil
				return ctrlfwk.PatchCustomResourceStatus(ctx, reconciler)
			}

			return nil
		}),

		// Update the Status condition on ConfigMap creation
		ctrlfwk.WithChildOnCreate(func(ctx context.Context, _ *corev1.ConfigMap) error {
			cond := meta.FindStatusCondition(cr.Status.Conditions, "ConfigMap")
			if cond == nil {
				cond = &metav1.Condition{
					Type:               "ConfigMap",
					Status:             metav1.ConditionTrue,
					ObservedGeneration: cr.Generation,
					Reason:             "Created",
				}

				changed := meta.SetStatusCondition(&cr.Status.Conditions, *cond)
				if !changed {
					return nil
				}
			}

			cr.Status.ConfigMapStatus = &testv1.ConfigMapStatus{
				Name: cr.Spec.ConfigMap.Name,
			}

			return ctrlfwk.PatchCustomResourceStatus(ctx, reconciler)
		}),

		// Update the Status condition on ConfigMap update
		ctrlfwk.WithChildOnUpdate(func(ctx context.Context, _ *corev1.ConfigMap) error {
			cond := meta.FindStatusCondition(cr.Status.Conditions, "ConfigMap")
			if cond == nil {
				cond = &metav1.Condition{
					Type:   "ConfigMap",
					Status: metav1.ConditionTrue,
					Reason: "Updated",
				}

				changed := meta.SetStatusCondition(&cr.Status.Conditions, *cond)
				if !changed {
					return nil
				}
			}
			cond.Status = metav1.ConditionTrue
			cond.Reason = "Updated"
			cond.LastTransitionTime = metav1.Now()
			cond.ObservedGeneration = cr.Generation

			return ctrlfwk.PatchCustomResourceStatus(ctx, reconciler)
		}),
	)
}
