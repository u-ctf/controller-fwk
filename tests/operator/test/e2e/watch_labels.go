//go:build e2e
// +build e2e

package e2e

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"operator/internal/testlabels"
)

type labeledClient struct {
	client.Client
}

func newLabeledClient(delegate client.Client) client.Client {
	return &labeledClient{Client: delegate}
}

func (c *labeledClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	testlabels.ApplyToObject(obj)
	return c.Client.Create(ctx, obj, opts...)
}

func (c *labeledClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	testlabels.ApplyToObject(obj)
	return c.Client.Update(ctx, obj, opts...)
}
