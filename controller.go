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
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	// storageinformers "k8s.io/client-go/informers/storage/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/resource"

	// "github.com/operator-framework/operator-sdk/pkg/sdk"
	// v1beta1 "k8s.io/api/apps/v1beta1"
	exv1beta1 "k8s.io/api/extensions/v1beta1"
	// storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	// "github.com/piersharding/nfsshare-controller/kubeapi"
	// "github.com/piersharding/nfsshare-controller/pvc"

	nfssharev1alpha1 "github.com/piersharding/nfsshare-controller/pkg/apis/nfssharecontroller/v1alpha1"
	clientset "github.com/piersharding/nfsshare-controller/pkg/client/clientset/versioned"
	nfssharescheme "github.com/piersharding/nfsshare-controller/pkg/client/clientset/versioned/scheme"
	informers "github.com/piersharding/nfsshare-controller/pkg/client/informers/externalversions/nfssharecontroller/v1alpha1"
	listers "github.com/piersharding/nfsshare-controller/pkg/client/listers/nfssharecontroller/v1alpha1"
	// "github.com/sirupsen/logrus"
	// "k8s.io/apimachinery/pkg/api/resource"
	// "github.com/operator-framework/operator-sdk/pkg/sdk"
	// "k8s.io/api/apps/v1beta1"
	// storagev1 "k8s.io/api/storage/v1"
)

const controllerAgentName = "nfsshare-controller"

const (
	pvcOwnerKind       string = "Deployment"
	pvcOwnerAPIVersion string = "extensions/v1beta1"
)

const (
	// SuccessSynced is used as part of the Event 'reason' when a Nfsshare is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Nfsshare fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Nfsshare"
	// MessageResourceSynced is the message used for an Event fired when a Nfsshare
	// is synced successfully
	MessageResourceSynced = "Nfsshare synced successfully"
)

// Controller is the controller implementation for Nfsshare resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// nfsshareclientset is a clientset for our own API group
	nfsshareclientset clientset.Interface

	deploymentsLister appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced
	nfssharesLister   listers.NfsshareLister
	nfssharesSynced   cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new nfsshare controller
