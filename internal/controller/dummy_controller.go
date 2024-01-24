/*
Copyright 2024.

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

package controller

import (
	"context"
	"errors"
	"time"

	interviewv1alpha1 "github.com/bi6o/a9s-challenge/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	containerImageNginx = "nginx:latest"
	containerNameNginx  = "nginx"

	kindPod = "Pod"
)

// DummyReconciler reconciles a Dummy object
type DummyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=interview.a9s-interview.com,resources=dummies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=interview.a9s-interview.com,resources=dummies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=interview.a9s-interview.com,resources=dummies/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DummyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	dummy := &interviewv1alpha1.Dummy{}
	if err := r.Get(ctx, req.NamespacedName, dummy); err != nil {
		logger.Error(err, "Failed to get dummy from k8s api", "Name", dummy.Name)
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling Dummy", "Name", dummy.Name, "Namespace", dummy.Namespace, "Message", dummy.Spec.Message)

	dummy.Status.SpecEcho = dummy.Spec.Message
	if err := r.Status().Update(ctx, dummy); err != nil {
		logger.Error(err, "Failed to update dummy status", "Name", dummy.Name)
		return ctrl.Result{}, err
	}

	// Check if the pod already exists before creating a new one
	existingPod := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Name: dummy.Name + "-pod", Namespace: dummy.Namespace}, existingPod)
	switch {
	case err != nil && !k8serrors.IsNotFound(err):
		logger.Error(err, "Failed to check if pod already exists", "Name", dummy.Name)
		return ctrl.Result{}, err

	case err == nil:
		logger.Info("Pod already exists, updating dummy status with existing pod status", "Name", dummy.Name)

		dummy.Status.PodStatus = string(existingPod.Status.Phase)
		if err := r.Status().Update(ctx, dummy); err != nil {
			logger.Error(err, "Failed to update dummy's pod status", "Name", dummy.Name)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	logger.Info("Creating Pod for Dummy", "Name", dummy.Name)

	pod, err := r.createPodForDummy(ctx, logger, dummy)
	if err != nil {
		logger.Error(err, "Failed to create pod for dummy", "Name", dummy.Name)
		return ctrl.Result{}, err
	}

	pod, err = r.getPodWithRetry(ctx, pod)
	if err != nil {
		logger.Error(err, "Failed to get pod from k8 after creating it", "Name", pod.Name)
		return ctrl.Result{}, err
	}

	dummy.Status.PodStatus = string(pod.Status.Phase)
	if err := r.Status().Update(ctx, dummy); err != nil {
		logger.Error(err, "Failed to update dummy's pod status after creating it", "Name", dummy.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DummyReconciler) getPodWithRetry(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error) {
	timeout := time.After(30 * time.Second)
	tick := time.NewTicker(500 * time.Millisecond)

	for {
		select {
		case <-timeout:
			return nil, errors.New("timed out waiting for pod to be created")
		case <-tick.C:
			err := r.Get(ctx, types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, pod)
			if err == nil {
				return pod, nil
			}
		}
	}
}

func (r *DummyReconciler) createPodForDummy(ctx context.Context, logger logr.Logger, dummy *interviewv1alpha1.Dummy) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       kindPod,
			APIVersion: dummy.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dummy.Name + "-pod",
			Namespace: dummy.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dummy.APIVersion,
					Kind:       dummy.Kind,
					Name:       dummy.Name,
					UID:        dummy.UID,
				},
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  containerNameNginx,
					Image: containerImageNginx,
				},
			},
		},
	}

	err := r.Create(ctx, pod)
	if err != nil {
		return nil, err
	}

	return pod, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DummyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&interviewv1alpha1.Dummy{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
