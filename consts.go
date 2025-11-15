package ctrlfwk

const (
	FinalizerDependenciesManagedBy = "dependencies.ctrlfwk.com/cleanup-dependencies-managed-by"

	// LabelReconciliationPaused can be added to a resource to pause its reconciliation
	// when using resources that support pausing.
	// It can also be added to CRs to pause the whole reconciliation if the NotPausedPredicate is used.
	// You can set the value to anything, so you can use it to document who/what paused the reconciliation.
	LabelReconciliationPaused = "ctrlfwk.com/pause"
)