func NewController(
	kubeclientset kubernetes.Interface,
	nfsshareclientset clientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	nfsshareInformer informers.NfsshareInformer) *Controller {

	// Create event broadcaster
	// Add nfsshare-controller types to the default Kubernetes Scheme so Events can be
	// logged for nfsshare-controller types.
	nfssharescheme.AddToScheme(scheme.Scheme)
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:     kubeclientset,
		nfsshareclientset: nfsshareclientset,
		deploymentsLister: deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		nfssharesLister:   nfsshareInformer.Lister(),
		nfssharesSynced:   nfsshareInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Nfsshares"),
		recorder:          recorder,
	}

	glog.Info("Setting up event handlers")
	// Set up an event handler for when Nfsshare resources change
	nfsshareInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueNfsshare,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueNfsshare(new)
		},
	})
	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a Nfsshare resource will enqueue that Nfsshare resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1.Deployment)
			oldDepl := old.(*appsv1.Deployment)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	glog.Info("Starting Nfsshare controller")

	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.nfssharesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	glog.Info("Starting workers")
	// Launch two workers to process Nfsshare resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Nfsshare resource to be synced.
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		glog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Nfsshare resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Nfsshare resource with this namespace/name
	nfsshare, err := c.nfssharesLister.Nfsshares(namespace).Get(name)
	if err != nil {
		// The Nfsshare resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("nfsshare '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	shareName := nfsshare.Spec.ShareName
	if shareName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("%s: shareName name must be specified", key))
		return nil
	}

	// size := nfsshare.Spec.Size
	if nfsshare.Spec.Size == "" {
		nfsshare.Spec.Size = "2Gi"
	}
	// storageClass := nfsshare.Spec.StorageClass
	if nfsshare.Spec.StorageClass == "" {
		nfsshare.Spec.StorageClass = "standard"
	}
	// sharedDirectory := nfsshare.Spec.SharedDirectory
	if nfsshare.Spec.SharedDirectory == "" {
		nfsshare.Spec.SharedDirectory = "/nfsshare"
	}
	// image := nfsshare.Spec.Image
	if nfsshare.Spec.Image == "" {
		nfsshare.Spec.Image = "itsthenetwork/nfs-server-alpine:latest"
	}
	replicas := 1 // it should always be 1
	if nfsshare.Spec.Replicas == nil {
		replicas = 1
	}

	// see if deployment exists, svc, sc and pvc
	// Get the deployment with the name specified in Nfsshare.spec
	deployment, err := c.deploymentsLister.Deployments(nfsshare.Namespace).Get(shareName)

	// If the resource doesn't exist, we'll create it
	action := "update"
	if errors.IsNotFound(err) {
		action = "create"
	}
	glog.V(1).Infof("Handling Nfsshare[%s/%s] action: [%s] size: %s, StorageClass: %s, image: %s, replicas: %d",
		nfsshare.Namespace, shareName, action, nfsshare.Spec.Size, nfsshare.Spec.StorageClass, nfsshare.Spec.Image, replicas)

	// If the deployment doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		// first, do we have the storageClass
		_, checksvc, _ := CheckStorageClassExistence(c.kubeclientset, nfsshare.Spec.StorageClass)
		if !checksvc {
			glog.V(4).Infof("StorageClass [%s] for Nfs does not exist!", nfsshare.Spec.StorageClass)
			runtime.HandleError(fmt.Errorf("StorageClass [%s] for %s[%s] does not exist", nfsshare.Spec.StorageClass, shareName, key))
			return nil
		} else {
			glog.V(4).Infof("StorageClass [%s] exists - jolly good", nfsshare.Spec.StorageClass)
		}

		// now create the owner-dependent PVC - if it doesn't already exist
		pvcName := fmt.Sprintf("%s-data", shareName)
		_, checkpvc, _ := CheckPersistentVolumeClaimExistence(c.kubeclientset, nfsshare.Namespace, pvcName)
		if !checkpvc {
			glog.V(4).Infof("PersistentVolume claim [%s] for %s/%s[%s] does not exists!",
				pvcName, nfsshare.Namespace, shareName, key)
			_, checkpvc, _ := CreatePersistentVolumeClaim(c.kubeclientset, nfsshare.Namespace, pvcName, nfsshare.Spec.StorageClass, nfsshare.Spec.Size, deployment, nfsshare)
			if !checkpvc {
				glog.V(4).Infof("Failed to create PersistentVolume claim [%s] for %s/%s[%s] in StorageClass %s",
					pvcName, nfsshare.Namespace, shareName, nfsshare.Spec.StorageClass, key)
				runtime.HandleError(fmt.Errorf("Failed to create PersistentVolume [%s] for %s[%s] in StorageClass %s",
					pvcName, nfsshare.Namespace, shareName, nfsshare.Spec.StorageClass))
				return nil
			} else {
				glog.V(4).Infof("Created PersistentVolume claim [%s] for %s/%s[%s] StorageClass: %s Size: %s",
					pvcName, nfsshare.Namespace, shareName, key, nfsshare.Spec.StorageClass, nfsshare.Spec.Size)
				c.recorder.Event(nfsshare, corev1.EventTypeNormal, "CreatedPVC",
					fmt.Sprintf("Created PersistentVolume claim [%s] for %s/%s[%s] StorageClass: %s Size: %s",
						pvcName, nfsshare.Namespace, shareName, key,
						nfsshare.Spec.StorageClass, nfsshare.Spec.Size))

			}
		} else {
			glog.V(4).Infof("PersistentVolume claim [%s] for %s/%s[%s] already exists",
				pvcName, nfsshare.Namespace, shareName, key)
		}

		// create the deployment
		dout, _ := json.Marshal(newDeployment(nfsshare))
		glog.V(5).Infof("Deployment to create: %s", string(dout))

		deployment, err = c.kubeclientset.AppsV1().Deployments(nfsshare.Namespace).Create(newDeployment(nfsshare))

		if err != nil {
			glog.Errorf("error creating Deployment: %s/%s: "+err.Error(),
				nfsshare.Namespace, nfsshare.Name)
			runtime.HandleError(fmt.Errorf("Failed to create Deployment %s/%s [%s]",
				nfsshare.Namespace, shareName, key))
			return nil
		}
		c.recorder.Event(nfsshare, corev1.EventTypeNormal, "CreatedDeployment",
			fmt.Sprintf("Created Deployment %s/%s[%s]",
				nfsshare.Namespace, shareName, key))

		// Check for the service
		_, checksvc, _ = CheckServiceExistence(c.kubeclientset, shareName, nfsshare.Namespace)
		if !checksvc {
			glog.V(4).Infof("Nfs provider service does not exists %s/%s for %s",
				nfsshare.Namespace, shareName, key)
			svc, checksvc, _ := CreateService(c.kubeclientset, nfsshare.Namespace, shareName, deployment, nfsshare)
			if !checksvc {
				glog.V(4).Infof("Failed to create Service [%s] for %s/%s[%s]",
					shareName, nfsshare.Namespace, shareName, key)
				runtime.HandleError(fmt.Errorf("Failed to create Service [%s] for %s[%s]",
					shareName, nfsshare.Namespace, shareName))
				return nil
			} else {
				glog.V(4).Infof("Created Service [%s] for %s/%s[%s] ClusterIP: %s",
					shareName, nfsshare.Namespace, shareName, key, svc.Spec.ClusterIP)
				c.recorder.Event(nfsshare, corev1.EventTypeNormal, "CreatedService",
					fmt.Sprintf("Created Service [%s] for %s/%s[%s] ClusterIP: %s",
						shareName, nfsshare.Namespace, shareName, key, svc.Spec.ClusterIP))
			}
		} else {
			glog.V(4).Infof("Nfs provider service already exists %s/%s for %s",
				nfsshare.Namespace, shareName, key)
		}
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If the Deployment is not controlled by this Nfsshare resource, we should log
	// a warning to the event recorder and ret
	if !metav1.IsControlledBy(deployment, nfsshare) {
		msg := fmt.Sprintf(MessageResourceExists, deployment.Name)
		c.recorder.Event(nfsshare, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	// If this number of the replicas on the Nfsshare resource is specified, and the
	// number does not equal the current desired replicas on the Deployment, we
	// should update the Deployment resource.
	if nfsshare.Spec.Replicas != nil && *nfsshare.Spec.Replicas != *deployment.Spec.Replicas {
		glog.V(4).Infof("Nfsshare %s replicas: %d, deployment replicas: %d", name, *nfsshare.Spec.Replicas, *deployment.Spec.Replicas)
		deployment, err = c.kubeclientset.AppsV1().Deployments(nfsshare.Namespace).Update(newDeployment(nfsshare))
	}

	// If an error occurs during Update, we'll requeue the item so we can
	// attempt processing again later. THis could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// Finally, we update the status block of the Nfsshare resource to reflect the
	// current state of the world
	err = c.updateNfsshareStatus(nfsshare, deployment)
	if err != nil {
		return err
	}

	c.recorder.Event(nfsshare, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *Controller) updateNfsshareStatus(nfsshare *nfssharev1alpha1.Nfsshare, deployment *appsv1.Deployment) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	nfsshareCopy := nfsshare.DeepCopy()
	nfsshareCopy.Status.AvailableReplicas = deployment.Status.AvailableReplicas
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Nfsshare resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.nfsshareclientset.NfssharecontrollerV1alpha1().Nfsshares(nfsshare.Namespace).Update(nfsshareCopy)
	return err
}

// enqueueNfsshare takes a Nfsshare resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Nfsshare.
func (c *Controller) enqueueNfsshare(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the Nfsshare resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that Nfsshare resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		glog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	glog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a Nfsshare, we should not do anything more
		// with it.
		if ownerRef.Kind != "Nfsshare" {
			return
		}

		nfsshare, err := c.nfssharesLister.Nfsshares(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			glog.V(4).Infof("ignoring orphaned object '%s' of nfsshare '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueNfsshare(nfsshare)
		return
	}
}

func newTrue() *bool {
	b := true
	return &b
}

// newDeployment creates a new Deployment for a Nfsshare resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the Nfsshare resource that 'owns' it.
func newDeployment(nfsshare *nfssharev1alpha1.Nfsshare) *appsv1.Deployment {
	// docker run -d --name nfs --privileged -p 2049:2049 \
	// -v $(CURRENT_DIR)/:/arl \
	// -e SHARED_DIRECTORY=/arl itsthenetwork/nfs-server-alpine:latest
	// https://hub.docker.com/r/itsthenetwork/nfs-server-alpine/
	// privileged: true ?

	labels := map[string]string{
		"app":        nfsshare.Spec.ShareName,
		"controller": nfsshare.Name,
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       pvcOwnerKind,
			APIVersion: pvcOwnerAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nfsshare.Spec.ShareName,
			Namespace: nfsshare.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(nfsshare, schema.GroupVersionKind{
					Group:   nfssharev1alpha1.SchemeGroupVersion.Group,
					Version: nfssharev1alpha1.SchemeGroupVersion.Version,
					Kind:    "Nfsshare",
				}),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Replicas: nfsshare.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{Name: "nfs-prov-volume", VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: fmt.Sprintf("%s-data", nfsshare.Spec.ShareName),
							},
						}},
					},
					Containers: []corev1.Container{
						{
							Name:  "nfsshare",
							Image: nfsshare.Spec.Image,
							Ports: []corev1.ContainerPort{
								{Name: "nfs", ContainerPort: 2049, Protocol: "TCP"},
								{Name: "mountd", ContainerPort: 20048, Protocol: "TCP"},
								{Name: "rpcbind", ContainerPort: 111, Protocol: "TCP"},
								{Name: "rpcbind-udp", ContainerPort: 111, Protocol: "UDP"},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: newTrue(), // because *bool is stupid
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"DAC_READ_SEARCH",
										"SYS_RESOURCE",
									},
								},
							},
							// Args: []string{
							// 	"-provisioner=banzaicloud.com/nfs",
							// },
							Env: []corev1.EnvVar{
								{Name: "SHARED_DIRECTORY", Value: nfsshare.Spec.SharedDirectory},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "nfs-prov-volume", MountPath: nfsshare.Spec.SharedDirectory},
							},
						},
					},
				},
			},
		},
	}
}

