package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"time"

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

func CFListDNSRecords(apiKey string, zone *ResourceContainer) (*CFListDNSRecordsResponse, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zone.ID)

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set the necessary headers
	req.Header.Set("Authorization", "Bearer "+apiKey) // Use API Token
	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list DNS records: %s", resp.Status)
	}

	var response CFListDNSRecordsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

func CFListDNSRecordById(apiKey string, zone *ResourceContainer, recordID string) (*CFListDNSRecordsResponseSingle, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zone.ID, recordID)

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set the necessary headers
	req.Header.Set("Authorization", "Bearer "+apiKey) // Use API Token
	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list DNS records: %s", resp.Status)
	}
	var response CFListDNSRecordsResponseSingle
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &response, nil
}

func CFCreateDNSRecord(apiKey string, zone *ResourceContainer, record DNSRecord) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zone.ID)

	recordData := map[string]interface{}{
		"type":    "A",
		"name":    record.Name,
		"content": GetPublicIP(),
	}
	jsonData, err := json.Marshal(recordData)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set the necessary headers
	req.Header.Set("Authorization", "Bearer "+apiKey) // Use API Token
	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create DNS record: %s", resp.Status)
	}

	return nil
}

func GetPublicIP() string {
	url := "https://api.ipify.org?format=json"

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v\n", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %v\n", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error: received status code %d\n", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v\n", err)
	}

	var ipResponse IPResponse
	if err := json.Unmarshal(body, &ipResponse); err != nil {
		log.Fatalf("Error unmarshalling JSON: %v\n", err)
	}

	return ipResponse.IP
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
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Create a new HTTP request
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set the necessary headers
	req.Header.Set("Authorization", "Bearer "+apiKey) // Use API Token
	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update DNS record: %s", resp.Status)
	}

	var response CFListDNSRecordsResponseSingle
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("Record %s updated successfully, new content: %s!\n", response.Result.Name, response.Result.Content)
	}

	return nil
}

func CFDeleteDNSRecord(apiKey string, zone *ResourceContainer, recordID string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zone.ID, recordID)

	// Create a new HTTP request
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set the necessary headers
	req.Header.Set("Authorization", "Bearer "+apiKey) // Use API Token
	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete DNS record: %s", resp.Status)
	}

	return nil
}

func CheckAndUpdateDNSRecord(apiKey string, zone *ResourceContainer, record DNSRecord) {
	// Check if the DNS record needs to be updated
	getRecord, err := CFListDNSRecords(apiKey, zone)
	if err != nil {
		log.Fatalf("Error listing DNS records: %v\n", err)
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
			log.Fatalf("Error creating DNS record: %v\n", err)
		}
		fmt.Printf("DNS record %s created successfully!\n", record.Name)
		return
	}
	// If the record exists, check if it needs to be updated
	// Get the current content of the DNS record
	resultById, err := CFListDNSRecordById(apiKey, zone, recordID)
	if err != nil {
		log.Fatalf("Error listing DNS record by ID: %v\n", err)
	}
	currentContent := resultById.Result.Content
	CurrentPublicIP := GetPublicIP()
	// Compare the current content with the new content
	if currentContent == CurrentPublicIP {
		fmt.Printf("No update needed for DNS record %s.\n", recordName)
		return
	}
	if currentContent != CurrentPublicIP {
		// Update the DNS record
		err := CFUpdateDNSRecord(apiKey, zone, recordName, recordID, CurrentPublicIP)
		if err != nil {
			log.Fatalf("Error updating DNS record: %v\n", err)
		}
	}
}

func GetCredentialsFromSecret(clientset *kubernetes.Clientset) (string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	secret, err := clientset.CoreV1().Secrets("kubehomedns").Get(ctx, "cloudflare-credentials", metav1.GetOptions{})
	defer cancel()
	if err != nil {
		fmt.Printf("Error retrieving secret: %v\n", err)
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

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Printf("error getting cluster config from flags: %s\n", err.Error())
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(fmt.Sprintf("error getting cluster config from inCluster: %s\n", err.Error()))
		}
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return clientset
}

func watchIngresses(clientset *kubernetes.Clientset, cloudflareAPIKey, cloudflareZoneID string) {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		ingressList, err := clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
		cancel() // cancel context promptly after use

		if err != nil {
			fmt.Printf("Error listing ingresses: %v\n", err)
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
			fmt.Println("No ingress found")
		}

		time.Sleep(1800 * time.Second)
	}
}

func watchIngLabels(clientset *kubernetes.Clientset, cloudflareAPIKey, cloudflareZoneID string) {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		ingressList, err := clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
		cancel() // cancel context promptly after use

		if err != nil {
			fmt.Printf("Error listing ingresses: %v\n", err)
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
					fmt.Printf("Error updating ingress labels: %v\n", err)
				}
			}
		}
		time.Sleep(120 * time.Second)
	}
}

func main() {
	clientset := InitialClientSet()

	cloudflareAPIKey, cloudflareZoneID := GetCredentialsFromSecret(clientset)

	go watchIngLabels(clientset, cloudflareAPIKey, cloudflareZoneID)

	go watchIngresses(clientset, cloudflareAPIKey, cloudflareZoneID)

	select {}

}
