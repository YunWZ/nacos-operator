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

package v1alpha1

import (
	"errors"
	"github.com/YunWZ/nacos-operator/internal/controller/nacos/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	DefaultImage = "docker.io/nacos/nacos-server"
)

// log is for logging in this package.
var nacosclusterlog = logf.Log.WithName("nacoscluster-resource")

func (r *NacosCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-nacos-yunweizhan-com-cn-v1alpha1-nacoscluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=nacos.yunweizhan.com.cn,resources=nacosclusters,verbs=create;update,versions=v1alpha1,name=mnacoscluster.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &NacosCluster{}
var defaultReplicas int32 = 3

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *NacosCluster) Default() {
	nacosclusterlog.Info("default", "name", r.Name)

	if r.Spec.Replicas == nil {
		r.Spec.Replicas = &defaultReplicas
	}

	if r.Spec.Image == "" {
		r.Spec.Image = DefaultImage
	}

	if r.Spec.LivenessProbe == nil {
		r.Spec.LivenessProbe = NewDefaultLivenessProbe()
	}

	if r.Spec.ReadinessProbe == nil {
		r.Spec.ReadinessProbe = NewDefaultReadinessProbe()
	}

	if r.Spec.StartupProbe == nil {
		r.Spec.StartupProbe = NewDefaultStartupProbe()
	}

	if r.Spec.Domain == "" {
		r.Spec.Domain = "cluster.local"
	}
}

//+kubebuilder:webhook:path=/validate-nacos-yunweizhan-com-cn-v1alpha1-nacoscluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=nacos.yunweizhan.com.cn,resources=nacosclusters,verbs=create;update,versions=v1alpha1,name=vnacoscluster.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &NacosCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NacosCluster) ValidateCreate() (warnings admission.Warnings, err error) {
	nacosclusterlog.Info("validate create", "name", r.Name)

	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NacosCluster) ValidateUpdate(old runtime.Object) (warnings admission.Warnings, err error) {
	nacosclusterlog.Info("validate update", "name", r.Name)

	return r.validate()
}

func (r *NacosCluster) validate() (warnings admission.Warnings, err error) {
	nacosclusterlog.Info("validate update", "name", r.Name)

	if *r.Spec.Replicas < 0 || *r.Spec.Replicas == 1 || *r.Spec.Replicas == 2 {
		warnings = append(warnings, "replicas must be 0 or  greater than or equal to 3")
		err = errors.New("replicas must be 0 or  greater than or equal to 3")
	}

	if r.Spec.Image == "" {
		err = errors.New("image must be specified")
	}

	if r.Spec.LivenessProbe == nil {
		warnings = append(warnings, "livenessProbe is required")
		err = errors.New("livenessProbe is required")
	}

	if r.Spec.ReadinessProbe == nil {
		warnings = append(warnings, "readinessProbe is required")
		err = errors.New("readinessProbe is required")
	}

	if r.Spec.StartupProbe == nil {
		warnings = append(warnings, "startupProbe is required")
		err = errors.New("startupProbe is required")
	}
	if err = validateDatabase(r); err != nil {
		warnings = append(warnings, "startupProbe is required")
	}

	return
}

func validateDatabase(r *NacosCluster) error {
	if r.Spec.Database == nil {
		return errors.New("database must be specified")
	}

	if r.Spec.Database.Mysql != nil {
		return nil
	} else {
		return errors.New("database must contain at least one mysql")
	}

}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NacosCluster) ValidateDelete() (admission.Warnings, error) {
	nacosclusterlog.Info("validate delete", "name", r.Name)

	return nil, nil
}

func NewDefaultStartupProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: constants.DefaultNacosReadinessPath,
				Port: intstr.FromString(constants.DefaultNacosServerHttpPortName),
				//Host:   "127.0.0.1",
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		SuccessThreshold:    1,
		FailureThreshold:    50,
	}
}

func NewDefaultReadinessProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: constants.DefaultNacosReadinessPath,
				Port: intstr.FromString(constants.DefaultNacosServerHttpPortName),
				//Host:   "127.0.0.1",
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		TimeoutSeconds:      3,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
}

func NewDefaultLivenessProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: constants.DefaultNacosLivenessPath,
				Port: intstr.FromString(constants.DefaultNacosServerHttpPortName),
				//Host:   "127.0.0.1",
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		TimeoutSeconds:      10,
		SuccessThreshold:    1,
		FailureThreshold:    5,
	}
}
