package notifications

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SendWebhook sends a signed request
func SendWebhook(url string, payload interface{}, secret string) error {
	// 1. Convert to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// 2. Calculate HMAC Signature
	// This creates a unique "hash" using the secret key
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(jsonData)
	signature := hex.EncodeToString(h.Sum(nil))

	// 3. Prepare Request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoPay-Webhook/1.0")
	
	// 4. Attach Signature Header
	// The merchant checks this header to verify it's us
	req.Header.Set("X-GoPay-Signature", signature)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("merchant error: %d", resp.StatusCode)
}