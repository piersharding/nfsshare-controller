/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/diff"
	kubeinformers "k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	nfssharecontroller "github.com/piersharding/nfsshare-controller/pkg/apis/nfssharecontroller/v1alpha1"
	"github.com/piersharding/nfsshare-controller/pkg/client/clientset/versioned/fake"
	informers "github.com/piersharding/nfsshare-controller/pkg/client/informers/externalversions"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

type fixture struct {
	t *testing.T

	client     *fake.Clientset
	kubeclient *k8sfake.Clientset
	// Objects to put in the store.
	nfsshareLister   []*nfssharecontroller.Nfsshare
	deploymentLister []*apps.Deployment
	// Actions expected to happen on the client.
	kubeactions []core.Action
	actions     []core.Action
	// Objects from here preloaded into NewSimpleFake.
	kubeobjects []runtime.Object
	objects     []runtime.Object
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.objects = []runtime.Object{}
	f.kubeobjects = []runtime.Object{}
	return f
}

func newNfsshare(name string, replicas *int32) *nfssharecontroller.Nfsshare {
	return &nfssharecontroller.Nfsshare{
		TypeMeta: metav1.TypeMeta{APIVersion: nfssharecontroller.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: nfssharecontroller.NfsshareSpec{
			DeploymentName: fmt.Sprintf("%s-deployment", name),
			Replicas:       replicas,
		},
	}
}

func (f *fixture) newController() (*Controller, informers.SharedInformerFactory, kubeinformers.SharedInformerFactory) {
	f.client = fake.NewSimpleClientset(f.objects...)
	f.kubeclient = k8sfake.NewSimpleClientset(f.kubeobjects...)

	i := informers.NewSharedInformerFactory(f.client, noResyncPeriodFunc())
	k8sI := kubeinformers.NewSharedInformerFactory(f.kubeclient, noResyncPeriodFunc())

	c := NewController(f.kubeclient, f.client,
		k8sI.Apps().V1().Deployments(), i.Nfssharecontroller().V1alpha1().Nfsshares())

	c.nfssharesSynced = alwaysReady
	c.deploymentsSynced = alwaysReady
	c.recorder = &record.FakeRecorder{}

	for _, f := range f.nfsshareLister {
		i.Nfssharecontroller().V1alpha1().Nfsshares().Informer().GetIndexer().Add(f)
	}

	for _, d := range f.deploymentLister {
		k8sI.Apps().V1().Deployments().Informer().GetIndexer().Add(d)
	}

	return c, i, k8sI
}

func (f *fixture) run(nfsshareName string) {
	f.runController(nfsshareName, true, false)
}

func (f *fixture) runExpectError(nfsshareName string) {
	f.runController(nfsshareName, true, true)
}

func (f *fixture) runController(nfsshareName string, startInformers bool, expectError bool) {
	c, i, k8sI := f.newController()
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		i.Start(stopCh)
		k8sI.Start(stopCh)
	}

	err := c.syncHandler(nfsshareName)
	if !expectError && err != nil {
		f.t.Errorf("error syncing nfsshare: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing nfsshare, got nil")
	}

	actions := filterInformerActions(f.client.Actions())
	for i, action := range actions {
		if len(f.actions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(actions)-len(f.actions), actions[i:])
			break
		}

		expectedAction := f.actions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.actions) > len(actions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.actions)-len(actions), f.actions[len(actions):])
	}

	k8sActions := filterInformerActions(f.kubeclient.Actions())
	for i, action := range k8sActions {
		if len(f.kubeactions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(k8sActions)-len(f.kubeactions), k8sActions[i:])
			break
		}

		expectedAction := f.kubeactions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.kubeactions) > len(k8sActions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.kubeactions)-len(k8sActions), f.kubeactions[len(k8sActions):])
	}
}

// checkAction verifies that expected and actual actions are equal and both have
// same attached resources
func checkAction(expected, actual core.Action, t *testing.T) {
	if !(expected.Matches(actual.GetVerb(), actual.GetResource().Resource) && actual.GetSubresource() == expected.GetSubresource()) {
		t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expected, actual)
		return
	}

	if reflect.TypeOf(actual) != reflect.TypeOf(expected) {
		t.Errorf("Action has wrong type. Expected: %t. Got: %t", expected, actual)
		return
	}

	switch a := actual.(type) {
	case core.CreateAction:
		e, _ := expected.(core.CreateAction)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintDiff(expObject, object))
		}
	case core.UpdateAction:
		e, _ := expected.(core.UpdateAction)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintDiff(expObject, object))
		}
	case core.PatchAction:
		e, _ := expected.(core.PatchAction)
		expPatch := e.GetPatch()
		patch := a.GetPatch()

		if !reflect.DeepEqual(expPatch, expPatch) {
			t.Errorf("Action %s %s has wrong patch\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintDiff(expPatch, patch))
		}
	}
}

