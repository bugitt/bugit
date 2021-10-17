package kube

import (
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Quota struct {
	CPU    string
	Memory string
}

func (quota Quota) convertResourceList() (v1.ResourceList, error) {
	quota.Memory = strings.ReplaceAll(quota.Memory, "g", "G")
	resourceList := make(map[v1.ResourceName]resource.Quantity)
	if len(quota.CPU) > 0 {
		quantity, err := resource.ParseQuantity(quota.CPU)
		if err != nil {
			return nil, err
		}
		resourceList[v1.ResourceCPU] = quantity
	}
	if len(quota.Memory) > 0 {
		quantity, err := resource.ParseQuantity(quota.Memory)
		if err != nil {
			return nil, err
		}
		resourceList[v1.ResourceMemory] = quantity
	}
	if len(resourceList) <= 0 {
		return nil, nil
	}
	return resourceList, nil
}

func (cli *Client) createQuotaForNS(quota Quota) error {
	resourceList, err := quota.convertResourceList()
	if err != nil {
		return err
	}
	if len(resourceList) <= 0 {
		return nil
	}
	_, err = cli.CoreV1().ResourceQuotas(cli.namespace).Create(
		cli.Ctx,
		&v1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name: cli.namespace,
			},
			Spec: v1.ResourceQuotaSpec{
				Hard: resourceList,
			},
		},
		metav1.CreateOptions{},
	)
	return err
}
