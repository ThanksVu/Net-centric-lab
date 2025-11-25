package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// ---------- Structs ----------

type Author struct {
	ID        int    `json:"id"`
	Name      string `json:"name" binding:"required"`
	Bio       string `json:"bio"`
	BirthYear int    `json:"birth_year"`
	Country   string `json:"country"`
	CreatedAt string `json:"created_at"`
}

type Book struct {
	ID            int     `json:"id"`
	Title         string  `json:"title" binding:"required,min=3"`
	AuthorID      int     `json:"author_id" binding:"required"`
	ISBN          string  `json:"isbn" binding:"required"`
	Price         float64 `json:"price" binding:"required,min=0.01,max=1000"`
	Stock         int     `json:"stock" binding:"gte=0"`
	PublishedYear int     `json:"published_year"`
	Description   string  `json:"description"`
}

type BookWithAuthor struct {
	Book
	AuthorName string `json:"author_name"`
	AuthorBio  string `json:"author_bio"`
}

type PaginationMeta struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

type PaginatedBooksResponse struct {
	Books      []BookWithAuthor `json:"books"`
	Pagination PaginationMeta   `json:"pagination"`
}

type Statistics struct {
	TotalBooks    int             `json:"total_books"`
	TotalAuthors  int             `json:"total_authors"`
	TotalValue    float64         `json:"total_value"`
	LowStock      int             `json:"low_stock"`
	OutOfStock    int             `json:"out_of_stock"`
	MostExpensive *BookWithAuthor `json:"most_expensive"`
	Cheapest      *BookWithAuthor `json:"cheapest"`
	MostStocked   *BookWithAuthor `json:"most_stocked"`
	BooksByYear   map[int]int     `json:"books_by_year"`
	AveragePrice  float64         `json:"average_price"`
}

type RestockRequest struct {
	Quantity int `json:"quantity" binding:"required,gt=0"`
}

type SellRequest struct {
	Quantity int `json:"quantity" binding:"required,gt=0"`
}

type BulkCreateRequest struct {
	Books []Book `json:"books" binding:"required,min=1,dive"`
}

type BulkCreateResponse struct {
	Success      int      `json:"success"`
	Failed       int      `json:"failed"`
	CreatedBooks []Book   `json:"created_books"`
	Errors       []string `json:"errors,omitempty"`
}

