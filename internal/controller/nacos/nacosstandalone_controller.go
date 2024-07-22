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
	"fmt"
	nacosv1alpha1 "github.com/YunWZ/nacos-operator/api/nacos/v1alpha1"
	"github.com/YunWZ/nacos-operator/internal/controller/nacos/constants"
	"github.com/YunWZ/nacos-operator/internal/util"
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
	"strconv"
	"strings"
)

var size = int32(1)

// NacosStandaloneReconciler reconciles a NacosStandalone object
type NacosStandaloneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=nacos.yunweizhan.com.cn,resources=nacosstandalones,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nacos.yunweizhan.com.cn,resources=nacosstandalones/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nacos.yunweizhan.com.cn,resources=nacosstandalones/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods;configmaps;secrets,verbs=get;list;
// +kubebuilder:rbac:groups=core,resources=services;persistentvolumes,verbs=get;list,watch;create;update;patch;delete
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
			r.Log.Info("NacosStandalone resource not found. Try to delete all referenced resources.", "NacosStandalone", req.NamespacedName)
			if err = r.deleteResourcesForNacosStandalone(ctx, req); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		r.Log.Error(err, "Failed to get NacosStandalone")
		return ctrl.Result{}, err
	}

	requeue, err := r.completeProbeForNacosStandalone(ns)
	if err != nil {
		return util.NewResult(requeue), err
	} else if requeue {
		return util.NewResult(true), nil
	}

	requeue, err = r.completePVCForNacosStandalone(ns)
	if err != nil {
		return util.NewResult(requeue), err
	} else if requeue {
		return util.NewResult(true), nil
	}

	requeue, err = r.completeDeploymentForNacosStandalone(ns)
	if err != nil {
		return util.NewResult(requeue), err
	} else if requeue {
		return util.NewResult(true), nil
	}

	requeue, err = r.completeServiceForNacosStandalone(ns)
	if err != nil {
		return util.NewResult(requeue), err
	} else if requeue {
		return util.NewResult(true), nil
	}

	requeue, err = r.updateStatusForNacosStandalone(ns)
	if err != nil {
		return util.NewResult(requeue), err
	} else if requeue {
		return util.NewResult(true), nil
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

func (r *NacosStandaloneReconciler) deploymentForNacosStandalone(ns *nacosv1alpha1.NacosStandalone) (dep *appsv1.Deployment) {
	ls := labelsForNacosStandalone(ns)
	replicas := size
	dep = &appsv1.Deployment{
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
						Image:           constants.DefaultImage,
						ImagePullPolicy: ns.Spec.ImagePullPolicy,
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
							{Name: constants.EnvMode, Value: constants.NacosModeStandalone},
						},
						LivenessProbe:  ns.Spec.LivenessProbe,
						ReadinessProbe: ns.Spec.ReadinessProbe,
						StartupProbe:   ns.Spec.StartupProbe,
						Resources:      ns.Spec.Resources,
					}},
					ImagePullSecrets: ns.Spec.ImagePullSecrets,
				},
			},
		},
	}

	if ns.Spec.Image != "" {
		dep.Spec.Template.Spec.Containers[0].Image = ns.Spec.Image
	}

	volumes := r.generateVolumesForDeployment(ns)

	dep.Spec.Template.Spec.Volumes = volumes

	volumeMounts := r.generateVolumeMountsForDeployment(ns)

	dep.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts

	_ = r.processDatabaseEnvForDeployment(ns, dep)

	_ = r.processJvmOptionsForDeployment(ns, dep)

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
	r.Log.Info("Try to delete deployment resources.", "NacosStandalone", req.NamespacedName)
	if err = r.deleteDeployment(req.NamespacedName); client.IgnoreNotFound(err) != nil {
		return err
	}

	r.Log.Info("Try to delete service resources.", "NacosStandalone", req.NamespacedName)
	if err = r.deleteService(req.NamespacedName); client.IgnoreNotFound(err) != nil {
		return err
	}

	r.Log.Info("Try to delete pvc resources.", "NacosStandalone", req.NamespacedName)
	err = r.deletePVC(req.NamespacedName)
	return client.IgnoreNotFound(err)
}

