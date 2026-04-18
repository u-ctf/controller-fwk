package ctrlfwk_test

import (
	"errors"
	"testing"

	ctrlfwk "github.com/u-ctf/controller-fwk"
	"github.com/u-ctf/controller-fwk/mocks"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type watchTestReconciler struct {
	client.Client
	ctrlfwk.WatchCache
}

func (watchTestReconciler) For(*corev1.ConfigMap) {}

func newWatchManager(t *testing.T, scheme *runtime.Scheme) ctrl.Manager {
	t.Helper()
	mgr, err := ctrl.NewManager(&rest.Config{Host: "https://127.0.0.1:65535"}, ctrl.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("failed to create watch test manager: %v", err)
	}
	return mgr
}

func newWatchTestReconciler(t *testing.T, objects ...client.Object) (*watchTestReconciler, *mocks.MockTypedController[reconcile.Request]) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add corev1 scheme: %v", err)
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
	mgr := newWatchManager(t, scheme)
	watchCache := ctrlfwk.NewWatchCache(mgr)
	controller := mocks.NewMockTypedController[reconcile.Request](gomock.NewController(t))
	watchCache.SetController(controller)
	return &watchTestReconciler{Client: client, WatchCache: watchCache}, controller
}

func TestSetupWatch_SkipsExistingWatchAndSupportsDependencySources(t *testing.T) {
	cr := &corev1.ConfigMap{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"}, ObjectMeta: metav1.ObjectMeta{Name: "controller", Namespace: "default"}}
	ctx := newResourceStepTestContext()
	ctx.SetCustomResource(cr)

	t.Run("already watching", func(t *testing.T) {
		reconciler, controller := newWatchTestReconciler(t, cr)
		watchKey := ctrlfwk.NewWatchKey(corev1.SchemeGroupVersion.WithKind("Secret"), ctrlfwk.CacheTypeEnqueueForOwner)
		reconciler.AddWatchSource(watchKey)
		controller.EXPECT().Watch(gomock.Any()).Times(0)

		result := ctrlfwk.SetupWatch(reconciler, &corev1.Secret{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"}}, false)(ctx, ctrl.Request{})
		if result.ShouldReturn() {
			t.Fatalf("expected existing watch to be skipped successfully, got %#v", result)
		}
	})

	t.Run("dependency watch", func(t *testing.T) {
		reconciler, controller := newWatchTestReconciler(t, cr)
		controller.EXPECT().Watch(gomock.Any()).Return(nil)

		result := ctrlfwk.SetupWatch(reconciler, &corev1.Secret{}, true)(ctx, ctrl.Request{})
		if result.ShouldReturn() {
			t.Fatalf("expected dependency watch setup to succeed, got %#v", result)
		}

		watchKey := ctrlfwk.NewWatchKey(corev1.SchemeGroupVersion.WithKind("Secret"), ctrlfwk.CacheTypeEnqueueForOwner)
		if !reconciler.IsWatchingSource(watchKey) {
			t.Fatal("expected dependency watch source to be cached")
		}
	})

	t.Run("controller watch error", func(t *testing.T) {
		reconciler, controller := newWatchTestReconciler(t, cr)
		controller.EXPECT().Watch(gomock.Any()).Return(errors.New("boom"))

		result := ctrlfwk.SetupWatch(reconciler, &corev1.Secret{}, false)(ctx, ctrl.Request{})
		if result.Error() == nil {
			t.Fatal("expected watch setup to return an error when controller watch fails")
		}
	})

	t.Run("resource version predicate", func(t *testing.T) {
		predicate := ctrlfwk.ResourceVersionChangedPredicate{}
		oldObj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "1"}}
		newObj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "2"}}

		if !predicate.Update(event.UpdateEvent{ObjectOld: oldObj, ObjectNew: newObj}) {
			t.Fatal("expected resource version changes to trigger update predicate")
		}
		if predicate.Create(event.CreateEvent{Object: oldObj}) {
			t.Fatal("expected create predicate to be false")
		}
		if !predicate.Delete(event.DeleteEvent{Object: oldObj}) || !predicate.Generic(event.GenericEvent{Object: oldObj}) {
			t.Fatal("expected delete and generic predicates to return true")
		}
	})
}
