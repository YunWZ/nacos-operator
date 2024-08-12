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
	"fmt"
	nacosv1alpha1 "github.com/YunWZ/nacos-operator/apis/nacos/v1alpha1"
	"github.com/YunWZ/nacos-operator/internal/controller/nacos/constants"
	"github.com/YunWZ/nacos-operator/internal/util"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NacosClusterReconciler reconciles a NacosCluster object
type NacosClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the NacosCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
// +kubebuilder:rbac:groups=nacos.yunweizhan.com.cn,resources=nacosclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nacos.yunweizhan.com.cn,resources=nacosclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nacos.yunweizhan.com.cn,resources=nacosclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods;configmaps;secrets,verbs=get;list
// +kubebuilder:rbac:groups=core,resources=services;persistentvolumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *NacosClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	//_ = logs.FromContext(ctx)
	_ = r.Log.WithValues("nacos-cluster", req.NamespacedName)
	nc := &nacosv1alpha1.NacosCluster{}
	err := r.Get(ctx, req.NamespacedName, nc)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info("NacosCluster resource not found. Try to delete all referenced resources.", "NacosCluster", req.NamespacedName)
			err = r.deleteResourcesForNacosCluster(ctx, req)

			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		r.Log.Error(err, "Failed to get NacosCluster")
		return ctrl.Result{RequeueAfter: time.Second * 3}, client.IgnoreNotFound(err)
	}

	requeue, err := r.completeServiceForNacosCluster(nc)
	if err != nil {
		return util.NewResult(requeue), err
	} else if requeue {
		return util.NewResult(true), nil
	}

	requeue, err = r.completeProbeForNacosCluster(nc)
	if err != nil {
		return util.NewResult(requeue), err
	} else if requeue {
		return util.NewResult(true), nil
	}

	requeue, err = r.completePVCForNacosCluster(nc)
	if err != nil {
		return util.NewResult(requeue), err
	} else if requeue {
		return util.NewResult(true), nil
	}

	requeue, err = r.completeDeploymentsForNacosCluster(nc)
	if err != nil {
		return util.NewResult(requeue), err
	} else if requeue {
		return util.NewResult(true), nil
	}

	requeue, err = r.updateStatusForNacosCluster(nc)
	if err != nil {
		return util.NewResult(requeue), err
	} else if requeue {
		return util.NewResult(true), nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NacosClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nacosv1alpha1.NacosCluster{}).
		Complete(r)
}