// GetPVC gets a PVC by name
func CheckPersistentVolumeClaimExistence(clientset kubernetes.Interface, namespace string, name string) (*corev1.PersistentVolumeClaim, bool, error) {

	options := metav1.GetOptions{}
	pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(name, options)
	if kerrors.IsNotFound(err) {
		glog.V(4).Infof("PVC " + name + " is not found")
		return pvc, false, err
	}

	if err != nil {
		glog.Errorf("error getting pvc ns: " + namespace + " name: " + name + " - " + err.Error())
		return pvc, false, err
	}

	return pvc, true, err

}

// Create new PVC
func CreatePersistentVolumeClaim(clientset kubernetes.Interface, namespace string, name string, storageClass string, size string, deployment *appsv1.Deployment, nfsshare *nfssharev1alpha1.Nfsshare) (*corev1.PersistentVolumeClaim, bool, error) {
	glog.V(4).Infof("Creating new PersistentVolumeClaim for Nfs provisioner..")
	// owner, _, err := GetDeployment(clientset, deployment.Name, deployment.Namespace)
	// ownerRef := asOwner(owner)
	// plusStorage, _ := resource.ParseQuantity(size)
	// parsedStorageSize := pv.Spec.Resources.Requests["storage"]
	// parsedStorageSize.Add(plusStorage)
	parsedStorageSize, _ := resource.ParseQuantity(size)
	pvc := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(nfsshare, schema.GroupVersionKind{
					Group:   nfssharev1alpha1.SchemeGroupVersion.Group,
					Version: nfssharev1alpha1.SchemeGroupVersion.Version,
					Kind:    "Nfsshare",
				}),
				// OwnerReferences: []metav1.OwnerReference{
				// 	ownerRef,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"storage": parsedStorageSize,
				},
			},
		},
	}
	pvcOut, _ := json.Marshal(pvc)
	glog.V(5).Infof("PVC to create: %s", string(pvcOut))

	resultpvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Create(pvc)
	if err != nil && !errors.IsAlreadyExists(err) {
		glog.Errorf("Error happened during creating a PersistentVolumeClaim for Nfs %s", err.Error())
		return nil, false, err
	}
	if err != nil {
		glog.Error("error creating pvc " + err.Error() + " in namespace " + namespace)
		return nil, false, err
	}

	glog.V(5).Infof("created PVC " + resultpvc.Name)

	return resultpvc, true, err
}