// ---------- Database Init ----------

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./bookstore.db")
	if err != nil {
		log.Fatal(err)
	}

	// Enable foreign keys
	db.Exec("PRAGMA foreign_keys = ON;")

	// Create authors table
	createAuthorsSQL := `
	CREATE TABLE IF NOT EXISTS authors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		bio TEXT,
		birth_year INTEGER,
		country TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	db.Exec(createAuthorsSQL)

	// Create books table
	createBooksSQL := `
	CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		author_id INTEGER,
		isbn TEXT NOT NULL,
		price REAL NOT NULL,
		stock INTEGER NOT NULL,
		published_year INTEGER,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(author_id) REFERENCES authors(id) ON DELETE SET NULL
	);`
	db.Exec(createBooksSQL)
}

// ---------- Helpers ----------

func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func validateISBN(isbn string) error {
	isbn = regexp.MustCompile(`-`).ReplaceAllString(isbn, "")
	if len(isbn) != 10 && len(isbn) != 13 {
		return fmt.Errorf("ISBN must be 10 or 13 digits")
	}
	if !regexp.MustCompile(`^\d+$`).MatchString(isbn) {
		return fmt.Errorf("ISBN must contain only digits")
	}
	return nil
}

func validatePublishedYear(year int) error {
	currentYear := time.Now().Year()
	if year < 1800 || year > currentYear {
		return fmt.Errorf("published year must be between 1800 and %d", currentYear)
	}
	return nil
}

// ---------- Author Endpoints ----------

func getAuthors(c *gin.Context) {
	rows, err := db.Query("SELECT id, name, bio, birth_year, country, created_at FROM authors")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	authors := []Author{}
	for rows.Next() {
		var a Author
		rows.Scan(&a.ID, &a.Name, &a.Bio, &a.BirthYear, &a.Country, &a.CreatedAt)
		authors = append(authors, a)
	}
	c.JSON(http.StatusOK, authors)
}

func getAuthor(c *gin.Context) {
	id := c.Param("id")
	var a Author
	err := db.QueryRow("SELECT id, name, bio, birth_year, country, created_at FROM authors WHERE id = ?", id).
		Scan(&a.ID, &a.Name, &a.Bio, &a.BirthYear, &a.Country, &a.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Author not found"})
		return
	}
	c.JSON(http.StatusOK, a)
}

func createAuthor(c *gin.Context) {
	var a Author
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := db.Exec("INSERT INTO authors (name, bio, birth_year, country) VALUES (?, ?, ?, ?)",
		a.Name, a.Bio, a.BirthYear, a.Country)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, _ := res.LastInsertId()
	a.ID = int(id)
	c.JSON(http.StatusCreated, a)
}

func updateAuthor(c *gin.Context) {
	id := c.Param("id")
	var a Author
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := db.Exec("UPDATE authors SET name=?, bio=?, birth_year=?, country=? WHERE id=?",
		a.Name, a.Bio, a.BirthYear, a.Country, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, a)
}

func deleteAuthor(c *gin.Context) {
	id := c.Param("id")
	var bookCount int
	db.QueryRow("SELECT COUNT(*) FROM books WHERE author_id = ?", id).Scan(&bookCount)
	if bookCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete author with existing books", "book_count": bookCount})
		return
	}
	db.Exec("DELETE FROM authors WHERE id = ?", id)
	c.JSON(http.StatusOK, gin.H{"message": "Author deleted"})
}

func getAuthorBooks(c *gin.Context) {
	id := c.Param("id")
	rows, _ := db.Query(`
		SELECT b.id, b.title, b.author_id, a.name, b.isbn, b.price, b.stock, b.published_year, b.description
		FROM books b LEFT JOIN authors a ON b.author_id = a.id WHERE b.author_id=?`, id)
	defer rows.Close()
	books := []BookWithAuthor{}
	for rows.Next() {
		var b BookWithAuthor
		rows.Scan(&b.ID, &b.Title, &b.AuthorID, &b.AuthorName, &b.ISBN, &b.Price, &b.Stock, &b.PublishedYear, &b.Description)
		books = append(books, b)
	}
	c.JSON(http.StatusOK, gin.H{"author_id": id, "books": books, "count": len(books)})
}

// ---------- Book Endpoints ----------

func getBooksPaginated(c *gin.Context) {
	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 20)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit
	var total int
	db.QueryRow("SELECT COUNT(*) FROM books").Scan(&total)

	rows, _ := db.Query(`
	SELECT b.id, b.title, b.author_id, a.name, b.isbn, b.price, b.stock, b.published_year, b.description
	FROM books b LEFT JOIN authors a ON b.author_id = a.id
	ORDER BY b.id LIMIT ? OFFSET ?`, limit, offset)
	defer rows.Close()

	books := []BookWithAuthor{}
	for rows.Next() {
		var b BookWithAuthor
		rows.Scan(&b.ID, &b.Title, &b.AuthorID, &b.AuthorName, &b.ISBN, &b.Price, &b.Stock, &b.PublishedYear, &b.Description)
		books = append(books, b)
	}

	totalPages := (total + limit - 1) / limit
	pagination := PaginationMeta{
		Page: page, Limit: limit, Total: total, TotalPages: totalPages, HasNext: page < totalPages, HasPrev: page > 1,
	}
	c.JSON(http.StatusOK, PaginatedBooksResponse{Books: books, Pagination: pagination})
}

func createBookEnhanced(c *gin.Context) {
	var book Book
	if err := c.ShouldBindJSON(&book); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateISBN(book.ISBN); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validatePublishedYear(book.PublishedYear); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var exists bool
	db.QueryRow("SELECT EXISTS(SELECT 1 FROM authors WHERE id=?)", book.AuthorID).Scan(&exists)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Author ID %d not found", book.AuthorID)})
		return
	}
	res, _ := db.Exec(`INSERT INTO books (title, author_id, isbn, price, stock, published_year, description)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, book.Title, book.AuthorID, book.ISBN, book.Price, book.Stock, book.PublishedYear, book.Description)
	id, _ := res.LastInsertId()
	book.ID = int(id)
	c.JSON(http.StatusCreated, book)
}

// ---------- Inventory Endpoints ----------

func restockBook(c *gin.Context) {
	id := c.Param("id")
	var req RestockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, _ := db.Exec("UPDATE books SET stock = stock + ? WHERE id = ?", req.Quantity, id)
	ra, _ := res.RowsAffected()
	if ra == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Book restocked"})
}

func sellBook(c *gin.Context) {
	id := c.Param("id")
	var req SellRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var stock int
	err := db.QueryRow("SELECT stock FROM books WHERE id=?", id).Scan(&stock)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}
	if stock < req.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient stock", "available": stock})
		return
	}
	db.Exec("UPDATE books SET stock = stock - ? WHERE id=?", req.Quantity, id)
	c.JSON(http.StatusOK, gin.H{"message": "Book sold"})
}

// ---------- Bulk Create ----------