func (r *NacosClusterReconciler) deleteResourcesForNacosCluster(ctx context.Context, req ctrl.Request) (err error) {
	r.Log.Info("Try to delete deployment resources.", "NacosCluster", req.NamespacedName)
	if err = r.deleteDeployment(req.NamespacedName); client.IgnoreNotFound(err) != nil {
		return err
	}

	r.Log.Info("Try to delete service resources.", "NacosCluster", req.NamespacedName)
	if err = r.deleteService(req.NamespacedName); client.IgnoreNotFound(err) != nil {
		return err
	}
	if err = r.deleteService(req.NamespacedName); client.IgnoreNotFound(err) != nil {
		return err
	}

	r.Log.Info("Try to delete pvc resources.", "NacosCluster", req.NamespacedName)
	err = r.deletePVC(req.NamespacedName)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

func (r *NacosClusterReconciler) deleteDeployment(name types.NamespacedName) (err error) {
	deleteOptions := []client.DeleteAllOfOption{
		client.InNamespace(name.Namespace),
		client.MatchingLabels(labelsForNacosCluster(name.Name))}
	err = r.DeleteAllOf(context.TODO(), &appsv1.Deployment{}, deleteOptions...)
	return err
}

func (r *NacosClusterReconciler) deleteService(name types.NamespacedName) (err error) {
	deleteOptions := []client.DeleteAllOfOption{
		client.InNamespace(name.Namespace),
		client.MatchingLabels(labelsForNacosCluster(name.Name))}
	err = r.DeleteAllOf(context.TODO(), &corev1.Service{}, deleteOptions...)
	return err
}

func (r *NacosClusterReconciler) deletePVC(name types.NamespacedName) (err error) {
	deleteOptions := []client.DeleteAllOfOption{
		client.InNamespace(name.Namespace),
		client.MatchingLabels(labelsForNacosCluster(name.Name))}
	return r.DeleteAllOf(context.TODO(), &corev1.PersistentVolumeClaim{}, deleteOptions...)
}

func (r *NacosClusterReconciler) completeProbeForNacosCluster(nc *nacosv1alpha1.NacosCluster) (requeue bool, err error) {
	r.Log.Info("Try to completeProbeForNacosCluster.", "NacosCluster", types.NamespacedName{Name: nc.Name, Namespace: nc.Namespace})
	needUpdate := false
	if nc.Spec.LivenessProbe == nil {
		needUpdate = true
		nc.Spec.LivenessProbe = nacosv1alpha1.NewDefaultLivenessProbe()
	}

	if nc.Spec.ReadinessProbe == nil {
		needUpdate = true
		nc.Spec.ReadinessProbe = nacosv1alpha1.NewDefaultReadinessProbe()
	}

	if nc.Spec.StartupProbe == nil {
		needUpdate = true
		nc.Spec.StartupProbe = nacosv1alpha1.NewDefaultStartupProbe()
	}

	if needUpdate {
		r.Log.Info("Update NacosCluster with default probes.", "NacosCluster", types.NamespacedName{Name: nc.Name, Namespace: nc.Namespace})
		err := r.Update(context.TODO(), nc)
		if err != nil && apierrors.IsNotFound(err) {
			return false, err
		} else if err != nil {
			return true, err
		}
	}

	return needUpdate, nil
}

func (r *NacosClusterReconciler) completePVCForNacosCluster(nc *nacosv1alpha1.NacosCluster) (requeue bool, err error) {
	pvcs, pvcExists, err := r.checkPVCExist(nc)
	if err != nil {
		r.Log.Error(err, "Failed to check if PVC exists")
		return true, err
	}

	if nc.Spec.Pvc == nil {
		if !pvcExists {
			return false, nil
		} else {
			// TODO: Do nothing. Reserve old PersistentVolumeClaims and PersistentVolumes.
			return false, nil
		}
	} else if !pvcExists { // Needed to create pvc.
		pvcs = r.persistentVolumeClaimsForNacosCluster(nc)
		for _, pvc := range pvcs {
			if err = r.Create(context.TODO(), &pvc); client.IgnoreAlreadyExists(err) != nil {
				r.Log.Error(err, "Create PVC failed.",
					"Current PVC", pvc.Namespace+"/"+pvc.Name)
				return true, err
			}
		}
		return true, nil
	} else {
		sort.Slice(pvcs, func(i, j int) bool {
			return pvcs[i].Name < pvcs[j].Name
		})

		for i := int32(0); i < *nc.Spec.Replicas; i++ {
			pvcName := generatePVCNameForNacosCluster(nc.Name, i)
			if pvcName == pvcs[i].Name {
				continue
			}
			pvc := r.persistentVolumeClaimForNacosCluster(nc, pvcName)
			if err = r.Create(context.TODO(), &pvc); client.IgnoreAlreadyExists(err) != nil {
				return true, err
			}
		}
	}

	return false, nil
}

func (r *NacosClusterReconciler) checkPVCExist(ns *nacosv1alpha1.NacosCluster) (pvc []corev1.PersistentVolumeClaim, existed bool, err error) {
	pvcList := &corev1.PersistentVolumeClaimList{}
	err = r.List(context.TODO(), pvcList, client.InNamespace(ns.Namespace),
		client.MatchingLabels(labelsForNacosCluster(ns.Name)))
	if err == nil {
		return pvcList.Items, len(pvcList.Items) != 0, nil
	}

	return nil, false, client.IgnoreNotFound(err)
}

func (r *NacosClusterReconciler) persistentVolumeClaimsForNacosCluster(nc *nacosv1alpha1.NacosCluster) (pvcs []corev1.PersistentVolumeClaim) {
	var pvcList []corev1.PersistentVolumeClaim
	for i := int32(0); i < *nc.Spec.Replicas; i++ {
		pvcList = append(pvcList, r.persistentVolumeClaimForNacosCluster(nc, generatePVCNameForNacosCluster(nc.Name, i)))
	}

	return pvcList
}

func (r *NacosClusterReconciler) persistentVolumeClaimForNacosCluster(nc *nacosv1alpha1.NacosCluster, name string) (pvc corev1.PersistentVolumeClaim) {
	return corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nc.Namespace,
		},
		Spec: *nc.Spec.Pvc,
	}
}

