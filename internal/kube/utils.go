package kube

import (
	"encoding/base64"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func _(i int32) *int32 {
	return &i
}

func getListOpt(labels map[string]string) (*metav1.ListOptions, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: labels,
	})
	if err != nil {
		return nil, err
	}
	return &metav1.ListOptions{
		LabelSelector: selector.String(),
	}, nil
}

func (p PullPolicy) official() v1.PullPolicy {
	return v1.PullPolicy(p)
}

func base64EncodeString(src []byte) string {
	return base64.StdEncoding.EncodeToString(src)
}
