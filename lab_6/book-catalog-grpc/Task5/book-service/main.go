package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"

	pb "book-catalog-grpc/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	_ "modernc.org/sqlite"
)

type bookCatalogServer struct {
	pb.UnimplementedBookCatalogServer
	db *sql.DB
}

func (s *bookCatalogServer) GetBook(ctx context.Context, req *pb.GetBookRequest) (*pb.GetBookResponse, error) {
	log.Printf("GetBook: id=%d", req.Id)

	var book pb.Book
	err := s.db.QueryRowContext(ctx,
		"SELECT id, title, author, isbn, price, stock, published_year, author_id FROM books WHERE id = ?",
		req.Id).Scan(&book.Id, &book.Title, &book.Author, &book.Isbn, &book.Price, &book.Stock, &book.PublishedYear, &book.AuthorId)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "book with id %d not found", req.Id)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	return &pb.GetBookResponse{Book: &book}, nil
}

func (s *bookCatalogServer) CreateBook(ctx context.Context, req *pb.CreateBookRequest) (*pb.CreateBookResponse, error) {
	log.Printf("CreateBook: title=%s, author=%s, author_id=%d", req.Title, req.Author, req.AuthorId)

	if req.Title == "" || req.Author == "" {
		return nil, status.Error(codes.InvalidArgument, "title and author are required")
	}
	if req.Price < 0 {
		return nil, status.Error(codes.InvalidArgument, "price must be positive")
	}

	result, err := s.db.ExecContext(ctx,
		"INSERT INTO books (title, author, isbn, price, stock, published_year, author_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
		req.Title, req.Author, req.Isbn, req.Price, req.Stock, req.PublishedYear, req.AuthorId)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert book: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get insert id: %v", err)
	}

	book := &pb.Book{
		Id:            int32(id),
		Title:         req.Title,
		Author:        req.Author,
		Isbn:          req.Isbn,
		Price:         req.Price,
		Stock:         req.Stock,
		PublishedYear: req.PublishedYear,
		AuthorId:      req.AuthorId,
	}

	return &pb.CreateBookResponse{Book: book}, nil
}

func (s *bookCatalogServer) UpdateBook(ctx context.Context, req *pb.UpdateBookRequest) (*pb.UpdateBookResponse, error) {
	log.Printf("UpdateBook: id=%d, title=%s", req.Id, req.Title)

	if req.Title == "" || req.Author == "" {
		return nil, status.Error(codes.InvalidArgument, "title and author are required")
	}
	if req.Price < 0 {
		return nil, status.Error(codes.InvalidArgument, "price must be positive")
	}

	result, err := s.db.ExecContext(ctx,
		"UPDATE books SET title=?, author=?, isbn=?, price=?, stock=?, published_year=?, author_id=? WHERE id=?",
		req.Title, req.Author, req.Isbn, req.Price, req.Stock, req.PublishedYear, req.AuthorId, req.Id)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update book: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "book with id %d not found", req.Id)
	}

	book := &pb.Book{
		Id:            req.Id,
		Title:         req.Title,
		Author:        req.Author,
		Isbn:          req.Isbn,
		Price:         req.Price,
		Stock:         req.Stock,
		PublishedYear: req.PublishedYear,
		AuthorId:      req.AuthorId,
	}

	return &pb.UpdateBookResponse{Book: book}, nil
}

func (s *bookCatalogServer) DeleteBook(ctx context.Context, req *pb.DeleteBookRequest) (*pb.DeleteBookResponse, error) {
	log.Printf("DeleteBook: id=%d", req.Id)

	result, err := s.db.ExecContext(ctx, "DELETE FROM books WHERE id = ?", req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete book: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "book with id %d not found", req.Id)
	}

	return &pb.DeleteBookResponse{
		Success: true,
		Message: fmt.Sprintf("Book with id %d deleted successfully", req.Id),
	}, nil
}

