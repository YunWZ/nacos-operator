package util

import (
	corev1 "k8s.io/api/core/v1"
	"reflect"
	"sort"
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

/*func sort[T corev1.EnvVar](a, b T) bool {
	if a.Name != b.Name {
		return a.Name < b.Name
	}
}*/
