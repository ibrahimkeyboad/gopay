package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SendWebhook sends the JSON payload to the merchant's URL
func SendWebhook(url string, payload interface{}) error {
	// 1. Convert Payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// 2. Prepare Request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoPay-Webhook/1.0")

	// 3. Send with Timeout (Don't let slow merchants block us!)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 4. Check Response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil // Success
	}

	return fmt.Errorf("merchant server returned error: %d", resp.StatusCode)
}