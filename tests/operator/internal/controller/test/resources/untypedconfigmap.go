package test_resources

import (
	ctrlfwk "github.com/u-ctf/controller-fwk"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	testv1 "operator/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// NewUntypedConfigMapResource creates a new Resource representing a ConfigMap
func NewUntypedConfigMapResource(ctx testv1.UntypedTestContext, reconciler ctrlfwk.ReconcilerWithEventRecorder[*testv1.UntypedTest]) testv1.UntypedTestResource {
	cr := ctx.GetCustomResource()

	return ctrlfwk.NewUntypedResourceBuilder(ctx, schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"}).
		WithCanBePaused(true).
		WithSkipAndDeleteOnCondition(func() bool {
			return !cr.Spec.ConfigMap.Enabled
		}).
		WithKeyFunc(func() types.NamespacedName {
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
		}).
		WithMutator(func(resource *unstructured.Unstructured) (err error) {
			datas := make(map[string]any)
			for k, v := range cr.Spec.ConfigMap.Data {
				datas[k] = v
			}

			unstructured.SetNestedMap(resource.Object, datas, "data")

			return controllerutil.SetOwnerReference(cr, resource, reconciler.Scheme())
		}).
		WithReadinessCondition(func(_ *unstructured.Unstructured) bool { return true }).
		WithBeforeReconcile(func(ctx testv1.UntypedTestContext) error {
			// This is the following state: The ConfigMap has been disabled
			if !cr.Spec.ConfigMap.Enabled {
				if err := CleanupConfigMapOnDeletionOnUntypedTest(ctx, reconciler); err != nil {
					return err
				}
				if err := CleanupStatusOnConfigMapDeletionOnUntypedTest(ctx, reconciler); err != nil {
					return err
				}
				return nil
			}

			// This would happen on a change from disabled to enabled (or initial creation)
			if cr.Status.ConfigMapStatus == nil {
				cr.Status.ConfigMapStatus = &testv1.ConfigMapStatus{}
			}

			// This is the following state: The ConfigMap has been renamed
			if cr.Status.ConfigMapStatus.Name != "" && cr.Status.ConfigMapStatus.Name != cr.Spec.ConfigMap.Name {
				if err := CleanupConfigMapOnDeletionOnUntypedTest(ctx, reconciler); err != nil {
					return err
				}
			}
			return nil
		}).
		WithAfterReconcile(func(ctx testv1.UntypedTestContext, resource *unstructured.Unstructured) error {
			if !cr.Spec.ConfigMap.Enabled {
				return nil
			}

			// This is the following state: The ConfigMap is up to date
			return SetStatusConfigMapIsUpToDateOnUntypedTest(ctx, reconciler)
		}).
		WithAfterCreate(func(ctx testv1.UntypedTestContext, resource *unstructured.Unstructured) error {
			reconciler.Eventf(cr, "Normal", "ConfigMapCreated", "ConfigMap %s/%s created", resource.GetNamespace(), resource.GetName())
			return nil
		}).
		WithAfterDelete(func(ctx testv1.UntypedTestContext, resource *unstructured.Unstructured) error {
			reconciler.Eventf(cr, "Normal", "ConfigMapDeleted", "ConfigMap %s/%s deleted", resource.GetNamespace(), resource.GetName())
			return nil
		}).
		WithAfterUpdate(func(ctx testv1.UntypedTestContext, resource *unstructured.Unstructured) error {
			reconciler.Eventf(cr, "Normal", "ConfigMapUpdated", "ConfigMap %s/%s updated", resource.GetNamespace(), resource.GetName())
			return nil
		}).
		Build()
}

func CleanupStatusOnConfigMapDeletionOnUntypedTest(
	ctx testv1.UntypedTestContext,
	reconciler ctrlfwk.Reconciler[*testv1.UntypedTest],
) error {
	cr := ctx.GetCustomResource()

	changed := meta.RemoveStatusCondition(&cr.Status.Conditions, "ConfigMap")
	if changed || cr.Status.ConfigMapStatus != nil {
		cr.Status.ConfigMapStatus = nil
		return ctrlfwk.PatchCustomResourceStatus(ctx, reconciler)
	}
	return nil
}

func CleanupConfigMapOnDeletionOnUntypedTest(
	ctx testv1.UntypedTestContext,
	reconciler ctrlfwk.Reconciler[*testv1.UntypedTest],
) error {
	cr := ctx.GetCustomResource()

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

func SetStatusConfigMapIsUpToDateOnUntypedTest(
	ctx testv1.UntypedTestContext,
	reconciler ctrlfwk.Reconciler[*testv1.UntypedTest],
) error {
	cr := ctx.GetCustomResource()

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
