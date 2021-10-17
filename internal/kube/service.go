package kube

import (
	"context"
	"strings"

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type ServiceOpt struct {
	BaseServiceOpt
	Type v1.ServiceType
}

type BaseServiceOpt struct {
	Name string
	// Labels 表示选取哪些Pod
	Labels map[string]string
	Ports  []Port
}

func (cli *Client) GetService(ctx context.Context, name string) (svc *v1.Service, err error) {
	return cli.CoreV1().Services(cli.namespace).Get(ctx, name, metav1.GetOptions{})
}

func (cli *Client) CreateOrReplaceService(ctx context.Context, opt *ServiceOpt) (svc *v1.Service, err error) {
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   opt.Name,
			Labels: opt.Labels,
		},

		Spec: v1.ServiceSpec{
			Type:     opt.Type,
			Selector: opt.Labels,
			Ports:    getSvePorts(opt.Ports),
		},
	}

	serviceClient := cli.CoreV1().Services(cli.namespace)
	// 先检查是不是存在之前的service
	oldSvc, err := serviceClient.Get(ctx, opt.Name, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return
	}
	if err == nil {
		if comparePorts(oldSvc.Spec.Ports, opt.Ports) {
			// 如果服务的端口号没变，那么直接返回就好
			svc = oldSvc
			return
		}
		// 删除原来的service
		deletePolicy := metav1.DeletePropagationForeground
		if err = serviceClient.Delete(ctx, opt.Name, metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}); err != nil {
			return
		}
	}

	// create
	_, err = serviceClient.Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		return
	}
	return serviceClient.Get(ctx, opt.Name, metav1.GetOptions{})
}

func comparePorts(svcPorts []v1.ServicePort, ports []Port) bool {
	if len(svcPorts) != len(ports) {
		return false
	}
	for i := range svcPorts {
		sp, pp := svcPorts[i], ports[i]
		if sp.Name != pp.Name ||
			sp.TargetPort.IntVal != pp.Port ||
			strings.ToLower(string(sp.Protocol)) != strings.ToLower(pp.Protocol) {
			return false
		}
	}
	return true
}

func getSvePorts(ports []Port) []v1.ServicePort {
	containerPorts := getContainerPorts(ports)
	var svcPorts []v1.ServicePort
	for _, port := range containerPorts {
		p := v1.ServicePort{
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
	return svcPorts
}