func (r *NacosStandaloneReconciler) persistentVolumeClaimForNacosStandalone(ns *nacosv1alpha1.NacosStandalone) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.generatePVCName(ns.Name),
			Namespace: ns.Namespace,
			Labels:    labelsForNacosStandalone(ns),
		},
		Spec:   *ns.Spec.Pvc,
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

	return nil, false, client.IgnoreNotFound(err)

}

func (r *NacosStandaloneReconciler) completePVCForNacosStandalone(ns *nacosv1alpha1.NacosStandalone) (requeue bool, err error) {
	r.Log.Info("Try to completePVCForNacosStandalone.", "NacosStandalone", types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
	// Needed to delete pvc.
	if ns.Spec.Pvc == nil {
		return false, nil
	}

	// Check PVC if exist
	pvc, pvcExists, err := r.checkPVCExist(ns)
	if err != nil {
		r.Log.Error(err, "Failed to check if PVC exists")
		return true, err
	}

	if ns.Spec.Pvc != nil && !pvcExists {
		// Needed to create pvc.
		pvc := r.persistentVolumeClaimForNacosStandalone(ns)
		if err = r.Create(context.TODO(), pvc); client.IgnoreAlreadyExists(err) != nil {
			r.Log.Error(err, "PVC.Namespace: %s , PVC.Name: %s", pvc.Namespace, pvc.Name)
			return true, err
		}
		r.Log.Info("PVC already exists")
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
	r.Log.Info("Abnormal logic, please check the NacosStandaloneReconciler.completePVCForNacosStandalone() method")
	return false, errors.New("unknown error")
}

func (r *NacosStandaloneReconciler) completeDeploymentForNacosStandalone(ns *nacosv1alpha1.NacosStandalone) (requeue bool, err error) {
	r.Log.Info("Try to completeDeploymentForNacosStandalone", "NacosStandalone", types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
	found := &appsv1.Deployment{}
	err = r.Get(context.TODO(), types.NamespacedName{
		Namespace: ns.Namespace,
		Name:      ns.Name,
	}, found)

	if err != nil && apierrors.IsNotFound(err) {
		dep := r.deploymentForNacosStandalone(ns)

		err = r.Create(context.TODO(), dep)
		return true, client.IgnoreAlreadyExists(err)
	} else if err != nil {
		r.Log.Error(err, "Failed to get Deployment")
		return true, err
	}

	needUpdate := r.processDatabaseEnvForDeployment(ns, found)

	if !reflect.DeepEqual(found.Spec.Template.Spec.Containers[0].Image, ns.Spec.Image) {
		needUpdate = true
		found.Spec.Template.Spec.Containers[0].Image = ns.Spec.Image
	}
	if !reflect.DeepEqual(found.Spec.Template.Spec.Containers[0].ImagePullPolicy, ns.Spec.ImagePullPolicy) {
		needUpdate = true
		found.Spec.Template.Spec.Containers[0].ImagePullPolicy = ns.Spec.ImagePullPolicy
	}
	if !reflect.DeepEqual(found.Spec.Template.Spec.ImagePullSecrets, ns.Spec.ImagePullSecrets) {
		needUpdate = true
		found.Spec.Template.Spec.ImagePullSecrets = ns.Spec.ImagePullSecrets
	}

	if r.checkVolumeChanged(found, ns) {
		needUpdate = true
		volumes := r.generateVolumesForDeployment(ns)

		found.Spec.Template.Spec.Volumes = volumes
		mounts := r.generateVolumeMountsForDeployment(ns)

		found.Spec.Template.Spec.Containers[0].VolumeMounts = mounts
	}

	needUpdate = r.processJvmOptionsForDeployment(ns, found) || needUpdate

	if *found.Spec.Replicas != size {
		needUpdate = true
		found.Spec.Replicas = &size
	}

	if ns.Spec.ReadinessProbe != nil && !reflect.DeepEqual(ns.Spec.ReadinessProbe, found.Spec.Template.Spec.Containers[0].ReadinessProbe) {
		needUpdate = true
		found.Spec.Template.Spec.Containers[0].ReadinessProbe = ns.Spec.ReadinessProbe
	}
	if ns.Spec.LivenessProbe != nil && !reflect.DeepEqual(ns.Spec.LivenessProbe, found.Spec.Template.Spec.Containers[0].LivenessProbe) {
		needUpdate = true
		found.Spec.Template.Spec.Containers[0].LivenessProbe = ns.Spec.LivenessProbe
	}
	if ns.Spec.StartupProbe != nil && !reflect.DeepEqual(ns.Spec.StartupProbe, found.Spec.Template.Spec.Containers[0].StartupProbe) {
		needUpdate = true
		found.Spec.Template.Spec.Containers[0].StartupProbe = ns.Spec.StartupProbe
	}

	if !reflect.DeepEqual(ns.Spec.Resources, found.Spec.Template.Spec.Containers[0].Resources) {
		needUpdate = true
		found.Spec.Template.Spec.Containers[0].Resources = ns.Spec.Resources
	}

	if needUpdate {
		if err = r.Update(context.TODO(), found); err != nil {
			r.Log.Error(err, "Failed to update Deployment, Deployment.Namespace: %s, Deployment.Name: %s", found.Namespace, found.Name)
			return true, err
		}
		// Spec updated - return and requeue
		return true, nil
	}

	return false, nil
}

func (r *NacosStandaloneReconciler) completeServiceForNacosStandalone(ns *nacosv1alpha1.NacosStandalone) (requeue bool, err error) {
	r.Log.Info("Try to completeServiceForNacosStandalone.", "NacosStandalone", types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
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
	r.Log.Info("Try to updateStatusForNacosStandalone.", "NacosStandalone", types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
	dep := &appsv1.Deployment{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: ns.Namespace, Name: ns.Name}, dep)
	if err != nil && apierrors.IsNotFound(err) {
		r.Log.Error(err, "Failed to get Deployment.")
		return true, nil
	}
	if len(dep.Status.Conditions) == 0 {
		return true, nil
	}

	sort.Slice(dep.Status.Conditions, func(i, j int) bool {
		return dep.Status.Conditions[i].LastUpdateTime.After(dep.Status.Conditions[j].LastUpdateTime.Time)
	})

	cond := dep.Status.Conditions[0]

	needUpdate := false
	if len(ns.Status.Conditions) != 0 {
		sort.Slice(ns.Status.Conditions, func(i, j int) bool {
			return ns.Status.Conditions[i].LastUpdateTime.After(ns.Status.Conditions[j].LastUpdateTime.Time)
		})
		if !reflect.DeepEqual(cond, ns.Status.Conditions[0]) {
			needUpdate = true
			ns.Status.Conditions = dep.Status.Conditions
		}
	} else {
		needUpdate = true
		ns.Status.Conditions = dep.Status.Conditions
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

	if dep.Status.AvailableReplicas == 1 {
		return false, nil
	}

	return true, nil
}

func (r *NacosStandaloneReconciler) completeProbeForNacosStandalone(ns *nacosv1alpha1.NacosStandalone) (bool, error) {
	r.Log.Info("Try to completeProbeForNacosStandalone.", "NacosStandalone", types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
	needUpdate := false
	if ns.Spec.LivenessProbe == nil {
		needUpdate = true
		ns.Spec.LivenessProbe = util.NewDefaultLivenessProbe()
	}

	if ns.Spec.ReadinessProbe == nil {
		needUpdate = true
		ns.Spec.ReadinessProbe = util.NewDefaultReadinessProbe()
	}

	if ns.Spec.StartupProbe == nil {
		needUpdate = true
		ns.Spec.StartupProbe = util.NewDefaultStartupProbe()
	}

	if needUpdate {
		r.Log.Info("Update NacosStandalone with default probes.", "NacosStandalone", types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
		err := r.Update(context.TODO(), ns)
		if err != nil && apierrors.IsNotFound(err) {
			return false, err
		} else if err != nil {
			return true, err
		}
	}

	return needUpdate, nil
}

func (r *NacosStandaloneReconciler) processDatabaseEnvForDeployment(ns *nacosv1alpha1.NacosStandalone, dep *appsv1.Deployment) (needUpdate bool) {
	// TODO: Maybe could support others database.
	var newEnv, oldDbEnv []corev1.EnvVar
	oldEnv := dep.Spec.Template.Spec.Containers[0].Env
	for _, item := range oldEnv {
		switch item.Name {
		case constants.EnvDBNum, constants.EnvDBPassword, constants.EnvDatabasePlatform, constants.EnvDBUser:
			oldDbEnv = append(oldDbEnv, item)
			continue
		}
		if strings.HasPrefix(item.Name, constants.EnvDBUrlPrefix) {
			oldDbEnv = append(oldDbEnv, item)
		} else {
			newEnv = append(newEnv, item)
		}
	}

	existed := len(oldDbEnv) != 0
	if !existed {
		if ns.Spec.Database == nil {
			return false
		}

		newDbEnv := r.generateDataBaseEnvForMysql(ns)
		newEnv = append(newEnv, newDbEnv...)
		needUpdate = true
	} else {
		if ns.Spec.Database == nil {
			needUpdate = true
		} else {
			newDbEnv := r.generateDataBaseEnvForMysql(ns)
			if !util.EnvVarsEqual(oldDbEnv, newDbEnv) {
				newEnv = append(newEnv, newDbEnv...)
				needUpdate = true
			}
		}
	}

	/*	if existed && ns.Spec.Database != nil {
		if ns.Spec.Database.Mysql != nil {
			newDbEnv := r.generateDataBaseEnvForMysql(ns)
			if !util.EnvVarsEqual(oldDbEnv, newDbEnv) {
				newEnv = append(newEnv, newDbEnv...)
			}
		} else {
			r.Log.Info("Unsupported database type for NacosStandalone, check your database spec")
			return false
		}
	}*/

	if needUpdate {
		dep.Spec.Template.Spec.Containers[0].Env = newEnv
	}

	return needUpdate
}

func (r *NacosStandaloneReconciler) generateDataBaseEnvForMysql(ns *nacosv1alpha1.NacosStandalone) (envs []corev1.EnvVar) {
	if ns.Spec.Database == nil {
		return
	}
	envs = append(envs, corev1.EnvVar{
		Name:  constants.EnvDatabasePlatform,
		Value: "mysql",
	})
	envs = append(envs, corev1.EnvVar{
		Name:  constants.EnvDBNum,
		Value: strconv.Itoa(len(ns.Spec.Database.Mysql.DbServer)),
	})

	secret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Namespace: ns.Namespace,
		Name:      ns.Spec.Database.Mysql.Secret.Name,
	}
	err := r.Get(context.TODO(), secretName, secret)
	if err != nil {
		r.Log.Error(err, "Failed to get secret. The secret must be created manually")
	}

	if _, found := secret.Data["user"]; !found {
		err = errors.New("user not found in secret. The secret must be contained in 'user' field")
		r.Log.Error(err, "", "secret", secretName)
		return nil
	}

	if _, found := secret.Data["password"]; !found {
		err = errors.New("password not found in secret. The secret must be contained in 'password' field")
		r.Log.Error(err, "", "secret", secretName)
		return nil
	}

	envs = append(envs, corev1.EnvVar{
		Name: constants.EnvDBUser,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: ns.Spec.Database.Mysql.Secret.Name},
				Key:                  "user",
			}},
	})

	envs = append(envs, corev1.EnvVar{
		Name: constants.EnvDBPassword,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: ns.Spec.Database.Mysql.Secret.Name},
				Key:                  "password",
			},
		},
	})

	dbName := ns.Spec.Database.Mysql.DbName
	if dbName == "" {
		dbName = constants.DefaultDatabaseName
	}

	jdbcTemplate := ns.Spec.Database.Mysql.JdbcUrl
	if jdbcTemplate == "" {
		jdbcTemplate = string(constants.DefaultMysqlJDBCTemplate)
	}

	for ix, server := range ns.Spec.Database.Mysql.DbServer {
		envs = append(envs, corev1.EnvVar{
			Name:  constants.EnvDBUrlPrefix + strconv.Itoa(ix),
			Value: fmt.Sprintf(jdbcTemplate, server.DbHost, server.DbPort, dbName),
		})
	}

	return
}

