package util

import (
	"fmt"
	"strings"
)

// GenURL adds 'http' prefix for URL
func GenURL(url string) string {
	if strings.Contains(url, "http") {
		return url
	}

	return fmt.Sprintf("http://%s", url)
}
