package url

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"plandex/term"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	// Constants for fetchURLContent function
	maxRedirections    = 10
	httpTimeout        = 30 * time.Second
	maxContentSizeInMB = 10
)

func FetchURLContent(url string) (string, error) {
	client := &http.Client{
		Timeout: httpTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirections {
				return errors.New("stopped after too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.New("non-2xx HTTP response status: " + resp.Status)
	}

	// Limit the response reader to a maximum amount
	limitedReader := io.LimitReader(resp.Body, maxContentSizeInMB*1024*1024)

	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", err
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		return ExtractTextualContent(string(content)), nil
	} else {
		return string(content), nil
	}
}

func ExtractTextualContent(htmlContent string) string {
	r := strings.NewReader(htmlContent)
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		term.OutputErrorAndExit("Failed to parse HTML: %v", err)
	}

	return doc.Text()
}

func SanitizeURL(url string) string {
	// remove protocol portion with a regex
	re := regexp.MustCompile(`^.*?://`)
	url = re.ReplaceAllString(url, "")

	// Replace common invalid filename characters. You can extend this list as needed.
	sanitized := strings.ReplaceAll(url, ":", "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "?", "_")
	sanitized = strings.ReplaceAll(sanitized, "&", "_")
	sanitized = strings.ReplaceAll(sanitized, "=", "_")
	sanitized = strings.ReplaceAll(sanitized, "#", "_")
	sanitized = strings.ReplaceAll(sanitized, "%", "_")
	sanitized = strings.ReplaceAll(sanitized, "*", "_")
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	return sanitized
}

func IsValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
