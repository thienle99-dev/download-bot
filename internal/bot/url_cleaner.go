package bot

import (
	"net/url"
	"strings"
)

// CleanURL removes tracking parameters from any given URL
func CleanURL(rawURL string) (string, error) {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return "", err
	}

	queries := parsed.Query()
	for key := range queries {
		kLower := strings.ToLower(key)
		// Blacklist of tracking parameters
		if strings.HasPrefix(kLower, "utm_") ||
			kLower == "si" ||
			kLower == "igsh" ||
			kLower == "fbclid" ||
			kLower == "gclid" ||
			kLower == "dclid" ||
			kLower == "yclid" ||
			kLower == "ttclid" ||
			kLower == "ref" ||
			kLower == "referrer" ||
			kLower == "spm" ||
			kLower == "_r" ||
			kLower == "mibextid" {
			queries.Del(key)
		}
	}

	parsed.RawQuery = queries.Encode()
	return parsed.String(), nil
}
