package kube

import (
	apiv1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnsureNS 确保命名空间存在，不存在则创建 幂等
func (cli *Client) EnsureNS(quota Quota) error {
	namespace := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: cli.namespace},
	}
	_, err := cli.CoreV1().Namespaces().Get(cli.Ctx, cli.namespace, metav1.GetOptions{})
	if err == nil && !kerrors.IsNotFound(err) {
		return err
	}

	// 没有找到就create
	_, err = cli.CoreV1().Namespaces().Create(cli.Ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return nil
	}
	return cli.createQuotaForNS(quota)
}

func (cli *Client) DeleteNS() error {
	return cli.CoreV1().Namespaces().Delete(cli.Ctx, cli.namespace, metav1.DeleteOptions{})
}
