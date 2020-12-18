package util

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func DeepCopyLabels(src, tgt *metav1.ObjectMeta) {
	if src.Labels != nil {
		tgt.Labels = make(map[string]string, len(src.Labels))
		for key, val := range src.Labels {
			tgt.Labels[key] = val
		}
	}
}
