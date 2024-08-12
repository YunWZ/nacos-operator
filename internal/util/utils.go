package util

import (
	"fmt"
	nacosv1alpha1 "github.com/YunWZ/nacos-operator/apis/nacos/v1alpha1"
	"github.com/YunWZ/nacos-operator/internal/controller/nacos/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sort"
	"strconv"
	"strings"
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

func NewResult(requeue bool) ctrl.Result {
	return ctrl.Result{Requeue: requeue, RequeueAfter: time.Second * 3}
}

func ProcessDatabaseEnvForDeployment(namespace, name string, database *nacosv1alpha1.DatabaseSource, dep *appsv1.Deployment) (needUpdate bool) {
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
		if database == nil {
			return false
		}

		newDbEnv := GenerateDataBaseEnvForMysql(namespace, name, database)
		newEnv = append(newEnv, newDbEnv...)
		needUpdate = true
	} else {
		if database == nil {
			needUpdate = true
		} else {
			newDbEnv := GenerateDataBaseEnvForMysql(namespace, name, database)
			if !EnvVarsEqual(oldDbEnv, newDbEnv) {
				newEnv = append(newEnv, newDbEnv...)
				needUpdate = true
			}
		}
	}

	/*	if existed && database != nil {
		if database.Mysql != nil {
			newDbEnv := GenerateDataBaseEnvForMysql(ns)
			if !util.EnvVarsEqual(oldDbEnv, newDbEnv) {
				newEnv = append(newEnv, newDbEnv...)
			}
		} else {
			r.Log.Info("Unsupported database type, check your database spec")
			return false
		}
	}*/

	if needUpdate {
		dep.Spec.Template.Spec.Containers[0].Env = newEnv
	}

	return needUpdate
}

func GenerateDataBaseEnvForMysql(namespace, name string, database *nacosv1alpha1.DatabaseSource) (envs []corev1.EnvVar) {
	if database == nil {
		return
	}
	envs = append(envs, corev1.EnvVar{
		Name:  constants.EnvDatabasePlatform,
		Value: "mysql",
	})
	envs = append(envs, corev1.EnvVar{
		Name:  constants.EnvDBNum,
		Value: strconv.Itoa(len(database.Mysql.DbServer)),
	})

	envs = append(envs, corev1.EnvVar{
		Name: constants.EnvDBUser,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: database.Mysql.Secret.Name},
				Key:                  "user",
			}},
	})

	envs = append(envs, corev1.EnvVar{
		Name: constants.EnvDBPassword,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: database.Mysql.Secret.Name},
				Key:                  "password",
			},
		},
	})

	dbName := database.Mysql.DbName
	if dbName == "" {
		dbName = constants.DefaultDatabaseName
	}

	jdbcTemplate := database.Mysql.JdbcUrl
	if jdbcTemplate == "" {
		jdbcTemplate = string(constants.DefaultMysqlJDBCTemplate)
	}

	for ix, server := range database.Mysql.DbServer {
		envs = append(envs, corev1.EnvVar{
			Name:  constants.EnvDBUrlPrefix + strconv.Itoa(ix),
			Value: fmt.Sprintf(jdbcTemplate, server.DbHost, server.DbPort, dbName),
		})
	}

	return
}

func GenerateVolumeMountsForDeployment(pvc *corev1.PersistentVolumeClaimSpec, applicationConfig *corev1.LocalObjectReference) (volumeMounts []corev1.VolumeMount) {
	volumeMounts = []corev1.VolumeMount{}
	if pvc != nil {
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

	if applicationConfig != nil && applicationConfig.Name != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "conf",
			ReadOnly:  true,
			MountPath: "/home/nacos/conf",
		})
	}

	return
}

func ProcessJvmOptionsEnvForDeployment(jvmOptions string, dep *appsv1.Deployment) (needUpdate bool) {
	var newEnv []corev1.EnvVar
	var existed *corev1.EnvVar
	for _, envVar := range dep.Spec.Template.Spec.Containers[0].Env {
		if envVar.Name != "JAVA_OPT" {
			newEnv = append(newEnv, envVar)
		} else {
			existed = &envVar
		}
	}
	if jvmOptions != "" {
		if existed != nil && jvmOptions == existed.Value {
			return false
		}
		dep.Spec.Template.Spec.Containers[0].Env = append(newEnv, corev1.EnvVar{Name: "JAVA_OPT", Value: jvmOptions})
		needUpdate = true
	} else if existed != nil {
		needUpdate = true
		dep.Spec.Template.Spec.Containers[0].Env = newEnv
	}
	return
}

func CheckVolumeChanged(dep *appsv1.Deployment, pvc *corev1.PersistentVolumeClaimSpec, config *corev1.LocalObjectReference) (hasChanged bool) {
	var dataExisted, confExisted bool
	for _, item := range dep.Spec.Template.Spec.Volumes {
		if item.Name == "data" {
			dataExisted = true
		} else if item.Name == "conf" {
			confExisted = true
		}
	}
	if (pvc != nil && !dataExisted) || (pvc == nil && dataExisted) {
		return true
	}

	if (config != nil && !confExisted) || (config == nil && confExisted) {
		return true
	}
	return
}

func GenerateNormalService(namespace, name string, svcType corev1.ServiceType, lbs map[string]string) (svc *corev1.Service) {
	svc = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: lbs},
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
			Selector: lbs,
			Type:     svcType,
		},
	}

	return
}
