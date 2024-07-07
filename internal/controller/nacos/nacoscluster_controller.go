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
	nacosv1alpha1 "github.com/YunWZ/nacos-operator/api/nacos/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
)

// NacosClusterReconciler reconciles a NacosCluster object
type NacosClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=nacos.yunweizhan.com.cn,resources=nacosclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nacos.yunweizhan.com.cn,resources=nacosclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nacos.yunweizhan.com.cn,resources=nacosclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NacosCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *NacosClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	nacosCluster := &nacosv1alpha1.NacosCluster{}
	if err := r.Get(ctx, req.NamespacedName, nacosCluster); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("NacosCluster resource not found. Ignoring since object must be deleted\"")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get NacosCluster")
		return ctrl.Result{}, err
	}

	// Check if the deployment already exists, if not create

	found := []*appsv1.Deployment{}
	success := 0
	for i := 0; i < *nacosCluster.Spec.Replicas; i++ {
		err := r.Get(ctx, types.NamespacedName{Namespace: nacosCluster.Namespace, Name: getDeploymentNameForNacosCluster(nacosCluster.Name, i)}, found[i])
		if err != nil && errors.IsNotFound(err) {
			dep := r.deploymentForNacosCluster(nacosCluster, i)
			found[i] = &dep
			err = r.Create(ctx, &dep)
			if err != nil {
				log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "Failed to get Deployment")
			return ctrl.Result{}, err
		}
		success++
	}

	/*	if err := r.ensureNacosDeployment(ctx, nacosCluster); err != nil {
			log.Error(err, "Failed to create or update NacosDeployment")
			return ctrl.Result{}, err
		}

		// Define a new Pod readiness condition
		podReady := &v1.PodCondition{
			Type:   v1.PodReady,
			Status: v1.ConditionTrue,
		}
		// Set the Pod's readiness condition on the NacosCluster object
		if err := r.updatePodCondition(ctx, nacosCluster, podReady); err != nil {
			log.Error(err, "Failed to update Pod condition on NacosCluster")
			return ctrl.Result{}, err
		}

		// Update the NacosCluster status with the new Pod condition
		nacosCluster.Status.Ready = true
		if err := r.Update(*/
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NacosClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nacosv1alpha1.NacosCluster{}).
		Complete(r)
}

func (r *NacosClusterReconciler) deploymentForNacosCluster(cluster *nacosv1alpha1.NacosCluster, i int) appsv1.Deployment {
	return appsv1.Deployment{}
}

func getDeploymentNameForNacosCluster(name string, i int) string {
	return name + "-" + strconv.Itoa(i)
}