func (r *NacosStandaloneReconciler) generateVolumesForDeployment(ns *nacosv1alpha1.NacosStandalone) (volumes []corev1.Volume) {
	if ns.Spec.Pvc != nil {
		volumes = append(volumes, corev1.Volume{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: r.generatePVCName(ns.Name),
					ReadOnly:  false,
				},
			},
		})
	}
	if ns.Spec.ApplicationConfig != nil && ns.Spec.ApplicationConfig.Name != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "conf",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: ns.Spec.ApplicationConfig.Name,
					},
				},
			},
		})
	}
	return
}

func (r *NacosStandaloneReconciler) generateVolumeMountsForDeployment(ns *nacosv1alpha1.NacosStandalone) (volumeMounts []corev1.VolumeMount) {
	volumeMounts = []corev1.VolumeMount{}
	if ns.Spec.Pvc != nil {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "data",
			ReadOnly:  false,
			MountPath: "/home/nacos/data",
			SubPath:   "data",
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "data",
			ReadOnly:  false,
			MountPath: "/home/nacos/logs",
			SubPath:   "logs",
		})
	}

	if ns.Spec.ApplicationConfig != nil && ns.Spec.ApplicationConfig.Name != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "conf",
			ReadOnly:  true,
			MountPath: "/home/nacos/conf",
		})
	}

	return
}

