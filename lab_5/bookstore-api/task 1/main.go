package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

type Book struct {
	ID            int     `json:"id"`
	Title         string  `json:"title" binding:"required"`
	Author        string  `json:"author" binding:"required"`
	ISBN          string  `json:"isbn"`
	Price         float64 `json:"price" binding:"required,gt=0"`
	Stock         int     `json:"stock" binding:"gte=0"`
	PublishedYear int     `json:"published_year"`
	Description   string  `json:"description"`
	CreatedAt     string  `json:"created_at"`
}

var db *sql.DB

// ------------------------------------------------------------
// INIT DB
// ------------------------------------------------------------
func initDB() error {
	var err error
	db, err = sql.Open("sqlite3", "./bookstore.db")
	if err != nil {
		return err
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		isbn TEXT UNIQUE,
		price REAL NOT NULL CHECK(price > 0),
		stock INTEGER DEFAULT 0,
		published_year INTEGER,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createTableSQL)
	return err
}

// ------------------------------------------------------------
// SEED DATA
// ------------------------------------------------------------
func seedData() {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM books").Scan(&count)
	if count > 0 {
		return
	}

	sampleBooks := []Book{
	{Title: "The Go Programming Language", Author: "Alan Donovan", ISBN: "978-0134190440", Price: 39.99, Stock: 15, PublishedYear: 2015, Description: "Complete guide to Go"},
	{Title: "Clean Code", Author: "Robert C. Martin", ISBN: "978-0132350884", Price: 45.99, Stock: 10, PublishedYear: 2008, Description: "A Handbook of Agile Software Craftsmanship"},
	{Title: "Design Patterns", Author: "Erich Gamma", ISBN: "978-0201633610", Price: 49.99, Stock: 8, PublishedYear: 1994, Description: "Elements of Reusable Object-Oriented Software"},
	{Title: "The Pragmatic Programmer", Author: "Andy Hunt", ISBN: "978-0201616224", Price: 42.50, Stock: 12, PublishedYear: 1999, Description: "Classic programming handbook"},
	{Title: "Code Complete", Author: "Steve McConnell", ISBN: "978-0735619678", Price: 55.00, Stock: 5, PublishedYear: 2004, Description: "Software engineering handbook"},
}


	stmt, _ := db.Prepare(`
		INSERT INTO books (title, author, isbn, price, stock, published_year, description)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)

	for _, b := range sampleBooks {
		stmt.Exec(b.Title, b.Author, b.ISBN, b.Price, b.Stock, b.PublishedYear, b.Description)
	}
}

// ------------------------------------------------------------
// GET /books
// ------------------------------------------------------------
func getBooks(c *gin.Context) {
	rows, err := db.Query("SELECT id, title, author, isbn, price, stock, published_year, description, created_at FROM books")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	books := []Book{}
	for rows.Next() {
		var b Book
		rows.Scan(&b.ID, &b.Title, &b.Author, &b.ISBN, &b.Price, &b.Stock, &b.PublishedYear, &b.Description, &b.CreatedAt)
		books = append(books, b)
	}

	c.JSON(http.StatusOK, gin.H{
		"books": books,
		"count": len(books),
	})
}

// ------------------------------------------------------------
// GET /books/:id
// ------------------------------------------------------------
func getBook(c *gin.Context) {
	id := c.Param("id")
	var b Book

	err := db.QueryRow(`
		SELECT id, title, author, isbn, price, stock, published_year, description, created_at
		FROM books WHERE id = ?`, id).
		Scan(&b.ID, &b.Title, &b.Author, &b.ISBN, &b.Price, &b.Stock, &b.PublishedYear, &b.Description, &b.CreatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}

	c.JSON(http.StatusOK, b)
}

// ------------------------------------------------------------
// POST /books
// ------------------------------------------------------------
func createBook(c *gin.Context) {
	var book Book

	if err := c.ShouldBindJSON(&book); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := db.Exec(`
		INSERT INTO books (title, author, isbn, price, stock, published_year, description)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		book.Title, book.Author, book.ISBN, book.Price, book.Stock, book.PublishedYear, book.Description,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	book.ID = int(id)

	c.JSON(http.StatusCreated, book)
}

// ------------------------------------------------------------
// PUT /books/:id
// ------------------------------------------------------------
func updateBook(c *gin.Context) {
	id := c.Param("id")
	var book Book

	if err := c.ShouldBindJSON(&book); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := db.Exec(`
		UPDATE books SET title=?, author=?, isbn=?, price=?, stock=?, published_year=?, description=?
		WHERE id=?`,
		book.Title, book.Author, book.ISBN, book.Price, book.Stock, book.PublishedYear, book.Description, id,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}

	book.ID = atoi(id)
	c.JSON(http.StatusOK, book)
}

// ------------------------------------------------------------
// DELETE /books/:id
// ------------------------------------------------------------
func deleteBook(c *gin.Context) {
	id := c.Param("id")

	result, err := db.Exec("DELETE FROM books WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Book deleted"})
}

// Helper for conversion
func atoi(s string) int {
	var i int
	fmt.Sscan(s, &i)
	return i
}

// ------------------------------------------------------------
// MAIN
// ------------------------------------------------------------
func main() {
	if err := initDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	seedData()

	router := gin.Default()

	router.GET("/books", getBooks)
	router.GET("/books/:id", getBook)
	router.POST("/books", createBook)
	router.PUT("/books/:id", updateBook)
	router.DELETE("/books/:id", deleteBook)

	fmt.Println("🚀 Server running at http://localhost:8080")
	router.Run(":8080")
}