func createBulkBooks(c *gin.Context) {
	var req BulkCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var resp BulkCreateResponse
	for _, book := range req.Books {
		if err := validateISBN(book.ISBN); err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, fmt.Sprintf("%s: %v", book.Title, err))
			continue
		}
		res, err := db.Exec(`INSERT INTO books (title, author_id, isbn, price, stock, published_year, description)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, book.Title, book.AuthorID, book.ISBN, book.Price, book.Stock, book.PublishedYear, book.Description)
		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, fmt.Sprintf("%s: %v", book.Title, err))
			continue
		}
		id, _ := res.LastInsertId()
		book.ID = int(id)
		resp.CreatedBooks = append(resp.CreatedBooks, book)
		resp.Success++
	}
	c.JSON(http.StatusCreated, resp)
}

// ---------- Statistics ----------

func getStatistics(c *gin.Context) {
	var stats Statistics
	db.QueryRow("SELECT COUNT(*) FROM books").Scan(&stats.TotalBooks)
	db.QueryRow("SELECT COUNT(*) FROM authors").Scan(&stats.TotalAuthors)
	db.QueryRow("SELECT SUM(price*stock) FROM books").Scan(&stats.TotalValue)
	db.QueryRow("SELECT COUNT(*) FROM books WHERE stock < 10 AND stock > 0").Scan(&stats.LowStock)
	db.QueryRow("SELECT COUNT(*) FROM books WHERE stock = 0").Scan(&stats.OutOfStock)
	db.QueryRow("SELECT AVG(price) FROM books").Scan(&stats.AveragePrice)

	// Most expensive
	row := db.QueryRow(`
	SELECT b.id, b.title, b.author_id, a.name, b.isbn, b.price, b.stock, b.published_year, b.description
	FROM books b LEFT JOIN authors a ON b.author_id = a.id
	ORDER BY b.price DESC LIMIT 1`)
	var me BookWithAuthor
	row.Scan(&me.ID, &me.Title, &me.AuthorID, &me.AuthorName, &me.ISBN, &me.Price, &me.Stock, &me.PublishedYear, &me.Description)
	stats.MostExpensive = &me

	// Cheapest
	row = db.QueryRow(`
	SELECT b.id, b.title, b.author_id, a.name, b.isbn, b.price, b.stock, b.published_year, b.description
	FROM books b LEFT JOIN authors a ON b.author_id = a.id
	ORDER BY b.price ASC LIMIT 1`)
	var ch BookWithAuthor
	row.Scan(&ch.ID, &ch.Title, &ch.AuthorID, &ch.AuthorName, &ch.ISBN, &ch.Price, &ch.Stock, &ch.PublishedYear, &ch.Description)
	stats.Cheapest = &ch

	// Most stocked
	row = db.QueryRow(`
	SELECT b.id, b.title, b.author_id, a.name, b.isbn, b.price, b.stock, b.published_year, b.description
	FROM books b LEFT JOIN authors a ON b.author_id = a.id
	ORDER BY b.stock DESC LIMIT 1`)
	var ms BookWithAuthor
	row.Scan(&ms.ID, &ms.Title, &ms.AuthorID, &ms.AuthorName, &ms.ISBN, &ms.Price, &ms.Stock, &ms.PublishedYear, &ms.Description)
	stats.MostStocked = &ms

	stats.BooksByYear = make(map[int]int)
	rows, _ := db.Query("SELECT published_year, COUNT(*) FROM books GROUP BY published_year")
	defer rows.Close()
	for rows.Next() {
		var year, count int
		rows.Scan(&year, &count)
		stats.BooksByYear[year] = count
	}

	c.JSON(http.StatusOK, stats)
}

// ---------- API Documentation ----------

func getAPIDocumentation(c *gin.Context) {
	docs := gin.H{"message": "Bookstore API - full endpoints"}
	c.JSON(http.StatusOK, docs)
}

// ---------- Main ----------

func main() {
	initDB()
	router := gin.Default()

	// Authors
	router.GET("/authors", getAuthors)
	router.GET("/authors/:id", getAuthor)
	router.POST("/authors", createAuthor)
	router.PUT("/authors/:id", updateAuthor)
	router.DELETE("/authors/:id", deleteAuthor)
	router.GET("/authors/:id/books", getAuthorBooks)

	// Books
	router.GET("/books", getBooksPaginated)
	router.POST("/books", createBookEnhanced)
	router.POST("/books/:id/restock", restockBook)
	router.POST("/books/:id/sell", sellBook)
	router.POST("/books/bulk", createBulkBooks)

	// Statistics
	router.GET("/stats", getStatistics)

	// Documentation
	router.GET("/", getAPIDocumentation)

	fmt.Println("🚀 Bookstore API running on :8080")
	router.Run(":8080")
}
