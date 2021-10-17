package kube

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	watch2 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/watch"
)

type PodDeployOpt struct {
	Labels               map[string]string
	ExtraLabels          map[string]string
	ReplicaNum           int32
	Stateful             bool
	Duration             time.Duration
	DockerRegistrySecret string
	Spec                 PodSpec
}

type PodSpec struct {
	Name       string
	ImageTag   string
	Envs       map[string]string
	Ports      []Port
	WorkDir    string
	Cmd        Cmd
	labels     map[string]string
	PullPolicy PullPolicy
	Quota
}

type PodExportNodePortOpt struct {
	BaseServiceOpt
}

func (opt *PodDeployOpt) fix() {
	if opt.Duration <= 0 {
		opt.Duration = DefaultDuration
	}

	// generate pod labels
	podLabels := make(map[string]string)
	for k, v := range opt.Labels {
		podLabels[k] = v
	}
	for k, v := range opt.ExtraLabels {
		podLabels[k] = v
	}
	opt.Spec.labels = podLabels

	if len(opt.Spec.PullPolicy) <= 0 {
		opt.Spec.PullPolicy = PullIfNotPresent
	}
}

type ErrPodDeploy struct {
	Msg string
}

func (err *ErrPodDeploy) Error() string {
	return err.Msg
}

var ErrPodDeployTimeout = &ErrPodDeploy{Msg: "timeout"}

func (cli *Client) PodDeploy(ctx context.Context, opt *PodDeployOpt) (err error) {
	opt.fix()
	ctx, cancel := context.WithTimeout(ctx, opt.Duration)
	defer cancel()

	container, err := getContainer(opt.Spec)
	if err != nil {
		return
	}

	errCh := make(chan error)

	go func() {
		var err error
		defer func() {
			errCh <- err
		}()

		deployOpt := &DeployOpt{
			Name:       opt.Spec.Name,
			Labels:     opt.Labels,
			ReplicaNum: opt.ReplicaNum,
			Namespace:  cli.namespace,
			PodLabels:  opt.Spec.labels,
		}
		if len(opt.DockerRegistrySecret) > 0 {
			deployOpt.ImagePullSecrets = append(deployOpt.ImagePullSecrets, v1.LocalObjectReference{Name: opt.DockerRegistrySecret})
		}

		var controller PodController
		if opt.Stateful {
			controller = NewStatefulSetController(container, cli.Clientset, deployOpt)
		} else {
			controller = NewDeploymentController(container, cli.Clientset, deployOpt)
		}

		// deploy
		err = controller.DeployOrUpdate(ctx)
		if err != nil {
			return
		}
		// wait pod for done
		//errMsg, err = waitPodsForRunning(ctx, clientSet, opt.Namespace, opt.Spec.labels)
		return
	}()

	select {
	case <-ctx.Done():
		err = ErrPodDeployTimeout
	case err = <-errCh:
	}
	return
}

func (cli *Client) PodExportNodePort(ctx context.Context, opt *PodExportNodePortOpt) (ports []SvcPort, err error) {
	svc, err := cli.CreateOrReplaceService(ctx, &ServiceOpt{
		BaseServiceOpt: opt.BaseServiceOpt,
		Type:           v1.ServiceTypeNodePort,
	})
	if err != nil {
		return
	}
	for _, port := range svc.Spec.Ports {
		ports = append(ports, SvcPort{
			Port: Port{
				Name:     port.Name,
				Protocol: string(port.Protocol),
				Port:     port.TargetPort.IntVal,
			},
			ExportPort: port.NodePort,
		})
	}
	return
}

func (cli *Client) GetPodsByLabels(ctx context.Context, labels map[string]string) ([]v1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: labels,
	})
	if err != nil {
		return nil, err
	}
	return listPodsBySelector(ctx, cli.Clientset, selector, cli.namespace)
}

