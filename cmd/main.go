package main

import (
	"fmt"
	"os"

	"github.com/smoothzz/kubehomedns/internal/ingress"
	"github.com/smoothzz/kubehomedns/internal/kube"
	"github.com/smoothzz/kubehomedns/pkg/config"
	"github.com/smoothzz/kubehomedns/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	if err := logger.Init(); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	if err := config.Load(logger.Logger); err != nil {
		logger.Logger.Fatal("Failed to load config", zap.Error(err))
	}

	clientset, err := kube.InitialClientSet()
	if err != nil {
		logger.Logger.Fatal("Error initializing clientset", zap.Error(err))
		return
	}

	cloudflareAPIKey := config.CloudflareAPIKey
	cloudflareZoneID := config.CloudflareZoneID

	logger.Logger.Info("Kubehomedns has started!")

	go ingress.WatchIngresses(clientset, cloudflareAPIKey, cloudflareZoneID)
	go ingress.WatchIngressLabels(clientset, cloudflareAPIKey, cloudflareZoneID)

	select {}
}