// Create new Service
func CreateService(clientset kubernetes.Interface, namespace string, name string, deployment *appsv1.Deployment, nfsshare *nfssharev1alpha1.Nfsshare) (*corev1.Service, bool, error) {

	glog.V(4).Infof("Creating new Service for Nfs provisioner..")
	// owner, _, err := GetDeployment(clientset, deployment.Name, deployment.Namespace)
	// ownerRef := asOwner(owner)
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": name,
			},
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(nfsshare, schema.GroupVersionKind{
					Group:   nfssharev1alpha1.SchemeGroupVersion.Group,
					Version: nfssharev1alpha1.SchemeGroupVersion.Version,
					Kind:    "Nfsshare",
				}),
				// OwnerReferences: []metav1.OwnerReference{
				// 	ownerRef,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Name: "nfs", Port: 2049, Protocol: "TCP"},
				{Name: "mountd", Port: 20048, Protocol: "TCP"},
				{Name: "rpcbind", Port: 111, Protocol: "TCP"},
				{Name: "rpcbind-udp", Port: 111, Protocol: "UDP"},
			},
			Selector: map[string]string{
				"app": name,
			},
		},
	}
	svcresult, err := clientset.Core().Services(namespace).Create(svc)
	if err != nil && !errors.IsAlreadyExists(err) {
		glog.Errorf("Error happened during creating the Service for Nfs %s", err.Error())
		return nil, false, err
	}
	if err != nil {
		glog.Error(err)
		glog.Error("error creating service " + svc.Name)
		return nil, false, err
	}

	glog.V(5).Info("created service " + svcresult.Name)
	return svcresult, true, err
}

