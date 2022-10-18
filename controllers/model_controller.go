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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mlv1alpha1 "github.com/kalkyai/model-serving-operator/api/v1alpha1"
	model "github.com/kalkyai/model-serving-operator/pkg/model"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// ModelReconciler reconciles a Model object
type ModelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// called when trying to create new resource.
func (r *ModelReconciler) reconcileCreate(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	ctrllog := log.FromContext(ctx).WithValues("models", req.NamespacedName)

	model_serving := &mlv1alpha1.Model{}
	err := r.Get(ctx, req.NamespacedName, model_serving)

	mod := &model.ModelServing{
		Name:      model_serving.Name,
		Replicas:  model_serving.Spec.Replicas,
		ModelURL:  model_serving.Spec.Location,
		Columns:   model_serving.Spec.Columns,
		Namespace: model_serving.Namespace,
		Version:   model_serving.Spec.Version,
		AccessKey: model_serving.Spec.Accesskey,
		SecretKey: model_serving.Spec.SecretKey,
		Endpoint:  model_serving.Spec.Endpoint,
		Bucket:    model_serving.Spec.Bucket,
	}

	config := mod.CreateConfigMap(ctx,
		model_serving.Spec.Location,
		model_serving.Spec.Columns,
		model_serving.Spec.Accesskey,
		model_serving.Spec.SecretKey,
		model_serving.Spec.Endpoint,
		model_serving.Spec.Bucket,
	)

	ctrllog.Info("Creating new volume")
	volume := mod.CreateVolume(ctx)
	deployment := mod.CreateDeployment(ctx, volume)
	service := mod.CreateService(ctx)

	ctrl.SetControllerReference(model_serving, volume, r.Scheme)
	ctrl.SetControllerReference(model_serving, deployment, r.Scheme)
	ctrl.SetControllerReference(model_serving, service, r.Scheme)
	ctrl.SetControllerReference(model_serving, config, r.Scheme)

	ctrllog.Info("Creating ConfigMap")
	err = r.Create(ctx, config)

	if err != nil {
		if apierrors.IsBadRequest(err) {
			ctrllog.Error(err, "Failed to create new configmap")
			return ctrl.Result{}, err
		}
	}

	ctrllog.Info("Creating Deployment")
	err = r.Create(ctx, deployment)

	if err != nil {
		if apierrors.IsBadRequest(err) {
			ctrllog.Error(err, "Failed to create new deployment")
			return ctrl.Result{}, err
		}
	}

	ctrllog.Info("Creating Service")
	err = r.Create(ctx, service)
	if err != nil {
		if apierrors.IsBadRequest(err) {
			ctrllog.Error(err, "Failed to create new service")
			return ctrl.Result{}, err
		}
		ctrllog.Error(err, "Failed to create new service")
	}

	return ctrl.Result{Requeue: true}, nil

}

//+kubebuilder:rbac:groups=ml.kalkyai.com,resources=models,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ml.kalkyai.com,resources=models/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ml.kalkyai.com,resources=models/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pvc,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Model object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *ModelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctrllog := log.FromContext(ctx).WithValues("models", req.NamespacedName)

	ctrllog.Info("Initializing Reconcile")
	model_serving := &mlv1alpha1.Model{}
	err := r.Get(ctx, req.NamespacedName, model_serving)

	ctrllog.Info(fmt.Sprintf("%s", err))

	if err != nil {
		if apierrors.IsNotFound(err) {
			ctrllog.Error(err, "Model not found")
			return ctrl.Result{}, nil
		}
		ctrllog.Info("Model found")
		return ctrl.Result{}, err
	}

	ctrllog.Info("Model Found")

	// See if the statefulset exists
	statefulset := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: model_serving.Namespace,
		Name:      model_serving.Name,
	}, statefulset)

	if err != nil {
		// create new resource if not found
		if apierrors.IsNotFound(err) {
			ctrllog.Info("Error --- statefulset not found create a new one")
			return r.reconcileCreate(ctx, req)
		}
		ctrllog.Info("Model found")
		return ctrl.Result{}, err
	}

	// Handle if not already exists

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ModelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mlv1alpha1.Model{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
