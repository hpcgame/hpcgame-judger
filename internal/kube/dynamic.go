package kube

import (
	"context"
	"errors"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

func trimBlank(str string) string {
	return strings.Trim(str, " \n\t\r")
}

func (c *Client) strToStrSlice(str string) []string {
	var rslt []string

	// Delim by yaml's --- rule
	for _, s := range strings.Split(str, "\n---") {
		if trimBlank(s) == "" {
			continue
		}

		rslt = append(rslt, s)
	}

	return rslt
}

func (c *Client) strToResource(str string) (
	schema.GroupVersionResource, *unstructured.Unstructured, error,
) {
	obj := unstructured.Unstructured{}

	_, gvk, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(str), nil, &obj)
	if err != nil {
		return schema.GroupVersionResource{}, nil, err
	}

	mapping, err := c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, nil, err
	}
	res := mapping.Resource

	return res, &obj, nil
}

func (c *Client) createItem(ctx context.Context, str string) error {
	res, obj, err := c.strToResource(str)
	if err != nil {
		return err
	}

	_, err = c.dc.Resource(res).Namespace(obj.GetNamespace()).Create(ctx, obj, metav1.CreateOptions{})
	return err
}

func (c *Client) Create(ctx context.Context, str string, continueOnFailure bool) error {
	errs := []error{}

	for _, s := range c.strToStrSlice(str) {
		err := c.createItem(ctx, s)
		if err != nil {
			errs = append(errs, err)
			if !continueOnFailure {
				break
			}
		}
	}

	return errors.Join(errs...)
}

func (c *Client) deleteItem(ctx context.Context, str string) error {
	res, obj, err := c.strToResource(str)
	if err != nil {
		return err
	}

	err = c.dc.Resource(res).Namespace(obj.GetNamespace()).Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Delete(ctx context.Context, str string, continueOnFailure bool) error {
	errs := []error{}

	for _, s := range c.strToStrSlice(str) {
		err := c.deleteItem(ctx, s)
		if err != nil {
			errs = append(errs, err)
			if !continueOnFailure {
				break
			}
		}
	}

	return errors.Join(errs...)
}

func (c *Client) updateItem(ctx context.Context, str string) error {
	res, obj, err := c.strToResource(str)
	if err != nil {
		return err
	}

	_, err = c.dc.Resource(res).Namespace(obj.GetNamespace()).Update(ctx, obj, metav1.UpdateOptions{})
	return err
}

func (c *Client) Update(ctx context.Context, str string, continueOnFailure bool) error {
	errs := []error{}

	for _, s := range c.strToStrSlice(str) {
		err := c.updateItem(ctx, s)
		if err != nil {
			errs = append(errs, err)
			if !continueOnFailure {
				break
			}
		}
	}

	return errors.Join(errs...)
}

func (c *Client) applyItem(ctx context.Context, str string) error {
	res, obj, err := c.strToResource(str)
	if err != nil {
		return err
	}

	_, err = c.dc.Resource(res).
		Namespace(obj.GetNamespace()).
		Apply(ctx, obj.GetName(), obj, metav1.ApplyOptions{})
	return err
}

func (c *Client) Apply(ctx context.Context, str string, continueOnFailure bool) error {
	errs := []error{}

	for _, s := range c.strToStrSlice(str) {
		err := c.applyItem(ctx, s)
		if err != nil {
			errs = append(errs, err)
			if !continueOnFailure {
				break
			}
		}
	}

	return errors.Join(errs...)
}
