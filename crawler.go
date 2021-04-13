package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/romanlevin/crawler-go/link_queue"
	"github.com/romanlevin/crawler-go/link_set"
	"golang.org/x/net/html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// fetchPage returns the body of a page from disk if it's already been written, otherwise fetches it using HTTP
func fetchPage(ctx context.Context, link string, client *http.Client, localPath string) ([]byte, error) {
	// Check if the path exists locally, and return its contents if it does
	if body, err := os.ReadFile(localPath); err == nil {
		log.Printf("read %q from disk", link)
		return body, nil
	}

	// If the local path does not contain a file, fetch its contents using `link`
	request, err := http.NewRequestWithContext(ctx, "GET", link, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := resp.Body.Close(); err != nil {
		log.Printf("error closing response body: %+v", err)
	}
	log.Printf("read %q remotely", link)
	return body, nil
}

// getLinks returns a slice of the values of the `href` attributes of all the `<a>` tags in an HTML page
func getLinks(page []byte) ([]string, error) {
	doc, err := html.Parse(bytes.NewReader(page))
	var links []string
	if err != nil {
		return nil, err
	}
	var f func(*html.Node)
	// Mostly copied directly from https://pkg.go.dev/golang.org/x/net/html#example-Parse
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					links = append(links, a.Val)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return links, nil
}

// writePage writes a `page` to `path` unless a file already exists there
func writePage(page []byte, path string) error {
	if fileInfo, err := os.Stat(path); err == nil {
		if fileInfo.Mode().IsRegular() {
			// Don't write back file that was fetched from disk
			return nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(path, page, 0644); err != nil {
		return err
	}

	return nil
}

// fileName converts a link url to its on-disk equivalent
func fileName(link string, start string, outDir string) (string, error) {
	// Make sure there's only one final `/`
	start = strings.TrimSuffix(start, "/") + "/"

	var pathAndQuery string
	// If this is a root document, call it `index.html`
	if len(link) > len(start) {
		pathAndQuery = link[len(start)-1:]
	}
	if len(pathAndQuery) == 0 {
		pathAndQuery = "index.html"
	}

	parsedUrl, err := url.Parse(pathAndQuery)
	if err != nil {
		return "", err
	}

	path := parsedUrl.EscapedPath()
	query := parsedUrl.Query()
	if len(query) == 0 {
		// If there's no query component, we have the full path
		return filepath.Join(outDir, filepath.FromSlash(path)), nil
	}

	// If there's a query component, we need to escape `/` characters
	queryStringBuilder := &strings.Builder{}
	for key, values := range query {
		// Replace `/` in query keys and values with "%2F"
		key = strings.ReplaceAll(key, "/", "%2F")

		// Handle queries keys with no values
		if len(values) == 1 && values[0] == "" {
			if _, err := fmt.Fprintf(queryStringBuilder, "%s&", key); err != nil {
				return "", err
			}
			continue
		}

		for _, value := range values {
			value = strings.ReplaceAll(value, "/", "%2F")
			if _, err := fmt.Fprintf(queryStringBuilder, "%s=%s&", key, value); err != nil {
				return "", err
			}
		}
	}

	// Concat the path and query components and return, trimming the final `&` in the query
	path = path + "?" + queryStringBuilder.String()[:queryStringBuilder.Len()-1]
	return filepath.Join(outDir, filepath.FromSlash(path)), nil
}

// urlDefrag returns `linkURL` minus its fragment portion
func urlDefrag(linkURL string) (string, error) {
	parsedLinkURL, err := url.Parse(linkURL)
	if err != nil {
		return "", err
	}
	// Remove the fragment from the url
	parsedLinkURL.Fragment = ""
	return parsedLinkURL.String(), nil
}

func processUrl(ctx context.Context, link string, client *http.Client, seenLinks *link_set.LinkSet, toCrawl *link_queue.LinkQueue, start string, outDir string) error {
	localPath, err := fileName(link, start, outDir)
	if err != nil {
		return err
	}

	page, err := fetchPage(ctx, link, client, localPath)
	if err != nil {
		return err
	}

	linkURLs, err := getLinks(page)
	if err != nil {
		return err
	}

	parsedStart, err := url.Parse(start)
	if err != nil {
		return err
	}

	for _, linkURL := range linkURLs {
		defraggedURL, err := urlDefrag(linkURL)
		if err != nil {
			// If the link is malformed, skip it
			log.Printf("malformed url: %q", linkURL)
			continue
		}

		joinedURL, err := parsedStart.Parse(defraggedURL)
		if err != nil {
			log.Printf("error joining url: %q", linkURL)
			continue
		}

		joinedURLString := joinedURL.String()

		if !seenLinks.Has(joinedURLString) && strings.HasPrefix(joinedURLString, start) {
			toCrawl.Push(joinedURLString)
		}

		seenLinks.Add(joinedURLString)
	}

	if err := writePage(page, localPath); err != nil {
		log.Fatalln(err)
	}

	return nil
}

// crawl crawls a website at `start`, saving its pages to `outDir`
func crawl(ctx context.Context, start string, outDir string) error {
	client := &http.Client{}

	seenLinks := link_set.New()
	seenLinks.Add(start)

	// XXX: Possibly replace the channel with some kind of queue implementation?
	toCrawl := &link_queue.LinkQueue{}
	toCrawl.Push(start)

	for {
		if currentUrl := toCrawl.Pop(); currentUrl != "" {
			if err := processUrl(ctx, currentUrl, client, seenLinks, toCrawl, start, outDir); err != nil {
				return err
			}
		}
	}
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Expected invocation of the form %s <start-url> <output-dir>", os.Args[0])
	}

	start := os.Args[1]
	if !strings.HasPrefix(start, "http://") && !strings.HasPrefix(start, "https://") {
		log.Fatalln("First parameter must start with `https://` or `http://`")
	}

	outDir := os.Args[2]

	ctx := context.TODO()

	if err := crawl(ctx, start, outDir); err != nil {
		log.Fatalln(err)
	}
}
