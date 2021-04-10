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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// nextIP 获取这次应该部署的到哪个IP上
var nextIP = func() func() string {
	i := -1
	return func() string {
		i = (i + 1) % len(conf.Devops.KubeIP)
		return conf.Devops.KubeIP[i]
	}
}()

func getKubeClient() (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube/config"))
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
	_, err = clientSet.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		// 排除 namespace 已经存在的情况
		if !checkErrNotFound(err) {
			return err
		}
	}
	return nil
}

func getNameSpace(ctx *CIContext) string {
	return fmt.Sprintf("%d-%d", ctx.repo.ProjectID, ctx.owner.ID)
}

func Deploy(ctx *CIContext, task *DeployTask) (err error) {
	// 前期准备
	clientSet, err := getKubeClient()
	if err != nil {
		return
	}
	var repNum int32 = 1
	config := ctx.config.Deploy
	ns := getNameSpace(ctx)
	validRepoName := strings.Replace(ctx.repo.LowerName, "_", "-", -1)
	deployName := validRepoName + "-deployment"
	serviceName := validRepoName + "-service"
	labels := map[string]string{
		"app":     validRepoName,
		"project": ns,
		"ref":     ctx.refName,
		"commit":  ctx.commit,
	}
	svcLabels := map[string]string{
		"app":     validRepoName,
		"project": ns,
	}

	// ensure namespace
	// TODO: 每个namespace中进行资源配额限制
	if err = ensureNS(ns); err != nil {
		return
	}

	// 删除之前存在的deployment
	deploymentsClient := clientSet.AppsV1().Deployments(ns)
	deletePolicy := metav1.DeletePropagationBackground
	if err := deploymentsClient.Delete(context.TODO(), deployName, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		if !checkErrNotFound(err) {
			return err
		}
	}

	// 等待deployment删除结束
	err = waitForDone(ctx, time.Second, func() (bool, error) {
		_, err := deploymentsClient.Get(context.TODO(), deployName, metav1.GetOptions{})
		if err != nil {
			if checkErrNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	if err != nil {
		return
	}

	// describe pod
	container := apiv1.Container{
		Name:  validRepoName + "-pod",
		Image: ctx.imageTag,
	}

	// 端口
	var ports []apiv1.ContainerPort
	for i, port := range config.Ports {
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
	container.Ports = ports

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

	// 定义 deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: deployName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &repNum,
			Selector: &metav1.LabelSelector{
				MatchLabels: svcLabels,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{container},
				},
			},
		},
	}

	// create
	_, err = deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		return
	}

	// 等待创建成功
	err = waitForDone(ctx, 5*time.Second, func() (bool, error) {
		result, err := deploymentsClient.Get(context.TODO(), deployName, metav1.GetOptions{})
		if err != nil {
			if checkErrNotFound(err) {
				return false, nil
			}
			return false, err
		}
		// TODO: 这里只能保证当部署成功时能退出，如果部署失败，比如镜像没有拉取到，就要一直等到pipeline超时才能得到结果
		return result.Status.AvailableReplicas == result.Status.ReadyReplicas && result.Status.AvailableReplicas == repNum, nil
	})
	if err != nil {
		return
	}

	// 部署 service

	// 删除之前存在的service
	serviceClient := clientSet.CoreV1().Services(ns)
	if err := serviceClient.Delete(context.TODO(), serviceName, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		if !checkErrNotFound(err) {
			return err
		}
	}

	// 等待service删除结束
	err = waitForDone(ctx, time.Second, func() (bool, error) {
		_, err := serviceClient.Get(context.TODO(), serviceName, metav1.GetOptions{})
		if err != nil {
			if checkErrNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	if err != nil {
		return
	}

	// service 系列端口
	// TODO：如果有旧的端口号，那么使用旧的端口号
	var svcPorts []apiv1.ServicePort
	for _, port := range ports {
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
			Labels: svcLabels,
		},

		Spec: apiv1.ServiceSpec{
			Type:     apiv1.ServiceTypeNodePort,
			Selector: svcLabels,
			Ports:    svcPorts,
		},
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
			if checkErrNotFound(err) {
				return false, nil
			}
			return false, err
		}
		// 确保端口都映射上了
		return result.Spec.ClusterIP != "" && len(result.Spec.Ports) == len(config.Ports), nil
	})
	if err != nil {
		return
	}

	// 获取部署后的端口
	deployPorts := make([]Port, 0, len(config.Ports))
	result, err := serviceClient.Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		return
	}
	for _, port := range result.Spec.Ports {
		deployPorts = append(deployPorts, Port{
			Name:     port.Name,
			Port:     port.NodePort,
			Protocol: string(port.Protocol),
		})
	}

	task.Ports = deployPorts
	task.StringPorts()
	task.IP = nextIP()

	return nil
}

func checkErrNotFound(err error) bool {
	e, ok := err.(*kerrors.StatusError)
	return ok && e.ErrStatus.Code/100 == 4
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
