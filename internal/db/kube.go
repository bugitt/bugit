package db

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type DeployContext struct {
	*CIContext
	*DeployResult

	labels    map[string]string
	svcLabels map[string]string
	container *apiv1.Container
	repNum    int32
	stateful  bool
}

// nextIP 获取这次应该部署的到哪个IP上
var nextIP = func() func() string {
	i := -1
	return func() string {
		i = (i + 1) % len(conf.Devops.KubeIP)
		return conf.Devops.KubeIP[i]
	}
}()

var clientSet *kubernetes.Clientset

func init() {
	clientSet, _ = getKubeClient()
}

func getKubeClient() (*kubernetes.Clientset, error) {
	kubeconfig := conf.Devops.KubeConfig

	if kubeconfig == "" {
		kubeconfig = filepath.Join(homedir.HomeDir(), ".kube/config")
	} else {
		kubeconfig, _ = filepath.Abs(kubeconfig)
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// ensureNS 确保 namespace 存在
func ensureNS(ns string) error {
	clientSet, err := getKubeClient()
	if err != nil {
		return err
	}
	namespace := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	}
	_, err = clientSet.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			// 没有找到就create
			_, err = clientSet.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
			if err != nil {
				return nil
			}
		} else {
			return err
		}
	}
	return nil
}

func Deploy(ctx *CIContext, task *DeployResult) (err error) {
	deployCtx := &DeployContext{
		CIContext:    ctx,
		DeployResult: task,
		repNum:       int32(1),
		labels:       GetPodLabels(ctx.repo, ctx.refName, ctx.commit),
		svcLabels:    GetSvcLabels(ctx.repo),
		stateful:     ctx.config.Deploy.Stateful,
	}
	config := ctx.config.Deploy

	// ensure namespace
	// TODO: 每个namespace中进行资源配额限制
	if err = ensureNS(task.Namespace); err != nil {
		return
	}

	// describe container
	deployCtx.container = getContainer(deployCtx)

	// deploy container
	if !config.Stateful {
		err = deployDeployment(deployCtx)
		if err != nil {
			return err
		}
	} else {
		err = deployStatefulSet(deployCtx)
		if err != nil {
			return err
		}
	}

	// 部署service
	service, err := deployService(deployCtx)
	if err != nil {
		return err
	}

	// 获取部署后的端口
	deployPorts := make([]Port, 0, len(config.Ports))
	for _, port := range service.Spec.Ports {
		deployPorts = append(deployPorts, Port{
			Name:     port.Name,
			Port:     port.NodePort,
			Protocol: string(port.Protocol),
		})
	}
	task.Ports = deployPorts
	task.StringPorts()

	// 获取恰当的IP
	task.IP = nextIP()

	return nil
}

