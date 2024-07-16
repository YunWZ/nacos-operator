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
	"errors"
	nacosv1alpha1 "github.com/YunWZ/nacos-operator/api/nacos/v1alpha1"
	"github.com/YunWZ/nacos-operator/internal/controller/nacos/constants"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sort"
)

// NacosStandaloneReconciler reconciles a NacosStandalone object
type NacosStandaloneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
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
	_ = context.Background()
	//_ = logs.FromContext(ctx)
	_ = r.Log.WithValues("nacos-standalone", req.NamespacedName)
	ns := &nacosv1alpha1.NacosStandalone{}
	err := r.Get(ctx, req.NamespacedName, ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info("NacosStandalone resource not found. Try to delete refrence resources.")
			if err = r.deleteResourcesForNacosStandalone(ctx, req); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		r.Log.Error(err, "Failed to get NacosStandalone")
		return ctrl.Result{}, err
	}

	requeue, err := r.completePVC(ns)
	if err != nil {
		return ctrl.Result{Requeue: requeue}, err
	} else if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	requeue, err = r.completeDeployment(ns)
	if err != nil {
		return ctrl.Result{Requeue: requeue}, err
	} else if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	requeue, err = r.completeService(ns)
	if err != nil {
		return ctrl.Result{Requeue: requeue}, err
	} else if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	requeue, err = r.updateStatusForNacosStandalone(ns)
	if err != nil {
		return ctrl.Result{Requeue: requeue}, err
	} else if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NacosStandaloneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nacosv1alpha1.NacosStandalone{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
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
							{
								Name:          constants.DefaultNacosServerPeerToPeerPortName,
								ContainerPort: constants.DefaultNacosServerPeerToPeerPort,
							},
						},
						Env: []corev1.EnvVar{
							{Name: "MODE", Value: "standalone"},
						},
					}},
					ImagePullSecrets: ns.Spec.ImagePullSecrets,
				},
			},
		},
	}
	if ns.Spec.Image != "" {
		dep.Spec.Template.Spec.Containers[0].Image = ns.Spec.Image
	}

	if ns.Spec.Pvc != nil {
		dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: r.generatePVCName(ns.Name),
					ReadOnly:  false,
				},
			},
		})
		dep.Spec.Template.Spec.Containers[0].VolumeMounts = append(dep.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "data",
			ReadOnly:  false,
			MountPath: "/home/nacos/data",
			SubPath:   "data",
		})
		dep.Spec.Template.Spec.Containers[0].VolumeMounts = append(dep.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "data",
			ReadOnly:  false,
			MountPath: "/home/nacos/logs",
			SubPath:   "logs",
		})
	}
	return dep

}

func (r *NacosStandaloneReconciler) generatePVCName(name string) string {
	return name + "-data"
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
				{
					Name:       constants.DefaultNacosServerPeerToPeerPortName,
					TargetPort: intstr.FromString(constants.DefaultNacosServerPeerToPeerPortName),
					Port:       constants.DefaultNacosServerPeerToPeerPort,
				},
			},
			Selector: ls,
			Type:     ns.Spec.Service.Type,
		},
	}

	return
}

func (r *NacosStandaloneReconciler) deleteDeployment(ns types.NamespacedName) error {
	deploy := &appsv1.Deployment{}
	err := r.Get(context.TODO(), ns, deploy)

	if err != nil && apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		r.Log.Error(err, "Delete Deployment failed")
		return err
	}

	return r.Delete(context.TODO(), deploy)
}

func (r *NacosStandaloneReconciler) deleteService(ns types.NamespacedName) error {
	svc := &corev1.Service{}
	err := r.Get(context.TODO(), ns, svc)

	if err != nil && apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		r.Log.Error(err, "Delete Service failed")
		return err
	}

	return r.Delete(context.TODO(), svc)
}

