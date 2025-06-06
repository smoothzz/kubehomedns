package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/smoothzz/kubehomedns/pkg/logger"
	"go.uber.org/zap"
)

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

type ResourceContainer struct {
	ID string
}

type ZoneIdentifier struct {
	ID string
}

func (z *ZoneIdentifier) ToResourceContainer() *ResourceContainer {
	return &ResourceContainer{ID: z.ID}
}

var client = &http.Client{
	Timeout: 10 * time.Second,
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
