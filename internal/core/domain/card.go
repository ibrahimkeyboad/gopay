package domain

import (
	"regexp"
	"strconv"
	"strings"
)

type CardType string

const (
	Visa       CardType = "VISA"
	Mastercard CardType = "MASTERCARD"
	Unknown    CardType = "UNKNOWN"
)

// ValidateCard checks if the card is valid and allowed
func ValidateCard(number string) (bool, CardType) {
	// 1. Remove spaces and dashes
	cleanNum := strings.ReplaceAll(number, " ", "")
	cleanNum = strings.ReplaceAll(cleanNum, "-", "")

	// 2. Check if it's a valid number (Luhn Algorithm)
	if !passesLuhn(cleanNum) {
		return false, Unknown
	}

	// 3. Check the Brand (Visa vs Mastercard)
	// Visa: Starts with 4, length 13 or 16
	visaRegex := regexp.MustCompile(`^4[0-9]{12}(?:[0-9]{3})?$`)
	
	// Mastercard: Starts with 51-55, length 16
	masterRegex := regexp.MustCompile(`^5[1-5][0-9]{14}$`)

	if visaRegex.MatchString(cleanNum) {
		return true, Visa
	}
	if masterRegex.MatchString(cleanNum) {
		return true, Mastercard
	}

	// If it's Amex, Discover, etc., we return FALSE because you only want Visa/Master
	return false, Unknown
}

// passesLuhn implements the standard Mod 10 check used by all banks
func passesLuhn(number string) bool {
	sum := 0
	alternate := false
	for i := len(number) - 1; i >= 0; i-- {
		n, _ := strconv.Atoi(string(number[i]))
		if alternate {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alternate = !alternate
	}
	return sum%10 == 0
}