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

type DeploymentController struct {
	Client *kubernetes.Clientset
	DCli   v1.DeploymentInterface
	D      *appsv1.Deployment
}

func NewDeploymentController(container *apiv1.Container, client *kubernetes.Clientset, opt *DeployOpt) DeploymentController {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   opt.Name,
			Labels: opt.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &opt.ReplicaNum,
			Selector: &metav1.LabelSelector{
				MatchLabels: opt.Labels,
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
	return DeploymentController{
		Client: client,
		D:      deployment,
		DCli:   client.AppsV1().Deployments(opt.Namespace),
	}
}

func (d DeploymentController) DeployOrUpdate(ctx context.Context) (err error) {
	// 看看之前部署的deployment还存不存在
	exists, err := d.Exists(ctx)
	if err != nil {
		return
	}
	if !exists {
		// create
		result, err := d.DCli.Create(ctx, d.D, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		d.D = result
		return err
	}
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, err := d.DCli.Update(ctx, d.D, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		d.D = result
		return nil
	})
	return retryErr
}

func (d DeploymentController) GetPods(ctx context.Context) ([]apiv1.Pod, error) {
	labels, err := metav1.LabelSelectorAsMap(d.D.Spec.Selector)
	if err != nil {
		return nil, err
	}
	return listPodsByLabels(ctx, d.Client, labels, d.D.Namespace)
}

func (d DeploymentController) Delete(ctx context.Context) error {
	deletePolicy := metav1.DeletePropagationForeground
	return d.DCli.Delete(ctx, d.D.Name, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
}

func (d DeploymentController) Exists(ctx context.Context) (bool, error) {
	_, err := d.DCli.Get(ctx, d.D.Name, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}
