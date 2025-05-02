package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/smoothzz/kubehomedns/logger"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type ResourceContainer struct {
	ID string
}

type ZoneIdentifier struct {
	ID string
}

func (z *ZoneIdentifier) ToResourceContainer() *ResourceContainer {
	return &ResourceContainer{ID: z.ID}
}

type DNSRecord struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

type IPResponse struct {
	IP string `json:"ip"`
}

type CFListDNSRecordsResult struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

type CFListDNSRecordsResponse struct {
	Result []CFListDNSRecordsResult `json:"result"`
}

type CFListDNSRecordsResponseSingle struct {
	Result CFListDNSRecordsResult `json:"result"`
}

var client = &http.Client{
	Timeout: 10 * time.Second,
}

func DoJSONRequest(ctx context.Context, method, url string, jsonData []byte, apiKey string, headers map[string]string) (*http.Response, error) {
	var body io.Reader
	if jsonData != nil {
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		logger.Logger.Error("Failed to create request", zap.Error(err))
		return nil, err
	}

	// Default headers - validation if there is apiKey
	// If apiKey is empty, set only Content-Type
	if apiKey == "" {
		req.Header.Set("Content-Type", "application/json")
	} else {
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
	}

	// Set additional headers if any
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Logger.Error("Failed to send request", zap.Error(err))
		return nil, err
	}

	return resp, nil
}

func CFListDNSRecords(apiKey string, zone *ResourceContainer) (*CFListDNSRecordsResponse, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zone.ID)

	ctx := context.Background()
	resp, err := DoJSONRequest(ctx, "GET", url, nil, apiKey, nil)
	if err != nil {
		logger.Logger.Error("Failed to make request", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Logger.Error("Failed to list DNS records", zap.String("status", resp.Status))
		return nil, err
	}

	var response CFListDNSRecordsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		logger.Logger.Error("Failed to decode response", zap.Error(err))
		return nil, err
	}

	return &response, nil
}

func CFListDNSRecordById(apiKey string, zone *ResourceContainer, recordID string) (*CFListDNSRecordsResponseSingle, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zone.ID, recordID)

	ctx := context.Background()
	resp, err := DoJSONRequest(ctx, "GET", url, nil, apiKey, nil)
	if err != nil {
		logger.Logger.Error("Failed to make request", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Logger.Error("Failed to list DNS records", zap.String("status", resp.Status))
		return nil, err
	}
	var response CFListDNSRecordsResponseSingle
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		logger.Logger.Error("Failed to decode response", zap.Error(err))
		return nil, err
	}
	return &response, nil
}

func CFCreateDNSRecord(apiKey string, zone *ResourceContainer, record DNSRecord) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zone.ID)

	ip, err := GetPublicIP()
	if err != nil {
		logger.Logger.Error("Failed to get public IP", zap.Error(err))
		return err
	}

	recordData := map[string]interface{}{
		"type":    "A",
		"name":    record.Name,
		"content": ip,
	}
	jsonData, err := json.Marshal(recordData)
	if err != nil {
		logger.Logger.Error("Failed to marshal JSON", zap.Error(err))
		return err
	}

	ctx := context.Background()
	resp, err := DoJSONRequest(ctx, "POST", url, jsonData, apiKey, nil)
	if err != nil {
		logger.Logger.Error("Failed to make request", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Logger.Error("Failed to create DNS record", zap.String("status", resp.Status))
		return nil
	}

	return nil
}

