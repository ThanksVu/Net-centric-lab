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

	// Query book từ database
	var book pb.Book
	err := s.db.QueryRowContext(ctx,
		"SELECT id, title, author, isbn, price, stock, published_year FROM books WHERE id = ?",
		req.Id).Scan(&book.Id, &book.Title, &book.Author, &book.Isbn, &book.Price, &book.Stock, &book.PublishedYear)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "book with id %d not found", req.Id)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	return &pb.GetBookResponse{Book: &book}, nil
}

func (s *bookCatalogServer) CreateBook(ctx context.Context, req *pb.CreateBookRequest) (*pb.CreateBookResponse, error) {
	log.Printf("CreateBook: title=%s, author=%s", req.Title, req.Author)

	// Validate input
	if req.Title == "" || req.Author == "" {
		return nil, status.Error(codes.InvalidArgument, "title and author are required")
	}
	if req.Price < 0 {
		return nil, status.Error(codes.InvalidArgument, "price must be positive")
	}

	// Insert vào database
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO books (title, author, isbn, price, stock, published_year) VALUES (?, ?, ?, ?, ?, ?)",
		req.Title, req.Author, req.Isbn, req.Price, req.Stock, req.PublishedYear)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert book: %v", err)
	}

	// Lấy ID vừa tạo
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
	}

	return &pb.CreateBookResponse{Book: book}, nil
}

func (s *bookCatalogServer) UpdateBook(ctx context.Context, req *pb.UpdateBookRequest) (*pb.UpdateBookResponse, error) {
	log.Printf("UpdateBook: id=%d, title=%s", req.Id, req.Title)

	// Validate input
	if req.Title == "" || req.Author == "" {
		return nil, status.Error(codes.InvalidArgument, "title and author are required")
	}
	if req.Price < 0 {
		return nil, status.Error(codes.InvalidArgument, "price must be positive")
	}

	// Update trong database
	result, err := s.db.ExecContext(ctx,
		"UPDATE books SET title=?, author=?, isbn=?, price=?, stock=?, published_year=? WHERE id=?",
		req.Title, req.Author, req.Isbn, req.Price, req.Stock, req.PublishedYear, req.Id)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update book: %v", err)
	}

	// Kiểm tra xem có update được không
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
	}

	return &pb.UpdateBookResponse{Book: book}, nil
}

func (s *bookCatalogServer) DeleteBook(ctx context.Context, req *pb.DeleteBookRequest) (*pb.DeleteBookResponse, error) {
	log.Printf("DeleteBook: id=%d", req.Id)

	// Delete từ database
	result, err := s.db.ExecContext(ctx, "DELETE FROM books WHERE id = ?", req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete book: %v", err)
	}

	// Kiểm tra xem có xóa được không
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

	// Set default values cho pagination
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 10
	}

	// Tính offset
	offset := (page - 1) * pageSize

	// Lấy tổng số sách
	var total int32
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM books").Scan(&total)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count books: %v", err)
	}

	// Query sách với LIMIT và OFFSET
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, title, author, isbn, price, stock, published_year FROM books ORDER BY id LIMIT ? OFFSET ?",
		pageSize, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query books: %v", err)
	}
	defer rows.Close()

	// Đọc kết quả
	var books []*pb.Book
	for rows.Next() {
		var book pb.Book
		err := rows.Scan(&book.Id, &book.Title, &book.Author, &book.Isbn, &book.Price, &book.Stock, &book.PublishedYear)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan book: %v", err)
		}
		books = append(books, &book)
	}

	if err = rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "rows error: %v", err)
	}

	return &pb.ListBooksResponse{
		Books:    books,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "./books.db")
	if err != nil {
		return nil, err
	}

	// Tạo bảng books
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		isbn TEXT,
		price REAL NOT NULL,
		stock INTEGER NOT NULL,
		published_year INTEGER
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %v", err)
	}

	// Kiểm tra xem đã có dữ liệu chưa
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM books").Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to count books: %v", err)
	}

	// Nếu chưa có dữ liệu thì seed
	if count == 0 {
		log.Println("Seeding sample data...")
		sampleBooks := []struct {
			title, author, isbn string
			price               float32
			stock, year         int32
		}{
			{"The Go Programming Language", "Alan Donovan", "978-0134190440", 39.99, 15, 2015},
			{"Clean Code", "Robert Martin", "978-0132350884", 42.50, 20, 2008},
			{"Design Patterns", "Gang of Four", "978-0201633610", 54.99, 10, 1994},
			{"The Pragmatic Programmer", "Andy Hunt", "978-0135957059", 45.00, 12, 2019},
			{"Introduction to Algorithms", "Thomas Cormen", "978-0262033848", 89.99, 8, 2009},
			{"Effective Go", "The Go Team", "978-1234567890", 29.99, 25, 2020},
		}

		for _, book := range sampleBooks {
			_, err := db.Exec(
				"INSERT INTO books (title, author, isbn, price, stock, published_year) VALUES (?, ?, ?, ?, ?, ?)",
				book.title, book.author, book.isbn, book.price, book.stock, book.year)
			if err != nil {
				return nil, fmt.Errorf("failed to insert sample book: %v", err)
			}
		}
		log.Println("Sample data seeded successfully")
	}

	return db, nil
}

