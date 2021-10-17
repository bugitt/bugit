package kube

import (
	"context"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Client is a wrapper for *kubernetes.Clientset
type Client struct {
	*kubernetes.Clientset
	Ctx       context.Context
	namespace string
}

type CreateClientOpt struct {
	ConfigPath string
	Namespace  string
}

func client(confPath string) (*kubernetes.Clientset, error) {
	if confPath == "" {
		confPath = filepath.Join(homedir.HomeDir(), ".kube/config")
	} else {
		confPath, _ = filepath.Abs(confPath)
	}
	config, err := clientcmd.BuildConfigFromFlags("", confPath)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func NewClient(ctx context.Context, opt *CreateClientOpt) (*Client, error) {
	clientSet, err := client(opt.ConfigPath)
	if err != nil {
		return nil, err
	}
	if len(opt.Namespace) <= 0 {
		opt.Namespace = DefaultNameSpace
	}
	return &Client{
		Clientset: clientSet,
		Ctx:       ctx,
		namespace: opt.Namespace,
	}, nil

}
