package main

import (
	"context"
	"fmt"
	"log"
	"time"

	authorpb "book-catalog-grpc/proto"
	bookpb "book-catalog-grpc/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to both services
	bookConn, err := grpc.Dial("127.0.0.1:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer bookConn.Close()

	authorConn, err := grpc.Dial("127.0.0.1:50052",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer authorConn.Close()

	bookClient := bookpb.NewBookCatalogClient(bookConn)
	authorClient := authorpb.NewAuthorCatalogClient(authorConn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("=== Microservice Demo ===\n")

	// 1. Create author
	fmt.Println("1. Creating author...")
	authorResp, err := authorClient.CreateAuthor(ctx, &authorpb.CreateAuthorRequest{
		Name:      "Martin Fowler",
		Bio:       "Software development expert",
		BirthYear: 1963,
		Country:   "UK",
	})
	if err != nil {
		log.Printf("Failed to create author: %v", err)
	} else {
		fmt.Printf("✓ Created author: %s (ID: %d)\n\n",
			authorResp.Author.Name, authorResp.Author.Id)

		// 2. Create books for this author
		fmt.Println("2. Creating books for author...")
		book1, _ := bookClient.CreateBook(ctx, &bookpb.CreateBookRequest{
			Title:         "Refactoring",
			Author:        "Martin Fowler",
			AuthorId:      authorResp.Author.Id,
			Isbn:          "978-0134757599",
			Price:         49.99,
			Stock:         15,
			PublishedYear: 2018,
		})
		fmt.Printf("✓ Created book: %s\n", book1.Book.Title)

		book2, _ := bookClient.CreateBook(ctx, &bookpb.CreateBookRequest{
			Title:         "Patterns of Enterprise Application Architecture",
			Author:        "Martin Fowler",
			AuthorId:      authorResp.Author.Id,
			Isbn:          "978-0321127426",
			Price:         54.99,
			Stock:         8,
			PublishedYear: 2002,
		})
		fmt.Printf("✓ Created book: %s\n\n", book2.Book.Title)

		// 3. Get author's books (cross-service call)
		fmt.Println("3. Fetching author's books (cross-service call)...")
		booksResp, err := authorClient.GetAuthorBooks(ctx, &authorpb.GetAuthorBooksRequest{
			AuthorId: authorResp.Author.Id,
		})
		if err != nil {
			log.Printf("Failed: %v", err)
		} else {
			fmt.Printf("✓ Author: %s\n", booksResp.Author.Name)
			fmt.Printf("✓ Books written: %d\n", booksResp.BookCount)
			for i, book := range booksResp.Books {
				fmt.Printf("  %d. %s (%d) - $%.2f\n", i+1, book.Title, book.PublishedYear, book.Price)
			}
		}
	}

	// 4. List all authors
	fmt.Println("\n4. Listing all authors...")
	authorsResp, err := authorClient.ListAuthors(ctx, &authorpb.ListAuthorsRequest{
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		log.Printf("Failed to list authors: %v", err)
	} else {
		fmt.Printf("✓ Total authors: %d\n", authorsResp.Total)
		for i, author := range authorsResp.Authors {
			fmt.Printf("  %d. %s (%d, %s)\n", i+1, author.Name, author.BirthYear, author.Country)
		}
	}

	// 5. Get statistics from Book service
	fmt.Println("\n5. Getting book statistics...")
	statsResp, err := bookClient.GetStats(ctx, &bookpb.GetStatsRequest{})
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
	} else {
		fmt.Printf("✓ Total books: %d\n", statsResp.TotalBooks)
		fmt.Printf("✓ Average price: $%.2f\n", statsResp.AveragePrice)
		fmt.Printf("✓ Total stock: %d\n", statsResp.TotalStock)
		fmt.Printf("✓ Year range: %d - %d\n", statsResp.EarliestYear, statsResp.LatestYear)
	}

	fmt.Println("\n✅ Microservice demo completed successfully!")
	fmt.Println("📊 Demonstrated:")
	fmt.Println("   - Service-to-service communication (Author → Book)")
	fmt.Println("   - CRUD operations across multiple services")
	fmt.Println("   - Cross-service data aggregation")
}
