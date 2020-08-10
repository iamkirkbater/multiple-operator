package spike

import (
	"context"
	"fmt"
	"time"

	spikev1alpha1 "github.com/iamkirkbater/multiple-operator/pkg/apis/spike/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_spike")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Spike Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSpike{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

func ignoreMetadataUpdatesPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			fmt.Printf("Metaold Lock: %+v\nMetaNew Lock: %+v\n", e.MetaOld.GetAnnotations()["Locked"], e.MetaNew.GetAnnotations()["Locked"])
			wasLocked := e.MetaOld.GetAnnotations()["Locked"]
			isLocked := e.MetaNew.GetAnnotations()["Locked"]

			if isLocked == "true" {
				// do not process if it's locked.
				return false
			}

			if wasLocked == "true" && isLocked == "false" {
				// do not process if we just unlocked
				return false
			}

			return true
		},
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("spike-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Spike
	err = c.Watch(&source.Kind{Type: &spikev1alpha1.Spike{}}, &handler.EnqueueRequestForObject{}, ignoreMetadataUpdatesPredicate())
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Spike
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &spikev1alpha1.Spike{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileSpike implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSpike{}

// ReconcileSpike reconciles a Spike object
type ReconcileSpike struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Spike object and makes changes based on the state read
// and what is in the Spike.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSpike) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Spike")

	// Fetch the Spike instance
	instance := &spikev1alpha1.Spike{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// If Instance is already locked, bail out immediately
	if instance.ObjectMeta.Annotations["Locked"] == "true" {
		log.Info("Instance already locked")
		return reconcile.Result{}, nil
	}

	// Apply our lock
	instance.ObjectMeta.Annotations["Locked"] = "true"
	err = r.client.Update(context.TODO(), instance)
	if err != nil {
		if errors.IsConflict(err) {
			log.Info("Resource is locked by another process.")
			return reconcile.Result{}, nil
		}
		log.Error(err, "There was an error obtaining the lock.")
		return reconcile.Result{}, err
	}

	log.Info("We have the lock.")
	time.Sleep(5 * time.Second)
	log.Info("Unlocking")

	// Apply changes and unlock
	instance.ObjectMeta.Annotations["Locked"] = "false"
	err = r.client.Update(context.TODO(), instance)
	if err != nil {
		log.Error(err, "There was an error applying updates.")
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *spikev1alpha1.Spike) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