func (s *bookCatalogServer) ListBooks(ctx context.Context, req *pb.ListBooksRequest) (*pb.ListBooksResponse, error) {
	log.Printf("ListBooks: page=%d, page_size=%d", req.Page, req.PageSize)

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 10
	}

	offset := (req.Page - 1) * req.PageSize

	rows, err := s.db.QueryContext(ctx,
		"SELECT id, title, author, isbn, price, stock, published_year, author_id FROM books LIMIT ? OFFSET ?",
		req.PageSize, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query books: %v", err)
	}
	defer rows.Close()

	var books []*pb.Book
	for rows.Next() {
		var book pb.Book
		if err := rows.Scan(&book.Id, &book.Title, &book.Author, &book.Isbn, &book.Price, &book.Stock, &book.PublishedYear, &book.AuthorId); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan book: %v", err)
		}
		books = append(books, &book)
	}

	var total int32
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM books").Scan(&total)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count books: %v", err)
	}

	return &pb.ListBooksResponse{
		Books:    books,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (s *bookCatalogServer) SearchBooks(ctx context.Context, req *pb.SearchBooksRequest) (*pb.SearchBooksResponse, error) {
	log.Printf("SearchBooks: query=%s, field=%s", req.Query, req.Field)

	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "search query is required")
	}

	var query string
	var args []interface{}

	switch req.Field {
	case "title":
		query = "SELECT id, title, author, isbn, price, stock, published_year, author_id FROM books WHERE title LIKE ?"
		args = append(args, "%"+req.Query+"%")
	case "author":
		query = "SELECT id, title, author, isbn, price, stock, published_year, author_id FROM books WHERE author LIKE ?"
		args = append(args, "%"+req.Query+"%")
	case "isbn":
		query = "SELECT id, title, author, isbn, price, stock, published_year, author_id FROM books WHERE isbn = ?"
		args = append(args, req.Query)
	case "all":
		query = "SELECT id, title, author, isbn, price, stock, published_year, author_id FROM books WHERE title LIKE ? OR author LIKE ? OR isbn LIKE ?"
		args = append(args, "%"+req.Query+"%", "%"+req.Query+"%", "%"+req.Query+"%")
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid field, must be title, author, isbn, or all")
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search books: %v", err)
	}
	defer rows.Close()

	var books []*pb.Book
	for rows.Next() {
		var book pb.Book
		if err := rows.Scan(&book.Id, &book.Title, &book.Author, &book.Isbn, &book.Price, &book.Stock, &book.PublishedYear, &book.AuthorId); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan book: %v", err)
		}
		books = append(books, &book)
	}

	return &pb.SearchBooksResponse{
		Books: books,
		Count: int32(len(books)),
		Query: req.Query,
	}, nil
}

func (s *bookCatalogServer) FilterBooks(ctx context.Context, req *pb.FilterBooksRequest) (*pb.FilterBooksResponse, error) {
	log.Printf("FilterBooks: price[%.2f-%.2f], year[%d-%d]", req.MinPrice, req.MaxPrice, req.MinYear, req.MaxYear)

	if req.MinPrice < 0 || req.MaxPrice < 0 {
		return nil, status.Error(codes.InvalidArgument, "price cannot be negative")
	}
	if req.MinPrice > 0 && req.MaxPrice > 0 && req.MinPrice > req.MaxPrice {
		return nil, status.Error(codes.InvalidArgument, "min_price cannot be greater than max_price")
	}
	if req.MinYear > 0 && req.MaxYear > 0 && req.MinYear > req.MaxYear {
		return nil, status.Error(codes.InvalidArgument, "min_year cannot be greater than max_year")
	}

	query := "SELECT id, title, author, isbn, price, stock, published_year, author_id FROM books WHERE 1=1"
	var args []interface{}

	if req.MinPrice > 0 {
		query += " AND price >= ?"
		args = append(args, req.MinPrice)
	}
	if req.MaxPrice > 0 {
		query += " AND price <= ?"
		args = append(args, req.MaxPrice)
	}
	if req.MinYear > 0 {
		query += " AND published_year >= ?"
		args = append(args, req.MinYear)
	}
	if req.MaxYear > 0 {
		query += " AND published_year <= ?"
		args = append(args, req.MaxYear)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to filter books: %v", err)
	}
	defer rows.Close()

	var books []*pb.Book
	for rows.Next() {
		var book pb.Book
		if err := rows.Scan(&book.Id, &book.Title, &book.Author, &book.Isbn, &book.Price, &book.Stock, &book.PublishedYear, &book.AuthorId); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan book: %v", err)
		}
		books = append(books, &book)
	}

	return &pb.FilterBooksResponse{
		Books: books,
		Count: int32(len(books)),
	}, nil
}