func (r *NacosStandaloneReconciler) deleteResourcesForNacosStandalone(ctx context.Context, req ctrl.Request) (err error) {
	if err = r.deleteDeployment(req.NamespacedName); err != nil {
		return err
	}

	if err = r.deleteService(req.NamespacedName); err != nil {
		return err
	}
	err = r.deletePVC(req.NamespacedName)
	return err
}

func (r *NacosStandaloneReconciler) persistentVolumeClaimForNacosStandalone(ns *nacosv1alpha1.NacosStandalone) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.generatePVCName(ns.Name),
			Namespace: ns.Namespace,
			Labels:    labelsForNacosStandalone(ns),
		},
		Spec:   corev1.PersistentVolumeClaimSpec{},
		Status: corev1.PersistentVolumeClaimStatus{},
	}

	return pvc
}

func (r *NacosStandaloneReconciler) checkPVCExist(ns *nacosv1alpha1.NacosStandalone) (*corev1.PersistentVolumeClaim, bool, error) {
	pvc := &corev1.PersistentVolumeClaim{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: r.generatePVCName(ns.Name), Namespace: ns.Namespace}, pvc)
	if err == nil {
		return pvc, true, nil
	}

	if apierrors.IsNotFound(err) {
		return nil, false, nil
	}
	return nil, false, err

}

func (r *NacosStandaloneReconciler) completePVC(ns *nacosv1alpha1.NacosStandalone) (bool, error) {
	// Check PVC if exist
	pvc, pvcExists, err := r.checkPVCExist(ns)
	if err != nil {
		r.Log.Error(err, "Failed to check if PVC exists")
		return true, err
	}
	// Needed to delete pvc.
	if ns.Spec.Pvc == nil {
		if pvcExists {
			err = r.Delete(context.TODO(), pvc)
			if err != nil {
				r.Log.Error(err, "Failed to delete PVC")
				return true, err
			}

		}
		r.Log.Info("NacosStandalone CR doesn't has a pvc", "instance", ns.Namespace+"/"+ns.Name)
		return false, nil
	}

	if ns.Spec.Pvc != nil && !pvcExists {
		// Needed to create pvc.
		pvc := r.persistentVolumeClaimForNacosStandalone(ns)
		if err = r.Create(context.TODO(), pvc); err != nil {
			r.Log.Error(err, "PVC.Namespace: %s , PVC.Name: %s", pvc.Namespace, pvc.Name)
			return true, err
		}
		r.Log.Info("Create PVC successfully!")
		return true, nil
	}

	if !reflect.DeepEqual(ns.Spec.Pvc.Resources.Requests.Storage(), pvc.Spec.Resources.Requests.Storage()) {
		pvc.Spec.Resources = ns.Spec.Pvc.Resources
		err = r.Update(context.TODO(), pvc)
		if err != nil {
			r.Log.Error(err, "Failed to update PVC")
			return true, err
		}
		return true, nil
	}
	r.Log.Info("Abnormal logic, please check the NacosStandaloneReconciler.completePVC() method")
	return false, errors.New("unknown error")
}

func (r *NacosStandaloneReconciler) completeDeployment(ns *nacosv1alpha1.NacosStandalone) (requeue bool, err error) {
	found := &appsv1.Deployment{}
	err = r.Get(context.TODO(), types.NamespacedName{
		Namespace: ns.Namespace,
		Name:      ns.Name,
	}, found)

	if err != nil && apierrors.IsNotFound(err) {
		dep := r.deploymentForNacosStandalone(ns)
		if err = r.Create(context.TODO(), dep); err != nil {
			r.Log.Error(err, "Deployment.Namespace: %s , Deployment.Name: %s", dep.Namespace, dep.Name)
			return true, err
		}
		// Deployment created successfully - return and requeue
		return true, nil
	} else if err != nil {
		r.Log.Error(err, "Failed to get Deployment")
		return true, err
	}

	needUpdate := false
	if !reflect.DeepEqual(found.Spec.Template.Spec.Containers[0].Image, ns.Spec.Image) {
		needUpdate = true
		found.Spec.Template.Spec.Containers[0].Image = ns.Spec.Image
	}

	size := int32(1)
	if *found.Spec.Replicas != size {
		needUpdate = true
		found.Spec.Replicas = &size
	}

	if needUpdate {
		if err = r.Update(context.TODO(), found); err != nil {
			r.Log.Error(err, "Failed to update Deployment, Deployment.Namespace: %s, Deployment.Name: %s", found.Namespace, found.Name)
			return true, err
		}
		// Spec updated - return and requeue
		return true, nil
	}
	// Update the NacosStandalone status with the pod names
	// List the pods for this NacosStandalone's deployment
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{client.InNamespace(ns.Namespace), client.MatchingLabels(labelsForNacosStandalone(ns))}
	if err = r.List(context.TODO(), podList, listOpts...); err != nil {
		r.Log.Error(err, "Failed to list pods, NacosStandalone.Namespace: %s , NacosStandalone.Name: %s", ns.Namespace, ns.Name)
		return true, err
	}

	return false, nil
}

