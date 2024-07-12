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

package nacos

import (
	"context"
	"github.com/YunWZ/nacos-operator/internal/controller/nacos/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logs "sigs.k8s.io/controller-runtime/pkg/log"

	nacosv1alpha1 "github.com/YunWZ/nacos-operator/api/nacos/v1alpha1"
)

// NacosStandaloneReconciler reconciles a NacosStandalone object
type NacosStandaloneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=nacos.yunweizhan.com.cn,resources=nacosstandalones,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nacos.yunweizhan.com.cn,resources=nacosstandalones/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nacos.yunweizhan.com.cn,resources=nacosstandalones/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list,watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *NacosStandaloneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logs.FromContext(ctx)

	ns := &nacosv1alpha1.NacosStandalone{}
	err := r.Get(ctx, req.NamespacedName, ns)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("NacosStandalone resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get NacosStandalone")
		return ctrl.Result{}, err
	}

	found := &appsv1.Deployment{}
	err = r.Get(ctx, req.NamespacedName, found)
	if err != nil && errors.IsNotFound(err) {
		dep := r.deploymentForNacosStandalone(ns)
		if err = r.Create(ctx, dep); err != nil {
			log.Error(err, "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	size := int32(1)
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}

	// Update the NacosStandalone status with the pod names
	// List the pods for this NacosStandalone's deployment
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{client.InNamespace(ns.Namespace), client.MatchingLabels(labelsForNacosStandalone(ns))}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "NacosStandalone.Namespace", ns.Namespace, "NacosStandalone.Name", ns.Name)
		return ctrl.Result{}, err
	}

	podNames := getPodName(podList.Items)
	if !reflect.DeepEqual(podNames, ns.Status.Nodes) {
		ns.Status.Nodes = podNames
		err = r.Update(ctx, ns)
		if err != nil {
			log.Error(err, "Failed to update NacosStandalone status")
			return ctrl.Result{}, err
		}
	}

	svc := corev1.Service{}
	err = r.Get(ctx, req.NamespacedName, &svc)
	if err != nil && errors.IsNotFound(err) {
		serv := r.serviceForNacosStandalone(ns)
		if err = r.Create(ctx, serv); err != nil {
			log.Error(err, "Service.Namespace", serv.Namespace, "Service.Name", serv.Name)
			return ctrl.Result{Requeue: true}, err
		}
	} else if err != nil {
		log.Error(err, "Failed to get Service")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func getPodName(items []corev1.Pod) []string {
	res := make([]string, 1)
	for _, pod := range items {
		res = append(res, pod.Name)
	}
	return res
}

// SetupWithManager sets up the controller with the Manager.
func (r *NacosStandaloneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nacosv1alpha1.NacosStandalone{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func (r *NacosStandaloneReconciler) deploymentForNacosStandalone(ns *nacosv1alpha1.NacosStandalone) *appsv1.Deployment {
	ls := labelsForNacosStandalone(ns)
	replicas := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ns.Name,
			Namespace: ns.Namespace,
			Labels:    ls,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: ls},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: ls},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "nacos-server",
						Image: constants.DefaultImage,
						Ports: []corev1.ContainerPort{
							{
								Name:          constants.DefaultNacosServerHttpPortName,
								ContainerPort: constants.DefaultNacosServerHttpPort,
							},
							{
								Name:          constants.DefaultNacosServerGrpcPortName,
								ContainerPort: constants.DefaultNacosServerGrpcPort,
							},
							{
								Name:          constants.DefaultNacosServerRaftPortName,
								ContainerPort: constants.DefaultNacosServerRaftPort,
							},
						},
					}},
					ImagePullSecrets: ns.Spec.ImagePullSecrets,
				},
			},
		},
	}
	return dep

}

func (r *NacosStandaloneReconciler) serviceForNacosStandalone(ns *nacosv1alpha1.NacosStandalone) (svc *corev1.Service) {
	ls := labelsForNacosStandalone(ns)
	svc = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: ns.Name, Namespace: ns.Namespace, Labels: ls},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       constants.DefaultNacosServerHttpPortName,
					TargetPort: intstr.FromString(constants.DefaultNacosServerHttpPortName),
					Port:       constants.DefaultNacosServerHttpPort,
				},
				{
					Name:       constants.DefaultNacosServerGrpcPortName,
					TargetPort: intstr.FromString(constants.DefaultNacosServerGrpcPortName),
					Port:       constants.DefaultNacosServerGrpcPort,
				},
				{
					Name:       constants.DefaultNacosServerRaftPortName,
					TargetPort: intstr.FromString(constants.DefaultNacosServerRaftPortName),
					Port:       constants.DefaultNacosServerRaftPort,
				},
			},
			Selector: ls,
		},
	}
	return
}

func labelsForNacosStandalone(ns *nacosv1alpha1.NacosStandalone) map[string]string {
	return map[string]string{constants.LabelNacosStandalone: ns.Name, constants.LabelApp: ns.Name}
}