func CheckStorageClassExistence(clientset kubernetes.Interface, name string) (*storagev1beta1.StorageClass, bool, error) {

	options := metav1.GetOptions{}
	sc, err := clientset.StorageV1beta1().StorageClasses().Get(name, options)
	if kerrors.IsNotFound(err) {
		glog.V(4).Infof("SC " + name + " is not found")
		return sc, false, err
	}

	if err != nil {
		glog.Errorf("error getting sc name: " + name + " - " + err.Error())
		return sc, false, err
	}

	return sc, true, err
}

// CheckServiceExistence gets a Service by name
func CheckServiceExistence(clientset kubernetes.Interface, name, namespace string) (*corev1.Service, bool, error) {
	svc, err := clientset.CoreV1().Services(namespace).Get(name, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return svc, false, err
	}
	if err != nil {
		glog.Error(err)
		return svc, false, err
	}

	return svc, true, err
}

// asOwner returns an OwnerReference set as the memcached CR
func asOwner(m *exv1beta1.Deployment) metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: pvcOwnerAPIVersion,
		Kind:       pvcOwnerKind,
		Name:       m.Name,
		UID:        m.UID,
		Controller: &trueVar,
	}
}

// GetDeployment gets a deployment by name
func GetDeployment(clientset kubernetes.Interface, name, namespace string) (*exv1beta1.Deployment, bool, error) {
	deploymentResult, err := clientset.ExtensionsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		glog.V(5).Infof("deployment " + name + " not found")
		return deploymentResult, false, err
	}
	if err != nil {
		glog.Error(err)
		glog.Error("error getting Deployment " + name)
		return deploymentResult, false, err
	}

	out, _ := json.Marshal(deploymentResult)
	glog.V(5).Infof("Deployment: %s", string(out))

	return deploymentResult, true, err
}

// func SetUpNfsProvisioner(clientset *kubernetes.Clientset, pv *corev1.PersistentVolumeClaim, name string, namespace string) error {
// 	glog.V(4).Infof("Creating new PersistentVolumeClaim for Nfs provisioner..")

// 	deployment, _, err := GetDeployment(clientset, name, namespace)
// 	ownerRef := asOwner(deployment)
// 	plusStorage, _ := resource.ParseQuantity("2Gi")
// 	parsedStorageSize := pv.Spec.Resources.Requests["storage"]
// 	parsedStorageSize.Add(plusStorage)

// 	pvc := &corev1.PersistentVolumeClaim{
// 		TypeMeta: metav1.TypeMeta{
// 			Kind:       "PersistentVolumeClaim",
// 			APIVersion: "v1",
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      fmt.Sprintf("%s-data", name),
// 			Namespace: namespace,
// 			OwnerReferences: []metav1.OwnerReference{
// 				ownerRef,
// 			},
// 		},
// 		Spec: corev1.PersistentVolumeClaimSpec{
// 			AccessModes: []corev1.PersistentVolumeAccessMode{
// 				corev1.ReadWriteOnce,
// 			},
// 			Resources: corev1.ResourceRequirements{
// 				Requests: corev1.ResourceList{
// 					"storage": parsedStorageSize,
// 				},
// 			},
// 		},
// 	}
// 	result, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Create(pvc)
// 	if err != nil && !errors.IsAlreadyExists(err) {
// 		glog.Errorf("Error happened during creating a PersistentVolumeClaim for Nfs %s", err.Error())
// 		return err
// 	}
// 	if err != nil {
// 		glog.Error("error creating pvc " + err.Error() + " in namespace " + namespace)
// 		return err
// 	}

// 	glog.V(5).Infof("created PVC " + result.Name)

