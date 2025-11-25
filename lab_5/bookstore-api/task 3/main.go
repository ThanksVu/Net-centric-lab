package main

import (
	"database/sql"
	"log"
	"net/http"
	//"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

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
	Title         string  `json:"title"`
	AuthorID      int     `json:"author_id"`
	ISBN          string  `json:"isbn"`
	Price         float64 `json:"price"`
	Stock         int     `json:"stock"`
	PublishedYear int     `json:"published_year"`
	Description   string  `json:"description"`
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

func initDB() error {
	var err error
	db, err = sql.Open("sqlite3", "./bookstore.db")
	if err != nil {
		return err
	}

	createBooksSQL := `
	CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		author_id INTEGER,
		isbn TEXT,
		price REAL,
		stock INTEGER DEFAULT 0,
		published_year INTEGER,
		description TEXT
	);
	`
	_, err = db.Exec(createBooksSQL)
	if err != nil {
		return err
	}

	createAuthorsSQL := `
	CREATE TABLE IF NOT EXISTS authors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		bio TEXT,
		birth_year INTEGER,
		country TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(createAuthorsSQL)
	if err != nil {
		return err
	}

	return nil
}

func seedAuthors() {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM authors").Scan(&count)
	if count > 0 {
		return
	}

	authors := []Author{
		{Name: "Robert C. Martin", Bio: "Software engineer and author", BirthYear: 1952, Country: "USA"},
		{Name: "Martin Fowler", Bio: "Software development expert", BirthYear: 1963, Country: "UK"},
		{Name: "Alan Donovan", Bio: "Go team member at Google", BirthYear: 0, Country: "USA"},
	}

	for _, a := range authors {
		_, _ = db.Exec(`
			INSERT INTO authors (name, bio, birth_year, country) 
			VALUES (?, ?, ?, ?)
		`, a.Name, a.Bio, a.BirthYear, a.Country)
	}
}

//
// AUTHORS API
//

func getAuthors(c *gin.Context) {
	rows, err := db.Query("SELECT id, name, bio, birth_year, country, created_at FROM authors")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var authors []Author
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
	err := db.QueryRow(`
		SELECT id, name, bio, birth_year, country, created_at
		FROM authors WHERE id = ?
	`, id).Scan(&a.ID, &a.Name, &a.Bio, &a.BirthYear, &a.Country, &a.CreatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Author not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	_, err := db.Exec(`
		INSERT INTO authors (name, bio, birth_year, country)
		VALUES (?, ?, ?, ?)
	`, a.Name, a.Bio, a.BirthYear, a.Country)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Author name must be unique"})
		return
	}

	c.JSON(http.StatusCreated, a)
}

func updateAuthor(c *gin.Context) {
	id := c.Param("id")
	var a Author

	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(`
		UPDATE authors 
		SET name = ?, bio = ?, birth_year = ?, country = ?
		WHERE id = ?
	`, a.Name, a.Bio, a.BirthYear, a.Country, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Author updated"})
}

func deleteAuthor(c *gin.Context) {
	id := c.Param("id")

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM books WHERE author_id = ?", id).Scan(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Cannot delete author with existing books",
			"book_count": count,
		})
		return
	}

	_, err := db.Exec("DELETE FROM authors WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Author deleted"})
}

func getAuthorBooks(c *gin.Context) {
	authorID := c.Param("id")

	var a Author
	err := db.QueryRow("SELECT id, name, bio, birth_year, country FROM authors WHERE id = ?", authorID).
		Scan(&a.ID, &a.Name, &a.Bio, &a.BirthYear, &a.Country)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Author not found"})
		return
	}

	rows, err := db.Query(`
		SELECT id, title, author_id, isbn, price, stock, published_year, description
		FROM books WHERE author_id = ?
	`, authorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		rows.Scan(&b.ID, &b.Title, &b.AuthorID, &b.ISBN, &b.Price, &b.Stock, &b.PublishedYear, &b.Description)
		books = append(books, b)
	}

	c.JSON(http.StatusOK, gin.H{
		"author": a,
		"books":  books,
		"count":  len(books),
	})
}

//
// BOOKS API WITH JOIN
//

func getBooksWithAuthors(c *gin.Context) {
	rows, err := db.Query(`
		SELECT b.id, b.title, b.author_id, a.name, 
		       b.isbn, b.price, b.stock, b.published_year, b.description
		FROM books b
		LEFT JOIN authors a ON b.author_id = a.id
		ORDER BY b.id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []BookWithAuthor
	for rows.Next() {
		var b BookWithAuthor
		rows.Scan(&b.ID, &b.Title, &b.AuthorID, &b.AuthorName,
			&b.ISBN, &b.Price, &b.Stock, &b.PublishedYear, &b.Description)
		list = append(list, b)
	}

	c.JSON(http.StatusOK, list)
}

func createBook(c *gin.Context) {
	var b Book
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(`
		INSERT INTO books (title, author_id, isbn, price, stock, published_year, description)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, b.Title, b.AuthorID, b.ISBN, b.Price, b.Stock, b.PublishedYear, b.Description)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, b)
}

//
// MAIN SETUP
//

func main() {
	if err := initDB(); err != nil {
		log.Fatal("DB ERROR: ", err)
	}

	seedAuthors()

	router := gin.Default()

	// Author routes
	router.GET("/authors", getAuthors)
	router.GET("/authors/:id", getAuthor)
	router.POST("/authors", createAuthor)
	router.PUT("/authors/:id", updateAuthor)
	router.DELETE("/authors/:id", deleteAuthor)
	router.GET("/authors/:id/books", getAuthorBooks)

	// Book routes with JOIN
	router.GET("/books", getBooksWithAuthors)
	router.POST("/books", createBook)

	router.Run(":8080")
}
