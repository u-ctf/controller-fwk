package v1

import ctrlfwk "github.com/u-ctf/controller-fwk"

/////
// Test Controller
/////

// TestContext is the context type used for Test controllers
// This is an alias to simplify usage in other packages
type TestContext = *ctrlfwk.ContextWithData[*Test, int]

// TestDependency is the dependency type used for Test controllers
// This is an alias to simplify usage in other packages
type TestDependency = ctrlfwk.GenericDependency[*Test, TestContext]

// TestResource is the resource type used for Test controllers
// This is an alias to simplify usage in other packages
type TestResource = ctrlfwk.GenericResource[*Test, TestContext]

/////
// UntypedTest Controller
/////

// UntypedTestContext is the context type used for UntypedTest controllers
// This is an alias to simplify usage in other packages
type UntypedTestContext = ctrlfwk.Context[*UntypedTest]

// UntypedTestDependency is the dependency type used for UntypedTest controllers
// This is an alias to simplify usage in other packages
type UntypedTestDependency = ctrlfwk.GenericDependency[*UntypedTest, UntypedTestContext]

// UntypedTestResource is the resource type used for UntypedTest controllers
// This is an alias to simplify usage in other packages
type UntypedTestResource = ctrlfwk.GenericResource[*UntypedTest, UntypedTestContext]