// filterInformerActions filters list and watch actions for testing resources.
// Since list and watch don't change resource state we can filter it to lower
// nose level in our tests.
func filterInformerActions(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "nfsshares") ||
				action.Matches("watch", "nfsshares") ||
				action.Matches("list", "deployments") ||
				action.Matches("watch", "deployments")) {
			continue
		}
		ret = append(ret, action)
	}

	return ret
}

func (f *fixture) expectCreateDeploymentAction(d *apps.Deployment) {
	f.kubeactions = append(f.kubeactions, core.NewCreateAction(schema.GroupVersionResource{Resource: "deployments"}, d.Namespace, d))
}

func (f *fixture) expectUpdateDeploymentAction(d *apps.Deployment) {
	f.kubeactions = append(f.kubeactions, core.NewUpdateAction(schema.GroupVersionResource{Resource: "deployments"}, d.Namespace, d))
}

func (f *fixture) expectUpdateNfsshareStatusAction(nfsshare *nfssharecontroller.Nfsshare) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "nfsshares"}, nfsshare.Namespace, nfsshare)
	// TODO: Until #38113 is merged, we can't use Subresource
	//action.Subresource = "status"
	f.actions = append(f.actions, action)
}

func getKey(nfsshare *nfssharecontroller.Nfsshare, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(nfsshare)
	if err != nil {
		t.Errorf("Unexpected error getting key for nfsshare %v: %v", nfsshare.Name, err)
		return ""
	}
	return key
}

func TestCreatesDeployment(t *testing.T) {
	f := newFixture(t)
	nfsshare := newNfsshare("test", int32Ptr(1))

	f.nfsshareLister = append(f.nfsshareLister, nfsshare)
	f.objects = append(f.objects, nfsshare)

	expDeployment := newDeployment(nfsshare)
	f.expectCreateDeploymentAction(expDeployment)
	f.expectUpdateNfsshareStatusAction(nfsshare)

	f.run(getKey(nfsshare, t))
}

func TestDoNothing(t *testing.T) {
	f := newFixture(t)
	nfsshare := newNfsshare("test", int32Ptr(1))
	d := newDeployment(nfsshare)

	f.nfsshareLister = append(f.nfsshareLister, nfsshare)
	f.objects = append(f.objects, nfsshare)
	f.deploymentLister = append(f.deploymentLister, d)
	f.kubeobjects = append(f.kubeobjects, d)

	f.expectUpdateNfsshareStatusAction(nfsshare)
	f.run(getKey(nfsshare, t))
}

func TestUpdateDeployment(t *testing.T) {
	f := newFixture(t)
	nfsshare := newNfsshare("test", int32Ptr(1))
	d := newDeployment(nfsshare)

	// Update replicas
	nfsshare.Spec.Replicas = int32Ptr(2)
	expDeployment := newDeployment(nfsshare)

	f.nfsshareLister = append(f.nfsshareLister, nfsshare)
	f.objects = append(f.objects, nfsshare)
	f.deploymentLister = append(f.deploymentLister, d)
	f.kubeobjects = append(f.kubeobjects, d)

	f.expectUpdateNfsshareStatusAction(nfsshare)
	f.expectUpdateDeploymentAction(expDeployment)
	f.run(getKey(nfsshare, t))
}

func TestNotControlledByUs(t *testing.T) {
	f := newFixture(t)
	nfsshare := newNfsshare("test", int32Ptr(1))
	d := newDeployment(nfsshare)

	d.ObjectMeta.OwnerReferences = []metav1.OwnerReference{}

	f.nfsshareLister = append(f.nfsshareLister, nfsshare)
	f.objects = append(f.objects, nfsshare)
	f.deploymentLister = append(f.deploymentLister, d)
	f.kubeobjects = append(f.kubeobjects, d)

	f.runExpectError(getKey(nfsshare, t))
}

func int32Ptr(i int32) *int32 { return &i }
