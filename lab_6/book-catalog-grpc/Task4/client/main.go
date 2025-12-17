package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "book-catalog-grpc/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	// Kết nối đến server Task4 (port 50053)
	conn, err := grpc.Dial("127.0.0.1:50053",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewBookCatalogClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test 1: Search by Title
	fmt.Println("=== Test 1: Search by Title ===")
	fmt.Println("Searching for \"go\"...")
	searchResp, err := client.SearchBooks(ctx, &pb.SearchBooksRequest{
		Query: "go",
		Field: "title",
	})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("Error: %s\n", st.Message())
	} else {
		fmt.Printf("Found %d books:\n", searchResp.Count)
		for _, book := range searchResp.Books {
			fmt.Printf("- %s\n", book.Title)
		}
	}

	// Test 2: Search by Author
	fmt.Println("\n=== Test 2: Search by Author ===")
	fmt.Println("Searching for \"Martin\"...")
	searchResp2, err := client.SearchBooks(ctx, &pb.SearchBooksRequest{
		Query: "Martin",
		Field: "author",
	})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("Error: %s\n", st.Message())
	} else {
		fmt.Printf("Found %d books:\n", searchResp2.Count)
		for _, book := range searchResp2.Books {
			fmt.Printf("- %s by %s\n", book.Title, book.Author)
		}
	}

	// Test 3: Search all fields
	fmt.Println("\n=== Test 3: Search All Fields ===")
	fmt.Println("Searching for \"programming\" in all fields...")
	searchResp3, err := client.SearchBooks(ctx, &pb.SearchBooksRequest{
		Query: "programming",
		Field: "all",
	})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("Error: %s\n", st.Message())
	} else {
		fmt.Printf("Found %d books\n", searchResp3.Count)
	}

	// Test 4: Filter by Price
	fmt.Println("\n=== Test 4: Filter by Price ===")
	fmt.Println("Books between $20 and $45:")
	filterResp, err := client.FilterBooks(ctx, &pb.FilterBooksRequest{
		MinPrice: 20.0,
		MaxPrice: 45.0,
	})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("Error: %s\n", st.Message())
	} else {
		fmt.Printf("Found %d books:\n", filterResp.Count)
		for _, book := range filterResp.Books {
			fmt.Printf("- %s: $%.2f\n", book.Title, book.Price)
		}
	}

	// Test 5: Filter by Year
	fmt.Println("\n=== Test 5: Filter by Year ===")
	fmt.Println("Books published after 2010:")
	filterResp2, err := client.FilterBooks(ctx, &pb.FilterBooksRequest{
		MinYear: 2010,
	})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("Error: %s\n", st.Message())
	} else {
		fmt.Printf("Found %d books:\n", filterResp2.Count)
		for _, book := range filterResp2.Books {
			fmt.Printf("- %s (%d)\n", book.Title, book.PublishedYear)
		}
	}

	// Test 6: Get Statistics
	fmt.Println("\n=== Test 6: Get Statistics ===")
	statsResp, err := client.GetStats(ctx, &pb.GetStatsRequest{})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("Error: %s\n", st.Message())
	} else {
		fmt.Printf("Total books: %d\n", statsResp.TotalBooks)
		fmt.Printf("Average price: $%.2f\n", statsResp.AveragePrice)
		fmt.Printf("Total stock: %d\n", statsResp.TotalStock)
		fmt.Printf("Year range: %d - %d\n", statsResp.EarliestYear, statsResp.LatestYear)
	}

	// Test 7: Error Cases
	fmt.Println("\n=== Test 7: Error Cases ===")

	// Empty search query
	fmt.Println("Test: Empty search query")
	_, err = client.SearchBooks(ctx, &pb.SearchBooksRequest{
		Query: "",
		Field: "title",
	})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("✓ Expected error: %s\n", st.Message())
	}

	// Invalid field
	fmt.Println("\nTest: Invalid search field")
	_, err = client.SearchBooks(ctx, &pb.SearchBooksRequest{
		Query: "test",
		Field: "invalid",
	})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("✓ Expected error: %s\n", st.Message())
	}

	// Invalid price range
	fmt.Println("\nTest: Invalid price range (min > max)")
	_, err = client.FilterBooks(ctx, &pb.FilterBooksRequest{
		MinPrice: 100.0,
		MaxPrice: 50.0,
	})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("✓ Expected error: %s\n", st.Message())
	}

	// Negative price
	fmt.Println("\nTest: Negative price")
	_, err = client.FilterBooks(ctx, &pb.FilterBooksRequest{
		MinPrice: -10.0,
	})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("✓ Expected error: %s\n", st.Message())
	}
}
