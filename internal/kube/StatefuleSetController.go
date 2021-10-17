package kube

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/util/retry"
)

type StatefulSetController struct {
	Client *kubernetes.Clientset
	SCli   v1.StatefulSetInterface
	S      *appsv1.StatefulSet
}

func NewStatefulSetController(container *apiv1.Container, client *kubernetes.Clientset, opt *DeployOpt) StatefulSetController {
	deployment := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:   opt.Name,
			Labels: opt.Labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &opt.ReplicaNum,
			Selector: &metav1.LabelSelector{
				MatchLabels: opt.PodLabels,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: opt.PodLabels,
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						*container,
					},
					ImagePullSecrets: opt.ImagePullSecrets,
				},
			},
		},
	}
	return StatefulSetController{
		Client: client,
		S:      deployment,
		SCli:   client.AppsV1().StatefulSets(opt.Namespace),
	}
}

func (s StatefulSetController) DeployOrUpdate(ctx context.Context) (err error) {
	// 看看之前部署的deployment还存不存在
	exists, err := s.Exists(ctx)
	if err != nil {
		return
	}
	if !exists {
		// create
		result, err := s.SCli.Create(ctx, s.S, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		s.S = result
		return err
	}
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, err := s.SCli.Update(ctx, s.S, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		s.S = result
		return nil
	})
	return retryErr
}

func (s StatefulSetController) GetPods(ctx context.Context) ([]apiv1.Pod, error) {
	labels, err := metav1.LabelSelectorAsMap(s.S.Spec.Selector)
	if err != nil {
		return nil, err
	}
	return listPodsByLabels(ctx, s.Client, labels, s.S.Namespace)
}

func (s StatefulSetController) Delete(ctx context.Context) error {
	deletePolicy := metav1.DeletePropagationForeground
	return s.SCli.Delete(ctx, s.S.Name, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
}

func (s StatefulSetController) Exists(ctx context.Context) (bool, error) {
	_, err := s.SCli.Get(ctx, s.S.Name, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}
