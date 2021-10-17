package kube

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
)

const (
	DefaultDuration  = 10 * time.Minute
	DefaultNameSpace = "default"
)

// PullPolicy describes a policy for if/when to pull a container image
type PullPolicy string

const (
	// PullAlways means that kubelet always attempts to pull the latest image. Container will fail If the pull fails.
	PullAlways PullPolicy = "Always"
	// PullNever means that kubelet never pulls an image, but only uses a local image. Container will fail if the image isn't present
	PullNever PullPolicy = "Never"
	// PullIfNotPresent means that kubelet pulls if the image isn't present on disk. Container will fail if the image isn't present and the pull fails.
	PullIfNotPresent PullPolicy = "IfNotPresent"
)

type AuthConfig struct {
}

type DeployOpt struct {
	Name             string
	Labels           map[string]string
	ReplicaNum       int32
	Namespace        string
	PodLabels        map[string]string
	ImagePullSecrets []v1.LocalObjectReference
}

type Port struct {
	Name     string
	Protocol string
	Port     int32
}

type SvcPort struct {
	Port
	ExportPort int32
}

type Cmd struct {
	Command []string
	Args    []string
}

type PodController interface {
	DeployOrUpdate(ctx context.Context) error
	GetPods(ctx context.Context) ([]v1.Pod, error)
	Delete(ctx context.Context) error
	Exists(ctx context.Context) (bool, error)
}
