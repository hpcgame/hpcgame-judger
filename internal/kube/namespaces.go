package kube

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func (c *Client) DeleteNamespace(ctx context.Context, namespace string, gracePeriodSecs int64) error {
	return c.cs.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{
		GracePeriodSeconds: ptr.To(gracePeriodSecs),
	})
}
