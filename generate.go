package ctrlfwk

//go:generate go tool mockgen -destination=./mocks/mock_instrumenter.go -package mocks github.com/yyewolf/controller-fwk/instrument Instrumenter
//go:generate go tool mockgen -destination=./mocks/mock_typedcontroller.go -package mocks sigs.k8s.io/controller-runtime/pkg/controller TypedController
//go:generate go tool mockgen -destination=./mocks/mock_typedreconciler.go -package mocks sigs.k8s.io/controller-runtime/pkg/reconcile TypedReconciler