func (s *bookCatalogServer) GetStats(ctx context.Context, req *pb.GetStatsRequest) (*pb.GetStatsResponse, error) {
	log.Printf("GetStats")

	var totalBooks int32
	var avgPrice sql.NullFloat64
	var totalStock int32
	var minYear, maxYear sql.NullInt32

	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM books").Scan(&totalBooks)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count books: %v", err)
	}

	err = s.db.QueryRowContext(ctx, "SELECT AVG(price) FROM books").Scan(&avgPrice)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to calculate average price: %v", err)
	}

	err = s.db.QueryRowContext(ctx, "SELECT SUM(stock) FROM books").Scan(&totalStock)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to sum stock: %v", err)
	}

	err = s.db.QueryRowContext(ctx, "SELECT MIN(published_year), MAX(published_year) FROM books").Scan(&minYear, &maxYear)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get year range: %v", err)
	}

	resp := &pb.GetStatsResponse{
		TotalBooks: totalBooks,
		TotalStock: totalStock,
	}

	if avgPrice.Valid {
		resp.AveragePrice = float32(avgPrice.Float64)
	}
	if minYear.Valid {
		resp.EarliestYear = minYear.Int32
	}
	if maxYear.Valid {
		resp.LatestYear = maxYear.Int32
	}

	return resp, nil
}

// NEW: Get books by author_id - for service-to-service communication
func (s *bookCatalogServer) GetBooksByAuthor(ctx context.Context, req *pb.GetBooksByAuthorRequest) (*pb.GetBooksByAuthorResponse, error) {
	log.Printf("GetBooksByAuthor: author_id=%d", req.AuthorId)

	if req.AuthorId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "author_id must be positive")
	}

	rows, err := s.db.QueryContext(ctx,
		"SELECT id, title, author, isbn, price, stock, published_year, author_id FROM books WHERE author_id = ?",
		req.AuthorId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query books: %v", err)
	}
	defer rows.Close()

	var books []*pb.Book
	for rows.Next() {
		var book pb.Book
		if err := rows.Scan(&book.Id, &book.Title, &book.Author, &book.Isbn, &book.Price, &book.Stock, &book.PublishedYear, &book.AuthorId); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan book: %v", err)
		}
		books = append(books, &book)
	}

	return &pb.GetBooksByAuthorResponse{
		Books: books,
		Count: int32(len(books)),
	}, nil
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "./books_task5.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create books table with author_id
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		isbn TEXT,
		price REAL,
		stock INTEGER,
		published_year INTEGER,
		author_id INTEGER DEFAULT 0
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	// Seed sample data
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM books").Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to count books: %w", err)
	}

	if count == 0 {
		log.Println("Seeding sample data...")
		sampleBooks := []struct {
			title, author, isbn            string
			price                          float32
			stock, publishedYear, authorId int32
		}{
			{"The Go Programming Language", "Alan Donovan", "978-0134190440", 44.99, 20, 2015, 1},
			{"Clean Code", "Robert C. Martin", "978-0132350884", 39.95, 15, 2008, 2},
			{"Design Patterns", "Gang of Four", "978-0201633610", 54.99, 10, 1994, 3},
			{"Effective Java", "Joshua Bloch", "978-0134685991", 42.50, 25, 2017, 4},
			{"Go Programming Blueprints", "Mat Ryer", "978-1783988020", 29.99, 18, 2015, 5},
			{"The Pragmatic Programmer", "Hunt & Thomas", "978-0135957059", 45.00, 22, 2019, 6},
		}

		for _, book := range sampleBooks {
			_, err := db.Exec(
				"INSERT INTO books (title, author, isbn, price, stock, published_year, author_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
				book.title, book.author, book.isbn, book.price, book.stock, book.publishedYear, book.authorId)
			if err != nil {
				return nil, fmt.Errorf("failed to seed data: %w", err)
			}
		}
		log.Println("Sample data seeded successfully")
	}

	log.Println("Database initialized successfully")
	return db, nil
}

func main() {
	// Initialize database
	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create listener on port 50051 (Book service)
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterBookCatalogServer(grpcServer, &bookCatalogServer{db: db})

	log.Println("📚 BookCatalog gRPC server (Task5) listening on :50051")
	log.Println("✨ Supports service-to-service communication with Author service")

	// Start serving
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
