package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"

	authorpb "book-catalog-grpc/proto"
	bookpb "book-catalog-grpc/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	_ "modernc.org/sqlite"
)

type authorCatalogServer struct {
	authorpb.UnimplementedAuthorCatalogServer
	db         *sql.DB
	bookClient bookpb.BookCatalogClient // Client to Book service
}

func newServer(db *sql.DB, bookClient bookpb.BookCatalogClient) *authorCatalogServer {
	return &authorCatalogServer{
		db:         db,
		bookClient: bookClient,
	}
}

func (s *authorCatalogServer) GetAuthor(ctx context.Context, req *authorpb.GetAuthorRequest) (*authorpb.GetAuthorResponse, error) {
	log.Printf("GetAuthor: id=%d", req.Id)

	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "author id must be positive")
	}

	var author authorpb.Author
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, bio, birth_year, country FROM authors WHERE id = ?",
		req.Id,
	).Scan(&author.Id, &author.Name, &author.Bio, &author.BirthYear, &author.Country)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "author not found: id=%d", req.Id)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	return &authorpb.GetAuthorResponse{Author: &author}, nil
}

func (s *authorCatalogServer) CreateAuthor(ctx context.Context, req *authorpb.CreateAuthorRequest) (*authorpb.CreateAuthorResponse, error) {
	log.Printf("CreateAuthor: name=%s", req.Name)

	// Validation
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "author name is required")
	}
	if req.BirthYear < 1800 || req.BirthYear > 2100 {
		return nil, status.Error(codes.InvalidArgument, "birth year must be between 1800 and 2100")
	}

	// Insert into database
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO authors (name, bio, birth_year, country) VALUES (?, ?, ?, ?)",
		req.Name, req.Bio, req.BirthYear, req.Country)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert author: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get insert id: %v", err)
	}

	author := &authorpb.Author{
		Id:        int32(id),
		Name:      req.Name,
		Bio:       req.Bio,
		BirthYear: req.BirthYear,
		Country:   req.Country,
	}

	return &authorpb.CreateAuthorResponse{Author: author}, nil
}

func (s *authorCatalogServer) ListAuthors(ctx context.Context, req *authorpb.ListAuthorsRequest) (*authorpb.ListAuthorsResponse, error) {
	log.Printf("ListAuthors: page=%d, page_size=%d", req.Page, req.PageSize)

	// Default values
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 10
	}

	offset := (req.Page - 1) * req.PageSize

	// Query authors with pagination
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, bio, birth_year, country FROM authors LIMIT ? OFFSET ?",
		req.PageSize, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query authors: %v", err)
	}
	defer rows.Close()

	var authors []*authorpb.Author
	for rows.Next() {
		var author authorpb.Author
		if err := rows.Scan(&author.Id, &author.Name, &author.Bio, &author.BirthYear, &author.Country); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan author: %v", err)
		}
		authors = append(authors, &author)
	}

	// Get total count
	var total int32
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM authors").Scan(&total)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count authors: %v", err)
	}

	return &authorpb.ListAuthorsResponse{
		Authors: authors,
		Total:   total,
	}, nil
}

// 🚀 KEY FEATURE: Service-to-Service Communication
func (s *authorCatalogServer) GetAuthorBooks(ctx context.Context, req *authorpb.GetAuthorBooksRequest) (*authorpb.GetAuthorBooksResponse, error) {
	log.Printf("GetAuthorBooks: author_id=%d", req.AuthorId)

	if req.AuthorId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "author_id must be positive")
	}

	// Step 1: Get author from local database
	var author authorpb.Author
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, bio, birth_year, country FROM authors WHERE id = ?",
		req.AuthorId,
	).Scan(&author.Id, &author.Name, &author.Bio, &author.BirthYear, &author.Country)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "author not found: id=%d", req.AuthorId)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	// Step 2: Call Book service to get books by this author
	// This demonstrates MICROSERVICE COMMUNICATION!
	log.Printf("🔄 Calling Book service for author_id=%d", req.AuthorId)
	bookResp, err := s.bookClient.GetBooksByAuthor(ctx, &bookpb.GetBooksByAuthorRequest{
		AuthorId: req.AuthorId,
	})

	if err != nil {
		log.Printf("⚠️ Failed to get books from Book service: %v", err)
		// Continue even if book service fails (graceful degradation)
		return &authorpb.GetAuthorBooksResponse{
			Author:    &author,
			Books:     nil,
			BookCount: 0,
		}, nil
	}

	// Step 3: Convert books to BookSummary
	var bookSummaries []*authorpb.BookSummary
	for _, book := range bookResp.Books {
		bookSummaries = append(bookSummaries, &authorpb.BookSummary{
			Id:            book.Id,
			Title:         book.Title,
			Price:         book.Price,
			PublishedYear: book.PublishedYear,
		})
	}

	log.Printf("✅ Successfully retrieved %d books for author %s", len(bookSummaries), author.Name)

	return &authorpb.GetAuthorBooksResponse{
		Author:    &author,
		Books:     bookSummaries,
		BookCount: int32(len(bookSummaries)),
	}, nil
}

func connectToBookService() (bookpb.BookCatalogClient, error) {
	log.Println("🔗 Connecting to Book service on localhost:50051...")

	conn, err := grpc.Dial("127.0.0.1:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Book service: %w", err)
	}

	log.Println("✅ Successfully connected to Book service")
	return bookpb.NewBookCatalogClient(conn), nil
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "./authors.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create authors table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS authors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		bio TEXT,
		birth_year INTEGER,
		country TEXT
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	// Seed sample authors
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM authors").Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to count authors: %w", err)
	}

	if count == 0 {
		log.Println("Seeding sample authors...")
		sampleAuthors := []struct {
			name, bio, country string
			birthYear          int32
		}{
			{"Alan Donovan", "Go programming expert and co-author of The Go Programming Language", "USA", 1975},
			{"Robert C. Martin", "Software development expert, author of Clean Code", "USA", 1952},
			{"Gang of Four", "Authors of the seminal Design Patterns book", "International", 1960},
			{"Joshua Bloch", "Java expert and author of Effective Java", "USA", 1961},
			{"Mat Ryer", "Go developer and author of Go Programming Blueprints", "UK", 1978},
			{"Hunt & Thomas", "Authors of The Pragmatic Programmer", "USA", 1965},
		}

		for _, author := range sampleAuthors {
			_, err := db.Exec(
				"INSERT INTO authors (name, bio, birth_year, country) VALUES (?, ?, ?, ?)",
				author.name, author.bio, author.birthYear, author.country)
			if err != nil {
				return nil, fmt.Errorf("failed to seed data: %w", err)
			}
		}
		log.Println("Sample authors seeded successfully")
	}

	log.Println("Database initialized successfully")
	return db, nil
}

func main() {
	// Step 1: Initialize database
	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	// Step 2: Connect to Book service (service-to-service communication)
	bookClient, err := connectToBookService()
	if err != nil {
		log.Fatalf("Failed to connect to Book service: %v", err)
	}

	// Step 3: Create listener on port 50052 (Author service)
	lis, err := net.Listen("tcp", "0.0.0.0:50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Step 4: Create gRPC server
	grpcServer := grpc.NewServer()

	// Step 5: Register service with book client for cross-service calls
	authorpb.RegisterAuthorCatalogServer(grpcServer, newServer(db, bookClient))

	log.Println("🚀 Author Catalog gRPC server listening on :50052")
	log.Println("📚 Connected to Book Catalog service on :50051")
	log.Println("✨ Service-to-service communication enabled!")

	// Step 6: Start serving
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