func GetPublicIP() (string, error) {
	url := "https://api.ipify.org?format=json"

	ctx := context.Background()
	resp, err := DoJSONRequest(ctx, "GET", url, nil, "", nil)
	if err != nil {
		logger.Logger.Error("Error sending request", zap.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Logger.Error("Error response from API", zap.String("status", resp.Status))
		return "", fmt.Errorf("error response from API: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Logger.Error("Error reading response body", zap.Error(err))
		return "", err
	}

	var ipResponse IPResponse
	if err := json.Unmarshal(body, &ipResponse); err != nil {
		logger.Logger.Error("Error unmarshalling JSON", zap.Error(err))
		return "", err
	}

	return ipResponse.IP, nil
}

func CFUpdateDNSRecord(apiKey string, zone *ResourceContainer, recordName string, recordID string, newContent string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zone.ID, recordID)

	recordData := map[string]interface{}{
		"content": newContent,
		"type":    "A",
		"name":    recordName,
	}

	jsonData, err := json.Marshal(recordData)
	if err != nil {
		logger.Logger.Error("Failed to marshal JSON", zap.Error(err))
		return err
	}

	ctx := context.Background()
	resp, err := DoJSONRequest(ctx, "PUT", url, jsonData, apiKey, nil)
	if err != nil {
		logger.Logger.Error("Failed to make request", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Logger.Error("Failed to update DNS record", zap.String("status", resp.Status))
		return nil
	}

	var response CFListDNSRecordsResponseSingle
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		logger.Logger.Error("Failed to decode response", zap.Error(err))
		return err
	}

	if resp.StatusCode == http.StatusOK {
		logger.Logger.Info("DNS record updated successfully", zap.String("record_name", response.Result.Name), zap.String("new_content", response.Result.Content))
	}

	return nil
}

func CFDeleteDNSRecord(apiKey string, zone *ResourceContainer, recordID string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zone.ID, recordID)

	ctx := context.Background()
	resp, err := DoJSONRequest(ctx, "DELETE", url, nil, apiKey, nil)
	if err != nil {
		logger.Logger.Error("Failed to make request", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Logger.Error("Failed to delete DNS record", zap.String("status", resp.Status))
		return err
	}

	return nil
}

func CheckAndUpdateDNSRecord(apiKey string, zone *ResourceContainer, record DNSRecord) {
	// Check if the DNS record needs to be updated
	getRecord, err := CFListDNSRecords(apiKey, zone)
	if err != nil {
		logger.Logger.Fatal("Error listing DNS records", zap.Error(err))
	}

	var recordID string
	var recordName string
	for _, r := range getRecord.Result {
		if r.Name == record.Name {
			recordID = r.Id
			recordName = r.Name
			break
		}
	}

	if recordID == "" {
		// Create the DNS record if it doesn't exist
		err := CFCreateDNSRecord(apiKey, zone, record)
		if err != nil {
			logger.Logger.Fatal("Error creating DNS record", zap.Error(err))
		}
		logger.Logger.Info("DNS record created successfully", zap.String("record_name", record.Name))
		return
	}

	// If the record exists, check if it needs to be updated
	// Get the current content of the DNS record
	resultById, err := CFListDNSRecordById(apiKey, zone, recordID)
	if err != nil {
		logger.Logger.Fatal("Error listing DNS record by ID", zap.Error(err))
	}

	currentContent := resultById.Result.Content
	CurrentPublicIP, err := GetPublicIP()
	if err != nil {
		logger.Logger.Error("Failed to get public IP", zap.Error(err))
		return
	}

	if currentContent == CurrentPublicIP {
		logger.Logger.Info("No update needed for DNS record", zap.String("record_name", recordName))
		return
	}

	if currentContent != CurrentPublicIP {
		// Update the DNS record
		err := CFUpdateDNSRecord(apiKey, zone, recordName, recordID, CurrentPublicIP)
		if err != nil {
			logger.Logger.Fatal("Error updating DNS record", zap.Error(err))
		}
		// logger.Logger.Info("DNS record updated successfully", zap.String("record_name", recordName), zap.String("new_content", CurrentPublicIP))
	}
}

func GetCredentialsFromSecret(clientset *kubernetes.Clientset) (string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	secret, err := clientset.CoreV1().Secrets("kubehomedns").Get(ctx, "cloudflare-credentials", metav1.GetOptions{})
	defer cancel()
	if err != nil {
		logger.Logger.Fatal("Error retrieving secret", zap.Error(err))
		return "", ""
	}

	var cloudflare_api_key string
	var cloudflare_zone_id string

	for key, value := range secret.Data {
		if key == "cloudflare_api_key" {
			cloudflare_api_key = string(value)
		}
		if key == "cloudflare_zone_id" {
			cloudflare_zone_id = string(value)
		}
	}

	return cloudflare_api_key, cloudflare_zone_id
}

func InitialClientSet() *kubernetes.Clientset {
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
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Logger.Fatal("Error creating Kubernetes clientset", zap.Error(err))
	}

	logger.Logger.Info("Kubernetes clientset created successfully!")
	return clientset
}

func watchIngresses(clientset *kubernetes.Clientset, cloudflareAPIKey, cloudflareZoneID string) {
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
					CheckAndUpdateDNSRecord(cloudflareAPIKey, (&ZoneIdentifier{ID: cloudflareZoneID}).ToResourceContainer(), DNSRecord{Name: rule.Host})
				}
			}
		} else {
			logger.Logger.Info("No ingress found")
		}

		time.Sleep(1800 * time.Second)
	}
}

func watchIngLabels(clientset *kubernetes.Clientset, cloudflareAPIKey, cloudflareZoneID string) {
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
				CheckAndUpdateDNSRecord(cloudflareAPIKey, (&ZoneIdentifier{ID: cloudflareZoneID}).ToResourceContainer(), DNSRecord{Name: ingress.Spec.Rules[0].Host})
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
func main() {
	if err := logger.Init(); err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Logger.Info("Kubehomedns has started!")

	clientset := InitialClientSet()

	cloudflareAPIKey, cloudflareZoneID := GetCredentialsFromSecret(clientset)

	go watchIngLabels(clientset, cloudflareAPIKey, cloudflareZoneID)

	go watchIngresses(clientset, cloudflareAPIKey, cloudflareZoneID)

	select {}

}
