package db

import (
	"context"
	"fmt"
	"path/filepath"

	apiv1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func getKubeClient() (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube/config"))
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// ensureNS 确保 namespace 存在
func ensureNS(pid, uid int64) error {
	ns := fmt.Sprintf("%d-%d", pid, uid)
	clientSet, err := getKubeClient()
	if err != nil {
		return err
	}
	namespace := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	}
	_, err = clientSet.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		// 排除 namespace 已经存在的情况
		if e, ok := err.(*kerrors.StatusError); !ok || e.ErrStatus.Code != 409 {
			return err
		}
	}
	return nil
}
