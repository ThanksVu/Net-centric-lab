package main

import (
	"fmt"
)

type Book struct {
	Id     int
	Title  string
	Author string
	Year   int
}

var books []Book

// Seed initial data
func init() {
	books = []Book{
		{Id: 1, Title: "Book A", Author: "Author A", Year: 2020},
		{Id: 2, Title: "Book B", Author: "Author B", Year: 2021},
		{Id: 3, Title: "Book C", Author: "Author C", Year: 2022},
	}
}

// Create a new book
func createBook(b Book) {
	books = append(books, b)
	fmt.Println("Created:", b)
}

// Get all books
func getBooks() {
	fmt.Println("All Books:")
	for _, b := range books {
		fmt.Printf("%+v\n", b)
	}
}

// Update book by ID
func updateBook(id int, updated Book) {
	for i, b := range books {
		if b.Id == id {
			books[i] = updated
			fmt.Println("Updated:", updated)
			return
		}
	}
	fmt.Println("Book not found")
}

// Delete book by ID
func deleteBook(id int) {
	for i, b := range books {
		if b.Id == id {
			books = append(books[:i], books[i+1:]...)
			fmt.Println("Deleted ID:", id)
			return
		}
	}
	fmt.Println("Book not found")
}

func main() {
	fmt.Println("Initial books:")
	getBooks()

	// Create
	createBook(Book{Id: 4, Title: "Book D", Author: "Author D", Year: 2023})

	// Update
	updateBook(2, Book{Id: 2, Title: "Book B Updated", Author: "Author BB", Year: 2024})

	// Delete
	deleteBook(1)

	fmt.Println("\nFinal list:")
	getBooks()
}