// 	glog.V(4).Infof("Creating new Service for Nfs provisioner..")
// 	svc := &corev1.Service{
// 		TypeMeta: metav1.TypeMeta{
// 			Kind:       "Service",
// 			APIVersion: "v1",
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: "nfs-provisioner",
// 			Labels: map[string]string{
// 				"app": name,
// 			},
// 			Namespace: namespace,
// 			OwnerReferences: []metav1.OwnerReference{
// 				ownerRef,
// 			},
// 		},
// 		Spec: corev1.ServiceSpec{
// 			Ports: []corev1.ServicePort{
// 				{Name: "nfs", Port: 2049},
// 				{Name: "mountd", Port: 20048},
// 				{Name: "rpcbind", Port: 111},
// 				{Name: "rpcbind-udp", Port: 111, Protocol: "UDP"},
// 			},
// 			Selector: map[string]string{
// 				"app": "nfs-provisioner",
// 			},
// 		},
// 	}
// 	svcresult, err := clientset.Core().Services(namespace).Create(svc)
// 	if err != nil && !errors.IsAlreadyExists(err) {
// 		glog.Errorf("Error happened during creating the Service for Nfs %s", err.Error())
// 		return err
// 	}
// 	if err != nil {
// 		glog.Error(err)
// 		glog.Error("error creating service " + svc.Name)
// 		return err
// 	}

// 	glog.V(4).Info("created service " + svcresult.Name)
// 	// return svcresult, err

// 	glog.V(4).Infof("Creating new Deployment for Nfs provisioner..")
// 	replicas := int32(1)
// 	deployment = &exv1beta1.Deployment{
// 		TypeMeta: metav1.TypeMeta{
// 			Kind:       "Deployment",
// 			APIVersion: "extensions/v1beta1",
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      name,
// 			Namespace: namespace,
// 			OwnerReferences: []metav1.OwnerReference{
// 				ownerRef,
// 			},
// 		},
// 		Spec: exv1beta1.DeploymentSpec{
// 			Replicas: &replicas,
// 			Template: corev1.PodTemplateSpec{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Labels: map[string]string{
// 						"app": name,
// 					},
// 				},
// 				Spec: corev1.PodSpec{
// 					Volumes: []corev1.Volume{
// 						{Name: "nfs-prov-volume", VolumeSource: corev1.VolumeSource{
// 							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
// 								ClaimName: fmt.Sprintf("%s-data", *pv.Spec.StorageClassName),
// 							},
// 						}},
// 					},
// 					Containers: []corev1.Container{
// 						{
// 							Name:  "nfs-provisioner",
// 							Image: "quay.io/kubernetes_incubator/nfs-provisioner:v1.0.9",
// 							Ports: []corev1.ContainerPort{
// 								{Name: "nfs", ContainerPort: 2049},
// 								{Name: "mountd", ContainerPort: 20048},
// 								{Name: "rpcbind", ContainerPort: 111},
// 								{Name: "rpcbind-udp", ContainerPort: 111, Protocol: "UDP"},
// 							},
// 							SecurityContext: &corev1.SecurityContext{
// 								Capabilities: &corev1.Capabilities{
// 									Add: []corev1.Capability{
// 										"DAC_READ_SEARCH",
// 										"SYS_RESOURCE",
// 									},
// 								},
// 							},
// 							Args: []string{
// 								"-provisioner=banzaicloud.com/nfs",
// 							},
// 							Env: []corev1.EnvVar{
// 								{Name: "POD_IP", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"}}},
// 								{Name: "SERVICE_NAME", Value: "nfs-provisioner"},
// 								{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
// 							},
// 							VolumeMounts: []corev1.VolumeMount{
// 								{Name: "nfs-prov-volume", MountPath: "/export"},
// 							},
// 						},
// 					},
// 				},
// 			},
// 			Strategy: exv1beta1.DeploymentStrategy{
// 				Type: exv1beta1.RecreateDeploymentStrategyType,
// 			},
// 		},
// 	}
// 	deploymentResult, err := clientset.ExtensionsV1beta1().Deployments(namespace).Create(deployment)
// 	if err != nil && !errors.IsAlreadyExists(err) {
// 		glog.Errorf("Error happened during creating the Deployment for Nfs %s", err.Error())
// 		return err
// 	}
// 	if err != nil {
// 		glog.Error("error creating Deployment " + err.Error())
// 		return err
// 	}

// 	glog.V(4).Info("created deployment " + deploymentResult.Name)
// 	glog.V(4).Infof("Creating new StorageClass for Nfs provisioner..")
// 	reclaimPolicy := corev1.PersistentVolumeReclaimRetain
// 	sc := &storagev1beta1.StorageClass{
// 		TypeMeta: metav1.TypeMeta{
// 			Kind:       "StorageClass",
// 			APIVersion: "storage.k8s.io/v1",
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: *pv.Spec.StorageClassName,
// 			OwnerReferences: []metav1.OwnerReference{
// 				ownerRef,
// 			},
// 		},
// 		ReclaimPolicy: &reclaimPolicy,
// 		Provisioner:   "banzaicloud.com/nfs",
// 	}
// 	_, err = clientset.StorageV1beta1().StorageClasses().Create(sc)
// 	if err != nil && !errors.IsAlreadyExists(err) {
// 		glog.Errorf("Error happened during creating the StorageClass for Nfs %s", err.Error())
// 		return err
// 	}
// 	if err != nil {
// 		glog.Errorf("error creating StorageClass: " + err.Error())
// 		return err
// 	}
// 	return nil
// }