func (r *NacosStandaloneReconciler) completeService(ns *nacosv1alpha1.NacosStandalone) (requeue bool, err error) {
	svc := &corev1.Service{}
	err = r.Get(context.TODO(), types.NamespacedName{
		Namespace: ns.Namespace,
		Name:      ns.Name,
	}, svc)
	if err != nil && apierrors.IsNotFound(err) {
		svc = r.serviceForNacosStandalone(ns)
		if err = r.Create(context.TODO(), svc); err != nil {
			r.Log.Error(err, "Service.Namespace: %s , Service.Name: %s", svc.Namespace, svc.Name)
			return true, err
		}
		return true, nil
	} else if err != nil {
		r.Log.Error(err, "Failed to get Service")
		return true, err
	}

	if svc.Spec.Type != ns.Spec.Service.Type {
		svc.Spec.Type = ns.Spec.Service.Type
		if err = r.Update(context.TODO(), svc); err != nil {
			r.Log.Error(err, "Service.Namespace: %s , Service.Name: %s", svc.Namespace, svc.Name)
			return true, err
		}
		return true, nil
	}

	return false, nil
}

func (r *NacosStandaloneReconciler) deletePVC(name types.NamespacedName) error {
	err := r.Delete(context.TODO(), &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.generatePVCName(name.Name),
			Namespace: name.Namespace,
		},
	})
	if err != nil && apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (r *NacosStandaloneReconciler) updateStatusForNacosStandalone(ns *nacosv1alpha1.NacosStandalone) (bool, error) {
	dep := &appsv1.Deployment{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: ns.Namespace, Name: ns.Name}, dep)
	if err != nil && apierrors.IsNotFound(err) {
		r.Log.Error(err, "Failed to get Deployment.")
		return true, nil
	}
	if len(dep.Status.Conditions) == 0 {
		return true, nil
	}

	cond := appsv1.DeploymentCondition{}
	dep.Status.Conditions[0].DeepCopyInto(&cond)

	needUpdate := false
	if len(ns.Status.Conditions) != 0 {
		sort.Slice(ns.Status.Conditions, func(i, j int) bool {
			return ns.Status.Conditions[i].LastUpdateTime.After(ns.Status.Conditions[j].LastUpdateTime.Time)
		})
		if !reflect.DeepEqual(cond, ns.Status.Conditions[0]) {
			needUpdate = true
			ns.Status.Conditions = append(ns.Status.Conditions, cond)
		}
	} else {
		needUpdate = true
		ns.Status.Conditions = append(dep.Status.Conditions, cond)
	}

	if needUpdate {
		err = r.Status().Update(context.TODO(), ns)
		if err != nil && apierrors.IsNotFound(err) {
			return false, err
		} else if err != nil {
			r.Log.Error(err, "Failed to update status for NacosStandalone.")
			return true, err
		}
	}

	return false, nil
}

func labelsForNacosStandalone(ns *nacosv1alpha1.NacosStandalone) map[string]string {
	return map[string]string{constants.LabelNacosStandalone: ns.Name, constants.LabelApp: ns.Name}
}
