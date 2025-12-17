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
	// Kết nối đến server
	conn, err := grpc.Dial("127.0.0.1:50052",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewBookCatalogClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test 1: List All Books
	fmt.Println("=== Test 1: List All Books ===")
	listResp, err := client.ListBooks(ctx, &pb.ListBooksRequest{
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("Error: %s\n", st.Message())
	} else {
		fmt.Printf("Total books: %d\n", listResp.Total)
		for i, book := range listResp.Books {
			fmt.Printf("%d. %s by %s - $%.2f\n", i+1, book.Title, book.Author, book.Price)
		}
	}

	// Test 2: Get Book
	fmt.Println("\n=== Test 2: Get Book ===")
	getResp, err := client.GetBook(ctx, &pb.GetBookRequest{Id: 1})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("Error: %s\n", st.Message())
	} else {
		book := getResp.Book
		fmt.Printf("Book ID: %d\n", book.Id)
		fmt.Printf("Title: %s\n", book.Title)
		fmt.Printf("Author: %s\n", book.Author)
		fmt.Printf("Price: $%.2f\n", book.Price)
		fmt.Printf("Stock: %d\n", book.Stock)
		fmt.Printf("Published Year: %d\n", book.PublishedYear)
	}

	// Test 3: Create Book
	fmt.Println("\n=== Test 3: Create Book ===")
	createResp, err := client.CreateBook(ctx, &pb.CreateBookRequest{
		Title:         "Learning Go",
		Author:        "Jon Bodner",
		Isbn:          "978-1492077213",
		Price:         44.99,
		Stock:         30,
		PublishedYear: 2021,
	})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("Error: %s\n", st.Message())
	} else {
		fmt.Printf("Created book ID: %d\n", createResp.Book.Id)
		fmt.Printf("Title: %s\n", createResp.Book.Title)
		fmt.Printf("Author: %s\n", createResp.Book.Author)
	}

	// Test 4: Update Book
	fmt.Println("\n=== Test 4: Update Book ===")
	updateResp, err := client.UpdateBook(ctx, &pb.UpdateBookRequest{
		Id:            1,
		Title:         "The Go Programming Language (2nd Edition)",
		Author:        "Alan Donovan",
		Isbn:          "978-0134190440",
		Price:         35.99,
		Stock:         20,
		PublishedYear: 2015,
	})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("Error: %s\n", st.Message())
	} else {
		fmt.Printf("Updated book: %s\n", updateResp.Book.Title)
		fmt.Printf("New price: $%.2f\n", updateResp.Book.Price)
		fmt.Printf("New stock: %d\n", updateResp.Book.Stock)
	}

	// Test 5: Delete Book (xóa book vừa tạo)
	fmt.Println("\n=== Test 5: Delete Book ===")
	if createResp != nil && createResp.Book != nil {
		deleteResp, err := client.DeleteBook(ctx, &pb.DeleteBookRequest{
			Id: createResp.Book.Id,
		})
		if err != nil {
			st, _ := status.FromError(err)
			fmt.Printf("Error: %s\n", st.Message())
		} else {
			fmt.Printf("Success: %v\n", deleteResp.Success)
			fmt.Printf("Message: %s\n", deleteResp.Message)
		}
	}

	// Test 6: Pagination
	fmt.Println("\n=== Test 6: Pagination ===")

	// Page 1
	page1, err := client.ListBooks(ctx, &pb.ListBooksRequest{
		Page:     1,
		PageSize: 3,
	})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("Error: %s\n", st.Message())
	} else {
		fmt.Printf("Page 1 (Total: %d): %d books\n", page1.Total, len(page1.Books))
		for i, book := range page1.Books {
			fmt.Printf("  %d. %s\n", i+1, book.Title)
		}
	}

	// Page 2
	page2, err := client.ListBooks(ctx, &pb.ListBooksRequest{
		Page:     2,
		PageSize: 3,
	})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("Error: %s\n", st.Message())
	} else {
		fmt.Printf("Page 2 (Total: %d): %d books\n", page2.Total, len(page2.Books))
		for i, book := range page2.Books {
			fmt.Printf("  %d. %s\n", i+1, book.Title)
		}
	}

	// Test 7: Error handling - Get non-existent book
	fmt.Println("\n=== Test 7: Error Handling (Get Non-existent Book) ===")
	_, err = client.GetBook(ctx, &pb.GetBookRequest{Id: 9999})
	if err != nil {
		st, _ := status.FromError(err)
		fmt.Printf("Expected error: %s (Code: %s)\n", st.Message(), st.Code())
	}
}
