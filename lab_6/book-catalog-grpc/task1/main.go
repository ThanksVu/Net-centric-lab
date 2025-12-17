package main

import (
	"fmt"
	"log"

	pb "book-catalog-grpc/proto"

	"google.golang.org/protobuf/proto"
)

func main() {
	book := &pb.Book{
		Id:            1,
		Title:         "The Go Programming Language",
		Author:        "Alan Donovan",
		Isbn:          "978-0134190440",
		Price:         39.99,
		Stock:         15,
		PublishedYear: 2015,
	}

	detailedBook := &pb.DetailedBook{
		Book:        book,
		Category:    pb.BookCategory_NONFICTION,
		Description: "Go programming guide",
		Tags:        []string{"programming", "go", "technical"},
		Rating:      4.5,
	}

	data, err := proto.Marshal(book)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nDetailed Book: %v\n", detailedBook)
	fmt.Printf("Category: %s\n", detailedBook.Category.String())
	fmt.Printf("Tags: %v\n", detailedBook.Tags)

	newBook := &pb.Book{}
	if err := proto.Unmarshal(data, newBook); err != nil {
		log.Fatalf("unmarshal failed: %v", err)
	}

	author := &pb.Author{
		Id:        1,
		Name:      "Robert C. Martin",
		Bio:       "Clean Code author",
		BirthYear: 1952,
		Books: []*pb.Book{
			{Title: "Clean Code"},
			{Title: "Clean Architecture"},
		},
	}

	fmt.Println("Book:", book)
	fmt.Println("Serialized size:", len(data))
	fmt.Println("Deserialized:", newBook)
	fmt.Println("Author:", author.Name)

	for i, b := range author.Books {
		fmt.Printf("%d. %s\n", i+1, b.Title)
	}
}
