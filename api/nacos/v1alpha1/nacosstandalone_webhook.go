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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var nacosstandalonelog = logf.Log.WithName("nacosstandalone-resource")

func (r *NacosStandalone) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-nacos-yunweizhan-com-cn-v1alpha1-nacosstandalone,mutating=true,failurePolicy=fail,sideEffects=None,groups=nacos.yunweizhan.com.cn,resources=nacosstandalones,verbs=create;update,versions=v1alpha1,name=mnacosstandalone.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &NacosStandalone{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *NacosStandalone) Default() {
	nacosstandalonelog.Info("default", "name", r.Name)

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
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-nacos-yunweizhan-com-cn-v1alpha1-nacosstandalone,mutating=false,failurePolicy=fail,sideEffects=None,groups=nacos.yunweizhan.com.cn,resources=nacosstandalones,verbs=create;update,versions=v1alpha1,name=vnacosstandalone.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &NacosStandalone{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NacosStandalone) ValidateCreate() (admission.Warnings, error) {
	nacosstandalonelog.Info("validate create", "name", r.Name)

	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NacosStandalone) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	nacosstandalonelog.Info("validate update", "name", r.Name)

	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NacosStandalone) ValidateDelete() (admission.Warnings, error) {
	nacosstandalonelog.Info("validate delete", "name", r.Name)

	return nil, nil
}

func (r *NacosStandalone) validate() (warnings admission.Warnings, err error) {
	nacosclusterlog.Info("validate update", "name", r.Name)

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

	return
}
