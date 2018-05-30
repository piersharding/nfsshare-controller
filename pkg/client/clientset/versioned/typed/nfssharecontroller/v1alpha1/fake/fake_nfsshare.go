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

package fake

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	v1alpha1 "github.com/piersharding/nfsshare-controller/pkg/apis/nfssharecontroller/v1alpha1"
)

// FakeNfsshares implements NfsshareInterface
type FakeNfsshares struct {
	Fake *FakeNfssharecontrollerV1alpha1
	ns   string
}

var foosResource = schema.GroupVersionResource{Group: "nfssharecontroller.k8s.io", Version: "v1alpha1", Resource: "foos"}

var foosKind = schema.GroupVersionKind{Group: "nfssharecontroller.k8s.io", Version: "v1alpha1", Kind: "Nfsshare"}

// Get takes name of the foo, and returns the corresponding foo object, and an error if there is any.
func (c *FakeNfsshares) Get(name string, options v1.GetOptions) (result *v1alpha1.Nfsshare, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(foosResource, c.ns, name), &v1alpha1.Nfsshare{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Nfsshare), err
}

// List takes label and field selectors, and returns the list of Nfsshares that match those selectors.
func (c *FakeNfsshares) List(opts v1.ListOptions) (result *v1alpha1.NfsshareList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(foosResource, foosKind, c.ns, opts), &v1alpha1.NfsshareList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.NfsshareList{ListMeta: obj.(*v1alpha1.NfsshareList).ListMeta}
	for _, item := range obj.(*v1alpha1.NfsshareList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested foos.
func (c *FakeNfsshares) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(foosResource, c.ns, opts))

}

// Create takes the representation of a foo and creates it.  Returns the server's representation of the foo, and an error, if there is any.
func (c *FakeNfsshares) Create(foo *v1alpha1.Nfsshare) (result *v1alpha1.Nfsshare, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(foosResource, c.ns, foo), &v1alpha1.Nfsshare{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Nfsshare), err
}

// Update takes the representation of a foo and updates it. Returns the server's representation of the foo, and an error, if there is any.
func (c *FakeNfsshares) Update(foo *v1alpha1.Nfsshare) (result *v1alpha1.Nfsshare, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(foosResource, c.ns, foo), &v1alpha1.Nfsshare{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Nfsshare), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeNfsshares) UpdateStatus(foo *v1alpha1.Nfsshare) (*v1alpha1.Nfsshare, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(foosResource, "status", c.ns, foo), &v1alpha1.Nfsshare{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Nfsshare), err
}

// Delete takes name of the foo and deletes it. Returns an error if one occurs.
func (c *FakeNfsshares) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(foosResource, c.ns, name), &v1alpha1.Nfsshare{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeNfsshares) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(foosResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.NfsshareList{})
	return err
}

// Patch applies the patch and returns the patched foo.
func (c *FakeNfsshares) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Nfsshare, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(foosResource, c.ns, name, data, subresources...), &v1alpha1.Nfsshare{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Nfsshare), err
}
