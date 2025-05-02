package dns

import (
	"github.com/smoothzz/kubehomedns/internal/api"
	"github.com/smoothzz/kubehomedns/pkg/logger"
	"go.uber.org/zap"
)

type ResourceContainer = api.ResourceContainer
type DNSRecord = api.DNSRecord

// CheckAndUpdateDNSRecord checks if the DNS record exists and is up-to-date, creates or updates it otherwise
func CheckAndUpdateDNSRecord(apiKey string, zone *ResourceContainer, record DNSRecord) {
	// List all DNS records in the zone
	getRecord, err := api.CFListDNSRecords(apiKey, zone)
	if err != nil {
		logger.Logger.Fatal("Error listing DNS records", zap.Error(err))
		return
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
		err := api.CFCreateDNSRecord(apiKey, zone, record)
		if err != nil {
			logger.Logger.Fatal("Error creating DNS record", zap.Error(err))
			return
		}
		logger.Logger.Info("DNS record created successfully", zap.String("record_name", record.Name))
		return
	}

	// If the record exists, check if it needs to be updated
	resultById, err := api.CFListDNSRecordById(apiKey, zone, recordID)
	if err != nil {
		logger.Logger.Fatal("Error listing DNS record by ID", zap.Error(err))
		return
	}

	currentContent := resultById.Result.Content

	CurrentPublicIP, err := api.GetPublicIP()
	if err != nil {
		logger.Logger.Error("Failed to get public IP", zap.Error(err))
		return
	}

	if currentContent == CurrentPublicIP {
		logger.Logger.Info("No update needed for DNS record", zap.String("record_name", recordName))
		return
	}

	if currentContent != CurrentPublicIP {
		err := api.CFUpdateDNSRecord(apiKey, zone, recordName, recordID, CurrentPublicIP)
		if err != nil {
			logger.Logger.Fatal("Error updating DNS record", zap.Error(err))
		}
	}
}
