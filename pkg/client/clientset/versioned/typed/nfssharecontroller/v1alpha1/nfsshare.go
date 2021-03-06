/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	v1alpha1 "github.com/piersharding/nfsshare-controller/pkg/apis/nfssharecontroller/v1alpha1"
	scheme "github.com/piersharding/nfsshare-controller/pkg/client/clientset/versioned/scheme"
)

// NfssharesGetter has a method to return a NfsshareInterface.
// A group's client should implement this interface.
type NfssharesGetter interface {
	Nfsshares(namespace string) NfsshareInterface
}

// NfsshareInterface has methods to work with Nfsshare resources.
type NfsshareInterface interface {
	Create(*v1alpha1.Nfsshare) (*v1alpha1.Nfsshare, error)
	Update(*v1alpha1.Nfsshare) (*v1alpha1.Nfsshare, error)
	UpdateStatus(*v1alpha1.Nfsshare) (*v1alpha1.Nfsshare, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.Nfsshare, error)
	List(opts v1.ListOptions) (*v1alpha1.NfsshareList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Nfsshare, err error)
	NfsshareExpansion
}

// nfsshares implements NfsshareInterface
type nfsshares struct {
	client rest.Interface
	ns     string
}

// newNfsshares returns a Nfsshares
func newNfsshares(c *NfssharecontrollerV1alpha1Client, namespace string) *nfsshares {
	return &nfsshares{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the nfsshare, and returns the corresponding nfsshare object, and an error if there is any.
func (c *nfsshares) Get(name string, options v1.GetOptions) (result *v1alpha1.Nfsshare, err error) {
	result = &v1alpha1.Nfsshare{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("nfsshares").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Nfsshares that match those selectors.
func (c *nfsshares) List(opts v1.ListOptions) (result *v1alpha1.NfsshareList, err error) {
	result = &v1alpha1.NfsshareList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("nfsshares").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested nfsshares.
func (c *nfsshares) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("nfsshares").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a nfsshare and creates it.  Returns the server's representation of the nfsshare, and an error, if there is any.
func (c *nfsshares) Create(nfsshare *v1alpha1.Nfsshare) (result *v1alpha1.Nfsshare, err error) {
	result = &v1alpha1.Nfsshare{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("nfsshares").
		Body(nfsshare).
		Do().
		Into(result)
	return
}

// Update takes the representation of a nfsshare and updates it. Returns the server's representation of the nfsshare, and an error, if there is any.
func (c *nfsshares) Update(nfsshare *v1alpha1.Nfsshare) (result *v1alpha1.Nfsshare, err error) {
	result = &v1alpha1.Nfsshare{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("nfsshares").
		Name(nfsshare.Name).
		Body(nfsshare).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *nfsshares) UpdateStatus(nfsshare *v1alpha1.Nfsshare) (result *v1alpha1.Nfsshare, err error) {
	result = &v1alpha1.Nfsshare{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("nfsshares").
		Name(nfsshare.Name).
		SubResource("status").
		Body(nfsshare).
		Do().
		Into(result)
	return
}

// Delete takes name of the nfsshare and deletes it. Returns an error if one occurs.
func (c *nfsshares) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("nfsshares").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *nfsshares) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("nfsshares").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched nfsshare.
func (c *nfsshares) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Nfsshare, err error) {
	result = &v1alpha1.Nfsshare{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("nfsshares").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