func waitPodForReady(ctx context.Context, client *kubernetes.Clientset, namespace string, labels map[string]string) (errMsg string, err error) {
	listOpt, err := getListOpt(labels)
	if err != nil {
		return err.Error(), err
	}
	condition := func(event watch2.Event) (bool, error) {
		pod, ok := event.Object.(*v1.Pod)
		if !ok {
			return false, nil
		}
		switch pod.Status.Phase {
		case v1.PodRunning, v1.PodSucceeded:
			return true, nil
		case v1.PodFailed:
			errMsg = pod.Status.Message
			return false, nil
		default:
			return false, nil
		}
	}
	client.CoreV1().Pods(namespace).GetLogs(namespace, &v1.PodLogOptions{
		TypeMeta:                     metav1.TypeMeta{},
		Container:                    "",
		Follow:                       false,
		Previous:                     false,
		SinceSeconds:                 nil,
		SinceTime:                    nil,
		Timestamps:                   false,
		TailLines:                    nil,
		LimitBytes:                   nil,
		InsecureSkipTLSVerifyBackend: false,
	})
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return client.CoreV1().Pods(namespace).List(ctx, *listOpt)
		},
		WatchFunc: func(options metav1.ListOptions) (watch2.Interface, error) {
			return client.CoreV1().Pods(namespace).Watch(ctx, *listOpt)
		},
	}

	watcher, err := watch.NewRetryWatcher("1", lw)
	if err != nil {
		return
	}
	ch := watcher.ResultChan()
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				err = watch.ErrWatchClosed
				return
			}
			if ok, err := condition(event); err != nil {
				return err.Error(), err
			} else if ok {
				return errMsg, nil
			}
		case <-ctx.Done():
			err = ErrPodDeployTimeout
			return
		}
	}
	//_, err = watch.Until(ctx, "1", lw, condition)
	//return
}

func getContainer(spec PodSpec) (*v1.Container, error) {
	container := &v1.Container{
		Name:            spec.Name,
		Image:           spec.ImageTag,
		ImagePullPolicy: spec.PullPolicy.official(),
	}

	// 端口
	container.Ports = getContainerPorts(spec.Ports)

	// 环境变量
	var envs []v1.EnvVar
	for k, v := range spec.Envs {
		envs = append(envs, v1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	container.Env = envs

	// workingDir
	if len(spec.WorkDir) > 0 {
		container.WorkingDir = spec.WorkDir
	}

	// command
	if len(spec.Cmd.Command) > 0 {
		container.Command = spec.Cmd.Command
	}
	if len(spec.Cmd.Args) > 0 {
		container.Args = spec.Cmd.Args
	}

	// quota
	resourceList, err := spec.Quota.convertResourceList()
	if err != nil {
		return nil, err
	}
	if len(resourceList) > 0 {
		container.Resources.Limits = resourceList
		container.Resources.Requests = resourceList
	}

	// TODO: 持久化
	return container, nil
}

func getContainerPorts(dbPorts []Port) []v1.ContainerPort {
	ports := make([]v1.ContainerPort, 0, len(dbPorts))
	for i, port := range dbPorts {
		p := v1.ContainerPort{}

		// 处理端口名称
		if port.Name != "" {
			p.Name = port.Name
		} else {
			p.Name = fmt.Sprintf("port-%d", i)
		}

		// 处理端口协议，默认为tcp
		var protocol v1.Protocol
		switch strings.ToLower(port.Protocol) {
		case "udp":
			protocol = v1.ProtocolUDP
		case "sctp":
			protocol = v1.ProtocolSCTP
		default:
			protocol = v1.ProtocolTCP
		}
		p.Protocol = protocol

		//  处理端口号
		p.ContainerPort = port.Port

		ports = append(ports, p)
	}
	return ports
}

func listPodsByLabels(ctx context.Context, client *kubernetes.Clientset, labels map[string]string, namespace string) ([]v1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: labels,
	})
	if err != nil {
		return nil, err
	}
	return listPodsBySelector(ctx, client, selector, namespace)
}

func listPodsBySelector(ctx context.Context, client *kubernetes.Clientset, selector labels.Selector, namespace string) ([]v1.Pod, error) {
	podList, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}
