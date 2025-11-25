package main

import (
	"database/sql"
	"fmt"
	//"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

// =========================
// ======= STRUCTS =========
// =========================

type PaginationMeta struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

type BookWithAuthor struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	AuthorID      int     `json:"author_id"`
	AuthorName    string  `json:"author_name"`
	ISBN          string  `json:"isbn"`
	Price         float64 `json:"price"`
	Stock         int     `json:"stock"`
	PublishedYear int     `json:"published_year"`
	Description   string  `json:"description"`
}

type PaginatedBooksResponse struct {
	Books      []BookWithAuthor `json:"books"`
	Pagination PaginationMeta   `json:"pagination"`
}

// Enhanced Book struct
type Book struct {
	Title         string  `json:"title" binding:"required,min=3"`
	AuthorID      int     `json:"author_id" binding:"required"`
	ISBN          string  `json:"isbn" binding:"required"`
	Price         float64 `json:"price" binding:"required,min=0.01,max=1000"`
	Stock         int     `json:"stock" binding:"gte=0"`
	PublishedYear int     `json:"published_year"`
	Description   string  `json:"description"`
}

// =========================
// ==== VALIDATION LOGIC ===
// =========================

func validateISBN(isbn string) error {
	isbn = regexp.MustCompile(`-`).ReplaceAllString(isbn, "")

	if len(isbn) != 10 && len(isbn) != 13 {
		return fmt.Errorf("ISBN must be 10 or 13 digits")
	}
	if !regexp.MustCompile(`^[0-9]+$`).MatchString(isbn) {
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

// =========================
// ====== HELPERS ==========
// =========================

func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	valStr := c.Query(key)
	if valStr == "" {
		return defaultValue
	}

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultValue
	}
	return val
}

// =========================
// ==== PAGINATION API =====
// =========================

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

	// Count total books
	var total int
	err := db.QueryRow("SELECT COUNT(*) FROM books").Scan(&total)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to count books"})
		return
	}

	// Query with LIMIT/OFFSET
	query := `
	SELECT b.id, b.title, b.author_id, a.name, 
	       b.isbn, b.price, b.stock, b.published_year, b.description
	FROM books b
	LEFT JOIN authors a ON b.author_id = a.id
	ORDER BY b.id
	LIMIT ? OFFSET ?`
	rows, err := db.Query(query, limit, offset)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch books"})
		return
	}
	defer rows.Close()

	var books []BookWithAuthor
	for rows.Next() {
		var bk BookWithAuthor
		rows.Scan(&bk.ID, &bk.Title, &bk.AuthorID, &bk.AuthorName,
			&bk.ISBN, &bk.Price, &bk.Stock, &bk.PublishedYear, &bk.Description)
		books = append(books, bk)
	}

	totalPages := (total + limit - 1) / limit

	pagination := PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	c.JSON(200, PaginatedBooksResponse{
		Books:      books,
		Pagination: pagination,
	})
}

// =========================
// ===== CREATE BOOK =======
// =========================

func createBookEnhanced(c *gin.Context) {
	var book Book

	if err := c.ShouldBindJSON(&book); err != nil {
		c.JSON(400, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Custom validations
	if err := validateISBN(book.ISBN); err != nil {
		c.JSON(400, gin.H{"error": "Invalid ISBN", "details": err.Error()})
		return
	}

	if err := validatePublishedYear(book.PublishedYear); err != nil {
		c.JSON(400, gin.H{"error": "Invalid published year", "details": err.Error()})
		return
	}

	// Check author exists
	var exists bool
	db.QueryRow("SELECT EXISTS(SELECT 1 FROM authors WHERE id = ?)", book.AuthorID).Scan(&exists)

	if !exists {
		c.JSON(400, gin.H{
			"error":   "Author not found",
			"details": fmt.Sprintf("Author with ID %d does not exist", book.AuthorID),
		})
		return
	}

	// Insert book
	insertQuery := `
	INSERT INTO books (title, author_id, isbn, price, stock, published_year, description)
	VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(insertQuery,
		book.Title, book.AuthorID, book.ISBN, book.Price,
		book.Stock, book.PublishedYear, book.Description)

	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create book"})
		return
	}

	c.JSON(201, gin.H{
		"message": "Book created successfully",
		"book":    book,
	})
}

// =========================
// ========= MAIN ==========
// =========================

func main() {

	var err error
	db, err = sql.Open("mysql", "root:123456@tcp(localhost:3306)/bookstore")
	if err != nil {
		panic(err)
	}

	router := gin.Default()

	// Routes
	router.GET("/books", getBooksPaginated)
	router.POST("/books", createBookEnhanced)

	router.Run(":8080")
}