func (r *NacosClusterReconciler) completeDeploymentsForNacosCluster(nc *nacosv1alpha1.NacosCluster) (requeue bool, err error) {
	if nc.Spec.Replicas == nil || *nc.Spec.Replicas == zero {
		err := r.DeleteAllOf(context.TODO(), &appsv1.Deployment{}, client.InNamespace(nc.Namespace), client.MatchingLabels(labelsForNacosCluster(nc.Name)))
		if err != nil {
			return requeue, err
		}
		return false, nil
	}

	deps := &appsv1.DeploymentList{}
	err = r.List(context.TODO(), deps, client.MatchingLabels(labelsForNacosCluster(nc.Name)), client.InNamespace(nc.Namespace))
	if err != nil && !apierrors.IsNotFound(err) {
		return true, err
	}

	oldDeps := deps.Items
	sort.Slice(oldDeps, func(i, j int) bool { return oldDeps[i].Name < oldDeps[j].Name })
	depMap := listToMap(deps.Items...)
	memberList := generateMemberList(nc)
	for i := zero; i < *nc.Spec.Replicas; i++ {
		curDepName := generateDeploymentNameForNacosCluster(nc.Name, i)

		if curDep, exist := depMap[curDepName]; !exist {
			err = r.Create(context.TODO(), r.generateDeploymentForNacosCluster(nc, curDepName, i, memberList))
			if err != nil && !apierrors.IsAlreadyExists(err) {
				return true, err
			}
		} else {
			requeue, err = r.completeDeploymentForNacosCluster(nc, curDep, i, memberList)
			if requeue {
				return requeue, err
			} else {
				delete(depMap, curDepName)
			}
		}
	}

	//delete others deployment
	for _, v := range depMap {
		err = r.Delete(context.TODO(), v)
		if err != nil && !apierrors.IsNotFound(err) {
			r.Log.Error(err, "Delete Deployment failed.")
			return true, err
		}
	}

	return requeue, err
}

func listToMap(items ...appsv1.Deployment) map[string]*appsv1.Deployment {
	depMap := make(map[string]*appsv1.Deployment, len(items))
	for index, dep := range items {
		depMap[dep.Name] = &items[index]
	}

	return depMap
}

func (r *NacosClusterReconciler) generateDeploymentForNacosCluster(nc *nacosv1alpha1.NacosCluster, depName string, index int32, memberList string) *appsv1.Deployment {
	lbs := labelsForDeploymemt(nc.Name, depName)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      depName,
			Namespace: nc.Namespace,
			Labels:    lbs,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &one,
			Selector: &metav1.LabelSelector{
				MatchLabels: lbs,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: lbs},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "nacos-server",
							Image:           nacosv1alpha1.DefaultImage,
							ImagePullPolicy: nc.Spec.ImagePullPolicy,
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
								{Name: constants.EnvMode, Value: constants.NacosModeCluster},
								{Name: constants.EnvPreferHostName, Value: constants.EnvPreferHostNameValueForHostname},
							},
							LivenessProbe:  nc.Spec.LivenessProbe,
							ReadinessProbe: nc.Spec.ReadinessProbe,
							StartupProbe:   nc.Spec.StartupProbe,
							Resources:      nc.Spec.Resources,
						},
					},
					Hostname:  depName,
					Subdomain: generateHeadlessServiceNameForNacosCluster(nc.Name),
					//SetHostnameAsFQDN: &bool_true,
					Affinity:         nc.Spec.Affinity,
					ImagePullSecrets: nc.Spec.ImagePullSecrets,
					Tolerations:      nc.Spec.Tolerations,
				},
			},
		},
	}

	if nc.Spec.Affinity == nil {
		dep.Spec.Template.Spec.Affinity = generateAffinityForDeployment(nc)
	}

	if nc.Spec.Image != "" {
		dep.Spec.Template.Spec.Containers[0].Image = nc.Spec.Image
	}

	dep.Spec.Template.Spec.Volumes = r.generateVolumesForDeployment(nc, index)

	dep.Spec.Template.Spec.Containers[0].VolumeMounts = r.generateVolumeMountsForDeployment(nc)

	//var needUpdate bool
	_ = util.ProcessDatabaseEnvForDeployment(nc.Namespace, nc.Name, nc.Spec.Database, dep)

	_ = util.ProcessJvmOptionsEnvForDeployment(nc.Spec.JvmOptions, dep)

	_ = r.processMemberListEnvForDeployment(nc, dep, memberList)
	return dep
}

func (r *NacosClusterReconciler) generateVolumesForDeployment(nc *nacosv1alpha1.NacosCluster, index int32) (volumes []corev1.Volume) {
	if nc.Spec.Pvc != nil {
		volumes = append(volumes, corev1.Volume{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: generatePVCNameForNacosCluster(nc.Name, index),
					ReadOnly:  false,
				},
			},
		})
	}
	if nc.Spec.ApplicationConfig != nil && nc.Spec.ApplicationConfig.Name != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "conf",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: nc.Spec.ApplicationConfig.Name,
					},
				},
			},
		})
	}
	return
}