func (r *NacosStandaloneReconciler) checkVolumeChanged(dep *appsv1.Deployment, ns *nacosv1alpha1.NacosStandalone) (changed bool) {
	var dataExisted, confExisted bool
	for _, item := range dep.Spec.Template.Spec.Volumes {
		if item.Name == "data" {
			dataExisted = true
		} else if item.Name == "conf" {
			confExisted = true
		}
	}
	if (ns.Spec.Pvc != nil && !dataExisted) || (ns.Spec.Pvc == nil && dataExisted) {
		return true
	}

	if (ns.Spec.ApplicationConfig != nil && !confExisted) || (ns.Spec.ApplicationConfig == nil && confExisted) {
		return true
	}
	return
}

func (r *NacosStandaloneReconciler) processJvmOptionsForDeployment(ns *nacosv1alpha1.NacosStandalone, dep *appsv1.Deployment) (needUpdate bool) {
	var newEnv []corev1.EnvVar
	var existed *corev1.EnvVar
	for _, envVar := range dep.Spec.Template.Spec.Containers[0].Env {
		if envVar.Name != "JAVA_OPT" {
			newEnv = append(newEnv, envVar)
		} else {
			existed = &envVar
		}
	}
	if ns.Spec.JvmOptions != "" {
		if existed != nil && ns.Spec.JvmOptions == existed.Value {
			return false
		}
		dep.Spec.Template.Spec.Containers[0].Env = append(newEnv, corev1.EnvVar{Name: "JAVA_OPT", Value: ns.Spec.JvmOptions})
		needUpdate = true
	} else if existed != nil {
		needUpdate = true
		dep.Spec.Template.Spec.Containers[0].Env = newEnv
	}
	return
}
func labelsForNacosStandalone(ns *nacosv1alpha1.NacosStandalone) map[string]string {
	return map[string]string{constants.LabelNacosStandalone: ns.Name, constants.LabelApp: ns.Name}
}
