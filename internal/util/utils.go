package util

import (
	"github.com/YunWZ/nacos-operator/internal/controller/nacos/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sort"
	"time"
)

func EnvVarsEqual(a, b []corev1.EnvVar) bool {
	if len(a) != len(b) {
		return false
	}

	sort.Slice(a, func(i, j int) bool { return a[i].Name < a[j].Name })
	sort.Slice(b, func(i, j int) bool { return b[i].Name < b[j].Name })
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
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

func NewResult(requeue bool) ctrl.Result {
	return ctrl.Result{Requeue: requeue, RequeueAfter: time.Second * 3}
}