func (r *NacosClusterReconciler) completeServiceForNacosCluster(nc *nacosv1alpha1.NacosCluster) (requeue bool, err error) {
	svcList := &corev1.ServiceList{}
	lbs := labelsForNacosCluster(nc.Name)
	err = r.List(context.TODO(), svcList, client.InNamespace(nc.Namespace), client.MatchingLabels(lbs))
	if err != nil {
		r.Log.Error(err, "Failed to list services")
		return true, err
	}
	var normalSvc, headlessSvc *corev1.Service
	headlessSvcName := generateHeadlessServiceNameForNacosCluster(nc.Name)
	for index, svc := range svcList.Items {
		if svc.Name == nc.Name {
			normalSvc = &svcList.Items[index]

		} else if svc.Name == headlessSvcName {
			headlessSvc = &svcList.Items[index]
		}
	}

	if normalSvc == nil {
		err = r.Create(context.TODO(), util.GenerateNormalService(nc.Namespace, nc.Name, nc.Spec.Service.Type, lbs))
		if err == nil {
			requeue = true
		} else if err = client.IgnoreAlreadyExists(err); err != nil {
			r.Log.Error(err, "Failed to create normal service")
			return true, err
		}
	} else if (nc.Spec.Service.Type == "" && normalSvc.Spec.Type != corev1.ServiceTypeClusterIP) || normalSvc.Spec.Type != nc.Spec.Service.Type {
		normalSvc.Spec.Type = nc.Spec.Service.Type
		err = r.Update(context.TODO(), normalSvc)
		if err == nil {
			requeue = true
		} else {
			r.Log.Error(err, "Failed to update normal service")
			return true, err
		}
	}

	if headlessSvc == nil {
		err = r.Create(context.TODO(), &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      headlessSvcName,
				Namespace: nc.Namespace,
				Labels:    lbs,
			},
			Spec: corev1.ServiceSpec{
				ClusterIP:                corev1.ClusterIPNone,
				PublishNotReadyAddresses: true,
				Selector:                 lbs,
				Type:                     corev1.ServiceTypeClusterIP,
			},
		})
		if err != nil && !apierrors.IsAlreadyExists(err) {
			r.Log.Error(err, "Failed to create service")
			return true, err
		}
		requeue = true
	}

	return requeue, nil
}

func (r *NacosClusterReconciler) processMemberListEnvForDeployment(nc *nacosv1alpha1.NacosCluster, dep *appsv1.Deployment, memberList string) (needUpdate bool) {
	oldEnv := dep.Spec.Template.Spec.Containers[0].Env
	newEnv := []corev1.EnvVar{}
	var oldMemberListEnv corev1.EnvVar
	for _, env := range oldEnv {
		if env.Name == constants.EnvNacosServers {
			oldMemberListEnv = env
		} else {
			newEnv = append(newEnv, env)
		}
	}

	if reflect.DeepEqual(oldMemberListEnv, corev1.EnvVar{}) || oldMemberListEnv.Value != memberList {
		needUpdate = true
		newEnv = append(newEnv, corev1.EnvVar{
			Name:  constants.EnvNacosServers,
			Value: memberList,
		})
		dep.Spec.Template.Spec.Containers[0].Env = newEnv
	}

	return
}

func (r *NacosClusterReconciler) updateStatusForNacosCluster(nc *nacosv1alpha1.NacosCluster) (bool, error) {
	// TODO
	return false, nil
}