func CheckNfsServerExistence(clientset *kubernetes.Clientset, name string, namespace string) bool {
	_, check, _ := CheckPersistentVolumeClaimExistence(clientset, namespace, fmt.Sprintf("%s-data", name))
	if !check {
		glog.V(4).Infof("PersistentVolume claim for Nfs does not exists!")
		return false
	}
	if !checkNfsProviderDeployment(clientset, name, namespace) {
		glog.V(4).Infof("Nfs provider deployment does not exists!")
		return false
	}
	_, check, _ = CheckStorageClassExistence(clientset, name)
	if !check {
		glog.V(4).Infof("StorageClass for Nfs does not exist!")
		return false
	}
	return true

}

func checkNfsProviderDeployment(clientset *kubernetes.Clientset, name string, namespace string) bool {
	// deployment := &exv1beta1.Deployment{
	// 	TypeMeta: metav1.TypeMeta{
	// 		Kind:       "Deployment",
	// 		APIVersion: "extensions/v1beta1",
	// 	},
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      name,
	// 		Namespace: namespace,
	// 	},
	// 	Spec: exv1beta1.DeploymentSpec{
	// 		Template: corev1.PodTemplateSpec{
	// 			Spec: corev1.PodSpec{
	// 				Containers: []corev1.Container{
	// 					{
	// 						Args: []string{
	// 							"-provisioner=banzaicloud.com/nfs",
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 	},
	// }
	// service := &corev1.Service{
	// 	TypeMeta: metav1.TypeMeta{
	// 		Kind:       "Service",
	// 		APIVersion: "v1",
	// 	},
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      name,
	// 		Namespace: namespace,
	// 	},
	// 	Spec: corev1.ServiceSpec{
	// 		Selector: map[string]string{"app": "nfs-provisioner"},
	// 	},
	// }
	_, _, err := GetDeployment(clientset, name, namespace)

	if err != nil {
		logrus.Infof("Nfs provider deployment does not exists %s", err.Error())
		return false
	}
	_, _, err = CheckServiceExistence(clientset, name, namespace)
	if err != nil {
		logrus.Infof("Nfs provider service does not exists %s", err.Error())
		return false
	}
	logrus.Info("Nfs provider exists!")
	return true
}

// func createDeployment(foo *postgresv1.Postgres, c *Controller) (string, string, []string, string) {

//         deploymentsClient := c.kubeclientset.AppsV1().Deployments(apiv1.NamespaceDefault)

//     deploymentName := foo.Spec.ShareName
//     image := foo.Spec.Image
//     username := foo.Spec.Username
//     password := foo.Spec.Password
//     database := foo.Spec.Database
//     setupCommands := canonicalize(foo.Spec.Commands)

//     fmt.Printf("   Deployment:%v, Image:%v, User:%v\n", deploymentName, image, username)
//     fmt.Printf("   Password:%v, Database:%v\n", password, database)
//     fmt.Printf("   SetupCmds:%v\n", setupCommands)

//         deployment := &appsv1.Deployment{
//                 ObjectMeta: metav1.ObjectMeta{
//                         Name: deploymentName,
//                 },
//                 Spec: appsv1.DeploymentSpec{
//                         Replicas: int32Ptr(1),
//                         Selector: &metav1.LabelSelector{
//                                 MatchLabels: map[string]string{
//                                              "app": deploymentName,
//                                 },
//                         },
//                         Template: apiv1.PodTemplateSpec{
//                                 ObjectMeta: metav1.ObjectMeta{
//                                         Labels: map[string]string{
//                                                 "app": deploymentName,
//                                         },
//                                 },