func deployService(ctx *DeployContext) (result *v1.Service, err error) {
	// service 系列端口
	serviceName := ctx.ServiceName
	var svcPorts []apiv1.ServicePort
	for _, port := range ctx.container.Ports {
		p := apiv1.ServicePort{
			Name:     port.Name,
			Protocol: port.Protocol,
			Port:     port.ContainerPort,
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: port.ContainerPort,
			},
		}
		svcPorts = append(svcPorts, p)
	}
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   serviceName,
			Labels: ctx.svcLabels,
		},

		Spec: apiv1.ServiceSpec{
			Type:     apiv1.ServiceTypeNodePort,
			Selector: ctx.svcLabels,
			Ports:    svcPorts,
		},
	}

	// 删除之前存在的service
	serviceClient := clientSet.CoreV1().Services(ctx.Namespace)

	// 先检查是不是存在之前的service
	oldService, err := serviceClient.Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, err
		}
	} else {
		// 删除原来的service
		deletePolicy := metav1.DeletePropagationBackground
		if err := serviceClient.Delete(context.TODO(), serviceName, metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}); err != nil {
			return nil, err
		}

		// 等待service删除结束
		err = waitForDone(ctx, time.Second, func() (bool, error) {
			_, err := serviceClient.Get(context.TODO(), serviceName, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		if err != nil {
			return
		}
		// 如果有旧的端口号，那么使用旧的端口号
		for i := range service.Spec.Ports {
			nodePort := getSvcNodePortBySourcePort(oldService.Spec.Ports, service.Spec.Ports[i].Port)
			if nodePort <= 0 {
				nodePort = getSvcNodePortByName(oldService.Spec.Ports, service.Spec.Ports[i].Name)
			}
			if nodePort > 0 {
				service.Spec.Ports[i].NodePort = nodePort
			}
		}
	}

	// create
	_, err = serviceClient.Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		return
	}

	// 等待创建成功
	err = waitForDone(ctx, 5*time.Second, func() (bool, error) {
		result, err := serviceClient.Get(context.TODO(), serviceName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		// 确保端口都映射上了
		return result.Spec.ClusterIP != "" && len(result.Spec.Ports) == len(ctx.config.Deploy.Ports), nil
	})
	if err != nil {
		return
	}
	return serviceClient.Get(context.TODO(), serviceName, metav1.GetOptions{})
}

func deployDeployment(ctx *DeployContext) (err error) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: ctx.DeploymentName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &ctx.repNum,
			Selector: &metav1.LabelSelector{
				MatchLabels: ctx.svcLabels,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ctx.labels,
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						*ctx.container,
					},
				},
			},
		},
	}

	// 看看之前部署的deployment还存不存在
	deploymentsClient := clientSet.AppsV1().Deployments(ctx.Namespace)
	_, err = deploymentsClient.Get(context.TODO(), ctx.DeploymentName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			// create
			_, err = deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
			if err != nil {
				return
			}
		} else {
			return err
		}
	} else {
		// 否则，原来的deployment跑的好好的，那么就只需要更新就行
		_, err = deploymentsClient.Update(context.TODO(), deployment, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	err = waitForPodsDone(ctx)
	return
}

func deployStatefulSet(ctx *DeployContext) (err error) {
	ctx.repNum = 1
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: ctx.DeploymentName,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &ctx.repNum,
			ServiceName: ctx.ServiceName,
			Selector: &metav1.LabelSelector{
				MatchLabels: ctx.svcLabels,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ctx.labels,
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						*ctx.container,
					},
				},
			},
		},
	}

	// 看看之前部署的StatefulSet还存不存在
	statefulSetClient := clientSet.AppsV1().StatefulSets(ctx.Namespace)
	_, err = statefulSetClient.Get(context.TODO(), ctx.DeploymentName, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
	} else {
		// 删除原来的statefulSet
		deletePolicy := metav1.DeletePropagationBackground
		if err := statefulSetClient.Delete(context.TODO(), ctx.DeploymentName, metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}); err != nil {
			return err
		}

		// 等待statefulSet删除结束
		err = waitForDone(ctx, time.Second, func() (bool, error) {
			_, err := statefulSetClient.Get(context.TODO(), ctx.DeploymentName, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		if err != nil {
			return err
		}
	}

	// create
	_, err = statefulSetClient.Create(context.TODO(), statefulSet, metav1.CreateOptions{})
	if err != nil {
		return
	}

	err = waitForPodsDone(ctx)
	return
}

func waitForPodsDone(ctx *DeployContext) error {
	return waitForDone(ctx, 5*time.Second, func() (bool, error) {
		pods, err := listPods(ctx.labels, ctx.Namespace)
		if err != nil {
			return false, err
		}
		if len(pods) != int(ctx.repNum) {
			return false, nil
		}
		cntOK := 0
		for _, pod := range pods {
			if phase := pod.Status.Phase; phase == apiv1.PodRunning || phase == apiv1.PodSucceeded {
				cntOK++
			} else if phase != apiv1.PodPending {
				return false, fmt.Errorf("pod(%s) failed: %s", pod.Name, pod.Status.Message)
			}
		}
		return cntOK == int(ctx.repNum), nil
	})
}

func getContainer(ctx *DeployContext) *apiv1.Container {
	container := &apiv1.Container{
		Name:  ctx.repo.DeployName() + "-pod",
		Image: ctx.imageTag,
	}

	config := ctx.config.Deploy

	// 端口
	container.Ports = getContainerPorts(config.Ports)

	// 环境变量
	var envs []apiv1.EnvVar
	for k, v := range config.Envs {
		envs = append(envs, apiv1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	container.Env = envs

	// workingDir
	if len(config.WorkingDir) > 0 {
		container.WorkingDir = config.WorkingDir
	}

	// command
	if len(config.Cmd.Command) > 0 {
		container.Command = config.Cmd.Command
	}

	// args
	if len(config.Cmd.Args) > 0 {
		container.Args = config.Cmd.Args
	}

	// TODO: 替换策略

	// TODO: Stateful
	return container
}

// CheckKubeHealthy 检查已经部署的各个resource是否 working well
func CheckKubeHealthy(labels map[string]string, namespace, svcName string) (bool, error) {
	pods, err := listPods(labels, namespace)
	if err != nil {
		return false, err
	}
	podOK := false
	for _, pod := range pods {
		if phase := pod.Status.Phase; phase == apiv1.PodRunning || phase == apiv1.PodSucceeded {
			podOK = true
			break
		}
	}
	if !podOK {
		return false, nil
	}

	serviceClient := clientSet.CoreV1().Services(namespace)
	_, err = serviceClient.Get(context.TODO(), svcName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

func listPods(labels map[string]string, namespace string) ([]v1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: labels,
	})
	if err != nil {
		return nil, err
	}
	podList, err := clientSet.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

func getContainerPorts(dbPorts []Port) []apiv1.ContainerPort {
	var ports []apiv1.ContainerPort
	for i, port := range dbPorts {
		p := apiv1.ContainerPort{}

		// 处理端口名称
		if port.Name != "" {
			p.Name = port.Name
		} else {
			p.Name = fmt.Sprintf("port%d", i)
		}

		// 处理端口协议，默认为tcp
		var protocol apiv1.Protocol
		switch strings.ToLower(port.Protocol) {
		case "udp":
			protocol = apiv1.ProtocolTCP
		case "sctp":
			protocol = apiv1.ProtocolSCTP
		default:
			protocol = apiv1.ProtocolTCP
		}
		p.Protocol = protocol

		//  处理端口号
		p.ContainerPort = port.Port

		ports = append(ports, p)
	}
	return ports
}

func getSvcNodePortBySourcePort(ports []apiv1.ServicePort, sourcePort int32) int32 {
	for _, v := range ports {
		if v.Port == sourcePort {
			return v.NodePort
		}
	}
	return 0
}

func getSvcNodePortByName(ports []apiv1.ServicePort, name string) int32 {
	for _, v := range ports {
		if v.Name == name {
			return v.NodePort
		}
	}
	return 0
}

func waitForDone(ctx context.Context, atLeast time.Duration, judge func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	done := make(chan error)
	go func() {
		for {
			ok, err := judge()
			if err != nil {
				done <- err
				return
			}
			if ok {
				break
			}
			time.Sleep(atLeast)
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
		done <- nil
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
