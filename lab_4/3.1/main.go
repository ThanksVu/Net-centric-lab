package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/net/html"
)

// ======================
// DATA STRUCTURES
// ======================

type Book struct {
	Title string `json:"title"`
	Price string `json:"price"`
	Link  string `json:"link"`
}

type ScraperStats struct {
	PagesScraped int
	BooksFound   int
	Errors       int
	StartTime    time.Time
	EndTime      time.Time
}

// ======================
// SCRAPE A SINGLE PAGE
// ======================

func scrapeBooksFromPage(doc *html.Node) ([]Book, error) {
	var books []Book

	var walker func(*html.Node)
	walker = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "article" {
			var book Book

			// Find book title and link
			var parseTitle func(*html.Node)
			parseTitle = func(node *html.Node) {
				if node.Type == html.ElementNode && node.Data == "h3" {
					a := node.FirstChild
					for _, attr := range a.Attr {
						if attr.Key == "title" {
							book.Title = attr.Val
						}
						if attr.Key == "href" {
							book.Link = "http://books.toscrape.com/catalogue/" + attr.Val
						}
					}
				}
				for c := node.FirstChild; c != nil; c = c.NextSibling {
					parseTitle(c)
				}
			}

			// Find price
			var parsePrice func(*html.Node)
			parsePrice = func(node *html.Node) {
				if node.Type == html.ElementNode && node.Data == "p" {
					for _, attr := range node.Attr {
						if attr.Key == "class" && attr.Val == "price_color" {
							if node.FirstChild != nil {
								book.Price = node.FirstChild.Data
							}
						}
					}
				}
				for c := node.FirstChild; c != nil; c = c.NextSibling {
					parsePrice(c)
				}
			}

			parseTitle(n)
			parsePrice(n)

			if book.Title != "" {
				books = append(books, book)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walker(c)
		}
	}
	walker(doc)

	return books, nil
}

// ======================
// FETCH PAGE (WITH RETRY)
// ======================

func fetchPageWithRetry(pageURL string, retries int) (*html.Node, error) {
	var lastError error

	for attempt := 1; attempt <= retries; attempt++ {
		resp, err := http.Get(pageURL)
		if err == nil && resp.StatusCode == 200 {
			defer resp.Body.Close()
			return html.Parse(resp.Body)
		}

		lastError = err
		fmt.Printf("⚠️ Retry %d/%d failed: %v\n", attempt, retries, err)
		time.Sleep(500 * time.Millisecond)
	}

	return nil, lastError
}

// ======================
// GET NEXT PAGE URL
// ======================

func getNextPageURL(doc *html.Node, currentURL string) (string, bool) {
	var nextHref string

	// Look for <li class="next"><a href="page-2.html">
	var walker func(*html.Node)
	walker = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "li" {
			for _, a := range n.Attr {
				if a.Key == "class" && a.Val == "next" {
					if n.FirstChild != nil {
						aTag := n.FirstChild
						for _, attr := range aTag.Attr {
							if attr.Key == "href" {
								nextHref = attr.Val
								return
							}
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walker(c)
		}
	}
	walker(doc)

	if nextHref == "" {
		return "", false
	}

	// Resolve absolute URL
	base, _ := url.Parse(currentURL)
	nextURL, err := base.Parse(nextHref)
	if err != nil {
		return "", false
	}

	return nextURL.String(), true
}

// ======================
// PAGINATION SCRAPER
// ======================

func scrapePaginatedBooks(baseURL string, maxPages int) ([]Book, *ScraperStats, error) {
	stats := &ScraperStats{StartTime: time.Now()}
	var allBooks []Book

	currentURL := baseURL

	for page := 1; page <= maxPages; page++ {
		fmt.Printf("Scraping page %d/%d... ", page, maxPages)

		// Fetch HTML
		doc, err := fetchPageWithRetry(currentURL, 3)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			stats.Errors++
			break
		}

		// Extract books
		books, err := scrapeBooksFromPage(doc)
		if err != nil {
			fmt.Printf("❌ Error parsing page: %v\n", err)
			stats.Errors++
		}

		fmt.Printf("Found %d books\n", len(books))

		allBooks = append(allBooks, books...)
		stats.BooksFound += len(books)
		stats.PagesScraped++

		// Find next page
		nextURL, ok := getNextPageURL(doc, currentURL)
		if !ok {
			break
		}
		currentURL = nextURL

		// Rate limit
		time.Sleep(1 * time.Second)
	}

	stats.EndTime = time.Now()
	return allBooks, stats, nil
}

// ======================
// PRINT STATISTICS
// ======================

func printStats(stats *ScraperStats) {
	fmt.Println("\n=== Scraping Statistics ===")
	fmt.Printf("Pages scraped: %d\n", stats.PagesScraped)
	fmt.Printf("Total books found: %d\n", stats.BooksFound)
	fmt.Printf("Errors: %d\n", stats.Errors)
	fmt.Printf("Duration: %.1f seconds\n", stats.EndTime.Sub(stats.StartTime).Seconds())

	if stats.PagesScraped > 0 {
		avg := float64(stats.BooksFound) / float64(stats.PagesScraped)
		fmt.Printf("Average books per page: %.1f\n", avg)
	}
}

// ======================
// MAIN FUNCTION
// ======================

func main() {
	baseURL := "http://books.toscrape.com/catalogue/page-1.html"
	maxPages := 5

	fmt.Println("Starting paginated scraper...")
	fmt.Printf("Max pages: %d\n\n", maxPages)

	books, stats, err := scrapePaginatedBooks(baseURL, maxPages)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	printStats(stats)

	// Save JSON
	file, _ := json.MarshalIndent(books, "", "  ")
	_ = os.WriteFile("paginated_books.json", file, 0644)

	fmt.Printf("\nSaved %d books to paginated_books.json\n", len(books))
}
