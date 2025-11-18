package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type Book struct {
	Title        string `json:"title"`
	Price        string `json:"price"`
	Rating       string `json:"rating"`
	Availability string `json:"availability"`
	ImageURL     string `json:"image_url"`
}

// =============================
// scrapeBooks()
// =============================
func scrapeBooks(url string) ([]Book, error) {
	fmt.Println("Scraping books from:", url)

	// Step 1: Fetch HTML page
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %v", err)
	}
	defer resp.Body.Close()

	// Step 2: Parse HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Step 3: Find all <article class="product_pod">
	var books []Book
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "article" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "product_pod") {
					book := extractBookData(n)
					books = append(books, book)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return books, nil
}

// =============================
// extractBookData()
// =============================
func extractBookData(n *html.Node) Book {
	book := Book{}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Title: <h3><a title="...">
			if n.Data == "a" {
				for _, attr := range n.Attr {
					if attr.Key == "title" {
						book.Title = strings.TrimSpace(attr.Val)
					}
					if attr.Key == "href" && book.ImageURL == "" {
						// We'll get image URL later
					}
				}
			}

			// Price: <p class="price_color">
			if n.Data == "p" {
				for _, attr := range n.Attr {
					if attr.Key == "class" && strings.Contains(attr.Val, "price_color") {
						if n.FirstChild != nil {
							book.Price = strings.TrimSpace(n.FirstChild.Data)
						}
					}
					// Rating: <p class="star-rating Three">
					if attr.Key == "class" && strings.Contains(attr.Val, "star-rating") {
						parts := strings.Fields(attr.Val)
						if len(parts) == 2 {
							book.Rating = strings.TrimSpace(parts[1])
						}
					}
					// Availability: <p class="instock availability">
					if attr.Key == "class" && strings.Contains(attr.Val, "availability") {
						if n.FirstChild != nil {
							book.Availability = strings.TrimSpace(strings.ReplaceAll(n.FirstChild.Data, "\n", ""))
						}
					}
				}
			}

			// Image URL: <img src="...">
			if n.Data == "img" {
				for _, attr := range n.Attr {
					if attr.Key == "src" {
						// prepend base URL
						book.ImageURL = "http://books.toscrape.com/" + strings.TrimPrefix(attr.Val, "../")
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)

	return book
}

// =============================
// saveBooksToJSON()
// =============================
func saveBooksToJSON(books []Book, filename string) error {
	data, err := json.MarshalIndent(books, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// =============================
// calculateAveragePrice()
// =============================
func calculateAveragePrice(books []Book) float64 {
	var total float64
	count := 0
	for _, b := range books {
		priceStr := strings.ReplaceAll(b.Price, "£", "")
		if p, err := strconv.ParseFloat(priceStr, 64); err == nil {
			total += p
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// =============================
// main()
// =============================
func main() {
	url := "http://books.toscrape.com/catalogue/page-1.html"

	books, err := scrapeBooks(url)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Found %d books\n\n", len(books))

	if len(books) > 0 {
		fmt.Println("Book 1:")
		fmt.Printf("  Title: %s\n", books[0].Title)
		fmt.Printf("  Price: %s\n", books[0].Price)
		fmt.Printf("  Rating: %s\n", books[0].Rating)
		fmt.Printf("  Availability: %s\n\n", books[0].Availability)
	}

	avg := calculateAveragePrice(books)
	fmt.Println("Summary:")
	fmt.Printf("  Total books: %d\n", len(books))
	fmt.Printf("  Average price: £%.2f\n\n", avg)

	if err := saveBooksToJSON(books, "books.json"); err != nil {
		fmt.Println("Error saving JSON:", err)
		return
	}
	fmt.Printf("Saved %d books to books.json\n", len(books))
}
