package library

const (
	ConditionTypeReady = "Ready"
)

const (
	ReasonReconciling = "Reconciling"
	ReasonReconciled  = "Reconciled"
	ReasonFinalizing  = "Finalizing"
	ReasonUnknown     = "Unknown"
	ReasonNotFound    = "NotFound"
)

const (
	StepFindControllerResource = "FindControllerResource"
	StepResolveDependency      = "ResolveDependency%s"
	StepResolveDependencies    = "ResolveDependencies"
	StepReconcileChild         = "ReconcileChild%s"
	StepReconcileChildren      = "ReconcileChildren"
	StepEndReconciliation      = "EndReconciliation"
)
