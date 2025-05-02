package kube

import (
	"flag"
	"path/filepath"

	"github.com/smoothzz/kubehomedns/pkg/logger"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// InitialClientSet initializes the Kubernetes clientset by kubeconfig or in-cluster config
func InitialClientSet() (*kubernetes.Clientset, error) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		logger.Logger.Error("Error getting cluster config from flags", zap.Error(err))
		config, err = rest.InClusterConfig()
		if err != nil {
			logger.Logger.Fatal("Error getting cluster config from in-cluster", zap.Error(err))
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Logger.Fatal("Error creating Kubernetes clientset", zap.Error(err))
		return nil, err
	}

	logger.Logger.Info("Kubernetes clientset created successfully!")
	return clientset, nil
}
