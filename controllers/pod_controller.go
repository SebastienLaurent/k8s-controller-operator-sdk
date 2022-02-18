/*
Copyright 2022.

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

package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	samplev1 "github.com/SebastienLaurent/a4c-sample-operator/api/v1"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var pod corev1.Pod
	if err := r.Get(ctx, req.NamespacedName, &pod); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Delete pod request")
			return ctrl.Result{}, nil
		}

		log.Error(err, "unable to fetch Pod")
		return ctrl.Result{}, err
	}

	moduleList := &samplev1.ModuleList{}
	opts := []client.ListOption{
		client.InNamespace(req.NamespacedName.Namespace),
	}
	if err := r.List(ctx, moduleList, opts...); err != nil {
		log.Error(err, "Can't fetch Modules")
		return ctrl.Result{}, err
	}

	if len(moduleList.Items) == 0 {
		return ctrl.Result{}, nil
	} else if len(moduleList.Items) > 1 {
		log.Error(nil, "Too many module ressource", "len", len(moduleList.Items))
		return ctrl.Result{}, nil
	}

	module := moduleList.Items[0]

	val, ok := pod.Annotations["a4c/module"]

	if !ok || val != module.Spec.Module {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		if ok {
			log.Info("Updatating annotation", "pod", req.NamespacedName, "key", "a4c/module", "value", module.Spec.Module)
		} else {
			log.Info("Creating annotation", "pod", req.NamespacedName, "key", "a4c/module", "value", module.Spec.Module)
		}
		pod.ObjectMeta.Annotations["a4c/module"] = module.Spec.Module
	}

	if err := r.Update(ctx, &pod); err != nil {
		log.Error(err, "Can't annotate pod")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Watches(
			&source.Kind{Type: &samplev1.Module{}},
			handler.EnqueueRequestsFromMapFunc(r.findPodForModule),
		).
		Complete(r)
}

func (r *PodReconciler) findPodForModule(module client.Object) []reconcile.Request {
	relatedPods := &corev1.PodList{}

	listOps := &client.ListOptions{
		Namespace: module.GetNamespace(),
	}

	err := r.List(context.TODO(), relatedPods, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(relatedPods.Items))
	for i, item := range relatedPods.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}

	return requests
}