func (r *NacosClusterReconciler) completeDeploymentForNacosCluster(nc *nacosv1alpha1.NacosCluster, found *appsv1.Deployment, index int32, memberList string) (changed bool, err error) {
	changed = util.ProcessDatabaseEnvForDeployment(nc.Namespace, nc.Name, nc.Spec.Database, found)
	changed = r.processMemberListEnvForDeployment(nc, found, memberList) || changed

	if !reflect.DeepEqual(found.Spec.Template.Spec.Containers[0].Image, nc.Spec.Image) {
		changed = true
		found.Spec.Template.Spec.Containers[0].Image = nc.Spec.Image
	}
	if !reflect.DeepEqual(found.Spec.Template.Spec.Containers[0].ImagePullPolicy, nc.Spec.ImagePullPolicy) {
		changed = true
		found.Spec.Template.Spec.Containers[0].ImagePullPolicy = nc.Spec.ImagePullPolicy
	}
	if !reflect.DeepEqual(found.Spec.Template.Spec.ImagePullSecrets, nc.Spec.ImagePullSecrets) {
		changed = true
		found.Spec.Template.Spec.ImagePullSecrets = nc.Spec.ImagePullSecrets
	}

	if r.checkVolumeChanged(found, nc) {
		changed = true
		volumes := r.generateVolumesForDeployment(nc, index)

		found.Spec.Template.Spec.Volumes = volumes
		mounts := r.generateVolumeMountsForDeployment(nc)

		found.Spec.Template.Spec.Containers[0].VolumeMounts = mounts
	}

	changed = util.ProcessJvmOptionsEnvForDeployment(nc.Spec.JvmOptions, found) || changed

	if *found.Spec.Replicas != one {
		changed = true
		found.Spec.Replicas = &one
	}

	if nc.Spec.ReadinessProbe != nil && !reflect.DeepEqual(nc.Spec.ReadinessProbe, found.Spec.Template.Spec.Containers[0].ReadinessProbe) {
		changed = true
		found.Spec.Template.Spec.Containers[0].ReadinessProbe = nc.Spec.ReadinessProbe
	}
	if nc.Spec.LivenessProbe != nil && !reflect.DeepEqual(nc.Spec.LivenessProbe, found.Spec.Template.Spec.Containers[0].LivenessProbe) {
		changed = true
		found.Spec.Template.Spec.Containers[0].LivenessProbe = nc.Spec.LivenessProbe
	}
	if nc.Spec.StartupProbe != nil && !reflect.DeepEqual(nc.Spec.StartupProbe, found.Spec.Template.Spec.Containers[0].StartupProbe) {
		changed = true
		found.Spec.Template.Spec.Containers[0].StartupProbe = nc.Spec.StartupProbe
	}

	if !reflect.DeepEqual(nc.Spec.Resources, found.Spec.Template.Spec.Containers[0].Resources) {
		changed = true
		found.Spec.Template.Spec.Containers[0].Resources = nc.Spec.Resources
	}

	if changed {
		if err = r.Update(context.TODO(), found); err != nil {
			r.Log.Error(err, "Failed to update Deployment, Deployment.Namespace: %s, Deployment.Name: %s", found.Namespace, found.Name)
			//return true, err
		}
		// Spec updated - return and requeue
		//return true, nil
	}

	return changed, err
}

func (r *NacosClusterReconciler) checkVolumeChanged(found *appsv1.Deployment, nc *nacosv1alpha1.NacosCluster) bool {
	return util.CheckVolumeChanged(found, nc.Spec.Pvc, nc.Spec.ApplicationConfig)
}

func (r *NacosClusterReconciler) generateVolumeMountsForDeployment(nc *nacosv1alpha1.NacosCluster) []corev1.VolumeMount {
	return util.GenerateVolumeMountsForDeployment(nc.Spec.Pvc, nc.Spec.ApplicationConfig)
}

func generateAffinityForDeployment(nc *nacosv1alpha1.NacosCluster) *corev1.Affinity {
	aff := &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 1,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: labelsForNacosCluster(nc.Name),
						},
						Namespaces:  []string{nc.Namespace},
						TopologyKey: constants.NodeLabelHostName,
					},
				},
			},
		},
	}
	return aff
}

func generateMemberList(nc *nacosv1alpha1.NacosCluster) string {
	var memberList []string
	suffix := "." + generateHeadlessServiceNameForNacosCluster(nc.Name) + "." + nc.Namespace + ".svc." + nc.Spec.Domain
	for i := zero; i < *nc.Spec.Replicas; i++ {
		memberList = append(memberList, generateDeploymentNameForNacosCluster(nc.Name, i)+suffix)
	}
	return strings.Join(memberList, ",")
}
func generateDeploymentNameForNacosCluster(name string, i int32) string {
	return name + "-" + fmt.Sprint(i)
}
func generatePVCNameForNacosCluster(name string, i int32) string {
	return name + "-data-" + fmt.Sprint(i)
}

func generateHeadlessServiceNameForNacosCluster(name string) string {
	return name + "-headless"
}

func labelsForNacosCluster(name string) map[string]string {
	return map[string]string{constants.LabelNacosCluster: name}
}

func labelsForDeploymemt(ncName, depName string) map[string]string {
	depLabels := labelsForNacosCluster(ncName)
	depLabels[constants.LabelApp] = depName
	return depLabels
}