//                                 Spec: apiv1.PodSpec{
//                                         Containers: []apiv1.Container{
//                                                 {
//                                                         Name:  deploymentName,
//                                                         Image: image,
//                             Ports: []apiv1.ContainerPort{
//                                   {
//                                 ContainerPort: 5432,
//                                   },
//                             },
//                             ReadinessProbe: &apiv1.Probe{
//                                 Handler: apiv1.Handler{
//                                    TCPSocket: &apiv1.TCPSocketAction{
//                                       Port: apiutil.FromInt(5432),
//                                    },
//                                 },
//                                 InitialDelaySeconds: 5,
//                                 TimeoutSeconds: 60,
//                                 PeriodSeconds: 2,
//                             },
//                                                         Env: []apiv1.EnvVar{
//                                                            {
//                                                              Name: "POSTGRES_PASSWORD",
//                                                              Value: PGPASSWORD,
//                                                            },
//                                                         },
//                                                 },
//                                         },
//                                 },
//                         },
//                 },
//         }

//         // Create Deployment
//         fmt.Println("Creating deployment...")
//         result, err := deploymentsClient.Create(deployment)
//         if err != nil {
//                 panic(err)
//         }
//         fmt.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())
//         fmt.Printf("------------------------------\n")

//         // Create Service
//         fmt.Printf("Creating service...\n")
//         serviceClient := c.kubeclientset.CoreV1().Services(apiv1.NamespaceDefault)
//         service := &apiv1.Service{
//                 ObjectMeta: metav1.ObjectMeta{
//                         Name: deploymentName,
//                         Labels: map[string]string{
//                                 "app": deploymentName,
//                         },
//                 },
//                 Spec: apiv1.ServiceSpec{
//                         Ports: []apiv1.ServicePort {
//                              {
//                                 Name: "my-port",
//                                 Port: 5432,
//                 TargetPort: apiutil.FromInt(5432),
//                 Protocol: apiv1.ProtocolTCP,
//                              },
//                         },
//                         Selector: map[string]string {
//                                   "app": deploymentName,
//                         },
//                         Type: apiv1.ServiceTypeNodePort,
//                 },
//         }

//         result1, err1 := serviceClient.Create(service)
//         if err1 != nil {
//                 panic(err1)
//         }
//         fmt.Printf("Created service %q.\n", result1.GetObjectMeta().GetName())
//         fmt.Printf("------------------------------\n")

//         // Parse ServiceIP and Port
//     // Minikube VM IP
//         serviceIP := MINIKUBE_IP

//         nodePort1 := result1.Spec.Ports[0].NodePort
//     nodePort := fmt.Sprint(nodePort1)
//     servicePort := nodePort
//     //fmt.Printf("NodePort:[%v]", nodePort)

//     //fmt.Println("About to get Pods")
//     time.Sleep(time.Second * 5)

//     for {
//         readyPods := 0
//         pods := getPods(c, deploymentName)
//         //fmt.Println("Got Pods:: %s", pods)
//         for _, d := range pods.Items {
//                 //fmt.Printf(" * %s %s \n", d.Name, d.Status)
//         podConditions := d.Status.Conditions
//         for _, podCond := range podConditions {
//             if podCond.Type == corev1.PodReady {
//                if podCond.Status == corev1.ConditionTrue {
//                      //fmt.Println("Pod is running.")
//                  readyPods += 1
//                  //fmt.Printf("ReadyPods:%d\n", readyPods)
//                  //fmt.Printf("TotalPods:%d\n", len(pods.Items))
//                   }
//             }
//         }
//             }
//         if readyPods >= len(pods.Items) {
//            break
//         } else {
//                fmt.Println("Waiting for Pod to get ready.")
//            // Sleep for the Pod to become active
//            time.Sleep(time.Second * 4)
//         }
//     }

//     // Wait couple of seconds more just to give the Pod some more time.
//     time.Sleep(time.Second * 2)

//     if len(setupCommands) > 0 {
//         file := createTempDBFile(setupCommands)
//         fmt.Println("Now setting up the database")
//         setupDatabase(serviceIP, servicePort, file)
//     }

//         // List Deployments
//         //fmt.Printf("Listing deployments in namespace %q:\n", apiv1.NamespaceDefault)
//         //list, err := deploymentsClient.List(metav1.ListOptions{})
//         //if err != nil {
//         //        panic(err)
//         //}
//         //for _, d := range list.Items {
//         //        fmt.Printf(" * %s (%d replicas)\n", d.Name, *d.Spec.Replicas)
//         //}

//         verifyCmd := strings.Fields("psql -h " + serviceIP + " -p " + nodePort + " -U <user> " + " -d <db-name>")
//     var verifyCmdString = strings.Join(verifyCmd, " ")
//     fmt.Printf("VerifyCmd: %v\n", verifyCmd)
//     return serviceIP, servicePort, setupCommands, verifyCmdString
// }