func (s *bookCatalogServer) SearchBooks(ctx context.Context, req *pb.SearchBooksRequest) (*pb.SearchBooksResponse, error) {
	log.Printf("SearchBooks: query=%s, field=%s", req.Query, req.Field)

	// Validate input
	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "search query required")
	}

	// Build SQL query based on field
	var sqlQuery string
	var args []interface{}
	searchPattern := "%" + req.Query + "%"

	switch req.Field {
	case "title":
		sqlQuery = "SELECT id, title, author, isbn, price, stock, published_year FROM books WHERE title LIKE ?"
		args = []interface{}{searchPattern}
	case "author":
		sqlQuery = "SELECT id, title, author, isbn, price, stock, published_year FROM books WHERE author LIKE ?"
		args = []interface{}{searchPattern}
	case "isbn":
		// ISBN search - exact match
		sqlQuery = "SELECT id, title, author, isbn, price, stock, published_year FROM books WHERE isbn = ?"
		args = []interface{}{req.Query}
	case "all", "":
		// Search in all fields
		sqlQuery = `SELECT id, title, author, isbn, price, stock, published_year 
		            FROM books 
		            WHERE title LIKE ? OR author LIKE ? OR isbn LIKE ?`
		args = []interface{}{searchPattern, searchPattern, searchPattern}
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid field: %s (must be 'title', 'author', 'isbn', or 'all')", req.Field)
	}

	// Execute query
	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search books: %v", err)
	}
	defer rows.Close()

	// Scan results
	var books []*pb.Book
	for rows.Next() {
		var book pb.Book
		err := rows.Scan(&book.Id, &book.Title, &book.Author, &book.Isbn, &book.Price, &book.Stock, &book.PublishedYear)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan book: %v", err)
		}
		books = append(books, &book)
	}

	if err = rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "rows error: %v", err)
	}

	return &pb.SearchBooksResponse{
		Books: books,
		Count: int32(len(books)),
		Query: req.Query,
	}, nil
}

func (s *bookCatalogServer) FilterBooks(ctx context.Context, req *pb.FilterBooksRequest) (*pb.FilterBooksResponse, error) {
	log.Printf("FilterBooks: price[%.2f-%.2f], year[%d-%d]",
		req.MinPrice, req.MaxPrice, req.MinYear, req.MaxYear)

	// Validate ranges
	if req.MinPrice < 0 || req.MaxPrice < 0 {
		return nil, status.Error(codes.InvalidArgument, "price cannot be negative")
	}
	if req.MinPrice > 0 && req.MaxPrice > 0 && req.MinPrice > req.MaxPrice {
		return nil, status.Error(codes.InvalidArgument, "min_price cannot be greater than max_price")
	}
	if req.MinYear > 0 && req.MaxYear > 0 && req.MinYear > req.MaxYear {
		return nil, status.Error(codes.InvalidArgument, "min_year cannot be greater than max_year")
	}

	// Build dynamic query
	query := "SELECT id, title, author, isbn, price, stock, published_year FROM books WHERE 1=1"
	var args []interface{}

	// Add price filters
	if req.MinPrice > 0 {
		query += " AND price >= ?"
		args = append(args, req.MinPrice)
	}
	if req.MaxPrice > 0 {
		query += " AND price <= ?"
		args = append(args, req.MaxPrice)
	}

	// Add year filters
	if req.MinYear > 0 {
		query += " AND published_year >= ?"
		args = append(args, req.MinYear)
	}
	if req.MaxYear > 0 {
		query += " AND published_year <= ?"
		args = append(args, req.MaxYear)
	}

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to filter books: %v", err)
	}
	defer rows.Close()

	// Scan results
	var books []*pb.Book
	for rows.Next() {
		var book pb.Book
		err := rows.Scan(&book.Id, &book.Title, &book.Author, &book.Isbn, &book.Price, &book.Stock, &book.PublishedYear)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan book: %v", err)
		}
		books = append(books, &book)
	}

	if err = rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "rows error: %v", err)
	}

	return &pb.FilterBooksResponse{
		Books: books,
		Count: int32(len(books)),
	}, nil
}

func (s *bookCatalogServer) GetStats(ctx context.Context, req *pb.GetStatsRequest) (*pb.GetStatsResponse, error) {
	log.Println("GetStats called")

	var stats pb.GetStatsResponse

	// Get total books
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM books").Scan(&stats.TotalBooks)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count books: %v", err)
	}

	// Get average price
	var avgPrice sql.NullFloat64
	err = s.db.QueryRowContext(ctx, "SELECT AVG(price) FROM books").Scan(&avgPrice)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to calculate average price: %v", err)
	}
	if avgPrice.Valid {
		stats.AveragePrice = float32(avgPrice.Float64)
	}

	// Get total stock
	err = s.db.QueryRowContext(ctx, "SELECT SUM(stock) FROM books").Scan(&stats.TotalStock)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to sum stock: %v", err)
	}

	// Get earliest and latest year
	var minYear, maxYear sql.NullInt32
	err = s.db.QueryRowContext(ctx, "SELECT MIN(published_year), MAX(published_year) FROM books").Scan(&minYear, &maxYear)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get year range: %v", err)
	}
	if minYear.Valid {
		stats.EarliestYear = minYear.Int32
	}
	if maxYear.Valid {
		stats.LatestYear = maxYear.Int32
	}

	return &stats, nil
}

func main() {
	// Khởi tạo database
	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("Database initialized successfully")

	// Tạo listener trên port 50053 (Task4)
	lis, err := net.Listen("tcp", "0.0.0.0:50053")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Tạo gRPC server
	grpcServer := grpc.NewServer()

	// Register service
	pb.RegisterBookCatalogServer(grpcServer, &bookCatalogServer{db: db})

	log.Println("📚 BookCatalog gRPC server (Task4) listening on :50053")

	// Start serving
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
