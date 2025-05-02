package ingress

import (
	"context"
	"time"

	"github.com/smoothzz/kubehomedns/internal/api"
	"github.com/smoothzz/kubehomedns/internal/dns"
	"github.com/smoothzz/kubehomedns/pkg/logger"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func WatchIngresses(clientset *kubernetes.Clientset, cloudflareAPIKey, cloudflareZoneID string) {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		ingressList, err := clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
		cancel()

		if err != nil {
			logger.Logger.Error("Error listing ingresses", zap.Error(err))
			time.Sleep(10 * time.Second)
			continue
		}

		ingressCtrls := ingressList.Items
		if len(ingressCtrls) > 0 {
			for _, ingress := range ingressCtrls {
				for _, rule := range ingress.Spec.Rules {
					rec := dns.DNSRecord{Name: rule.Host}
					zone := (&api.ZoneIdentifier{ID: cloudflareZoneID}).ToResourceContainer()
					dns.CheckAndUpdateDNSRecord(cloudflareAPIKey, zone, rec)
				}
			}
		} else {
			logger.Logger.Info("No ingress found")
		}

		time.Sleep(1800 * time.Second)
	}
}

func WatchIngressLabels(clientset *kubernetes.Clientset, cloudflareAPIKey, cloudflareZoneID string) {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		ingressList, err := clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
		cancel()

		if err != nil {
			logger.Logger.Error("Error listing ingresses", zap.Error(err))
			time.Sleep(10 * time.Second)
			continue
		}

		for _, ingress := range ingressList.Items {
			labels := ingress.Labels
			if _, exists := labels["kubehomedns"]; exists {
				rec := dns.DNSRecord{Name: ingress.Spec.Rules[0].Host}
				zone := (&api.ZoneIdentifier{ID: cloudflareZoneID}).ToResourceContainer()
				dns.CheckAndUpdateDNSRecord(cloudflareAPIKey, zone, rec)
				// remove label after update
				delete(ingress.Labels, "kubehomedns")
				_, err := clientset.NetworkingV1().Ingresses(ingress.Namespace).Update(context.Background(), &ingress, metav1.UpdateOptions{})
				if err != nil {
					logger.Logger.Error("Error updating ingress labels", zap.Error(err))
				}
			}
		}
		time.Sleep(120 * time.Second)
	}
}
