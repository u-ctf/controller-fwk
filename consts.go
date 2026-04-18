package ctrlfwk

const (
	FinalizerDependenciesManagedBy = "dependencies.ctrlfwk.uctf.io/cleanup-dependencies-managed-by"

	// LabelReconciliationPaused can be added to a resource to pause its reconciliation
	// when using resources that support pausing.
	// It can also be added to CRs to pause the whole reconciliation if the NotPausedPredicate is used.
	// You can set the value to anything, so you can use it to document who/what paused the reconciliation.
	LabelReconciliationPaused = "ctrlfwk.uctf.io/pause"

	// LabelManagedBy is used to indicate which controller is managing a resource. It is used by the watch setup to determine if a resource should be watched as a dependency.
	LabelManagedBy = "app.kubernetes.io/managed-by"

	// LabelDependentWatchedBy is used to indicate which controller is watching a resource as a dependency. It is used by the watch setup to determine if a resource should be watched as a dependency.
	LabelDependentWatchedBy = "ctrlfwk.uctf.io/dependent-watched-by"
)
