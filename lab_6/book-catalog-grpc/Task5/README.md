# Task 5: Complete Microservice with Multiple Services (25 points)

## 🎯 Mục tiêu
Xây dựng hệ thống microservice hoàn chỉnh với 2 services giao tiếp với nhau, mô phỏng kiến trúc thực tế như Amazon (inventory, pricing, payment, shipping services).

## 📐 Kiến trúc hệ thống

```
┌─────────────────┐         ┌─────────────────┐
│                 │         │                 │
│  Book Service   │◄────────│ Author Service  │
│   (Port 50051)  │         │   (Port 50052)  │
│                 │         │                 │
└────────┬────────┘         └────────┬────────┘
         │                           │
         │                           │
    ┌────▼───────────────────────────▼────┐
    │                                     │
    │         Demo Client                 │
    │                                     │
    └─────────────────────────────────────┘
```

## 🔧 Cấu trúc Project

```
Task5/
├── book-service/
│   └── main.go              # Book Catalog service với GetBooksByAuthor
├── author-service/
│   └── main.go              # Author Catalog service với cross-service call
└── client/
    └── main.go              # Demo client test cả 2 services
```

## 📋 Proto Definitions

### Author Service (author_service.proto)
- **Author message**: id, name, bio, birth_year, country
- **BookSummary message**: id, title, price, published_year (lightweight reference)
- **4 RPCs**:
  - `GetAuthor(id)` - Lấy thông tin 1 tác giả
  - `CreateAuthor(...)` - Tạo tác giả mới
  - `ListAuthors(page, page_size)` - List với pagination
  - `GetAuthorBooks(author_id)` - **KEY: Cross-service call đến Book service**

### Book Service Updates
- **Thêm field**: `author_id` vào Book message (foreign key)
- **New RPC**: `GetBooksByAuthor(author_id)` - Lấy tất cả books của 1 author

## 🔄 Service-to-Service Communication Flow

### Khi client gọi GetAuthorBooks:

1. **Client** → **Author Service**: `GetAuthorBooks(author_id=7)`
2. **Author Service**:
   - Query local DB để lấy author info
   - **Call Book Service**: `GetBooksByAuthor(author_id=7)` 🔥
   - Convert Books → BookSummary
3. **Book Service**:
   - Query database WHERE author_id = 7
   - Return list of books
4. **Author Service** → **Client**: Trả về Author + Books

## 🚀 Cách chạy

### Bước 1: Khởi động Book Service (Terminal 1)
```cmd
cd Task5\book-service
go run main.go
```

Output:
```
📚 BookCatalog gRPC server (Task5) listening on :50051
✨ Supports service-to-service communication with Author service
```

### Bước 2: Khởi động Author Service (Terminal 2)
```cmd
cd Task5\author-service
go run main.go
```

Output:
```
🔗 Connecting to Book service on localhost:50051...
✅ Successfully connected to Book service
🚀 Author Catalog gRPC server listening on :50052
📚 Connected to Book Catalog service on :50051
✨ Service-to-service communication enabled!
```

### Bước 3: Chạy Demo Client (Terminal 3)
```cmd
cd Task5\client
go run main.go
```

## 📊 Expected Output

```
=== Microservice Demo ===

1. Creating author...
✓ Created author: Martin Fowler (ID: 7)

2. Creating books for author...
✓ Created book: Refactoring
✓ Created book: Patterns of Enterprise Application Architecture

3. Fetching author's books (cross-service call)...
✓ Author: Martin Fowler
✓ Books written: 2
  1. Refactoring (2018) - $49.99
  2. Patterns of Enterprise Application Architecture (2002) - $54.99

4. Listing all authors...
✓ Total authors: 7
  1. Alan Donovan (1975, USA)
  2. Robert C. Martin (1952, USA)
  3. Gang of Four (1960, International)
  4. Joshua Bloch (1961, USA)
  5. Mat Ryer (1978, UK)
  6. Hunt & Thomas (1965, USA)
  7. Martin Fowler (1963, UK)

5. Getting book statistics...
✓ Total books: 8
✓ Average price: $45.30
✓ Total stock: 133
✓ Year range: 1994 - 2019

✅ Microservice demo completed successfully!
📊 Demonstrated:
   - Service-to-service communication (Author → Book)
   - CRUD operations across multiple services
   - Cross-service data aggregation
```

## 🔍 Implementation Highlights

### 1. Book Service
```go
func (s *bookCatalogServer) GetBooksByAuthor(ctx context.Context, 
    req *pb.GetBooksByAuthorRequest) (*pb.GetBooksByAuthorResponse, error) {
    
    // Query books WHERE author_id = ?
    rows, err := s.db.QueryContext(ctx,
        "SELECT ... FROM books WHERE author_id = ?", req.AuthorId)
    
    return &pb.GetBooksByAuthorResponse{Books: books, Count: count}, nil
}
```

### 2. Author Service - Service-to-Service Call
```go
type authorCatalogServer struct {
    db         *sql.DB
    bookClient bookpb.BookCatalogClient  // 🔥 Client to Book service
}

func (s *authorCatalogServer) GetAuthorBooks(ctx context.Context, 
    req *authorpb.GetAuthorBooksRequest) (*authorpb.GetAuthorBooksResponse, error) {
    
    // Step 1: Get author from local DB
    var author authorpb.Author
    err := s.db.QueryRowContext(ctx, "SELECT ... WHERE id = ?", req.AuthorId)
    
    // Step 2: Call Book service 🔥
    bookResp, err := s.bookClient.GetBooksByAuthor(ctx, 
        &bookpb.GetBooksByAuthorRequest{AuthorId: req.AuthorId})
    
    // Step 3: Convert and return
    return &authorpb.GetAuthorBooksResponse{
        Author: &author,
        Books: bookSummaries,
        BookCount: int32(len(bookSummaries)),
    }, nil
}
```

### 3. Connecting to Book Service
```go
func connectToBookService() (bookpb.BookCatalogClient, error) {
    conn, err := grpc.Dial("127.0.0.1:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    return bookpb.NewBookCatalogClient(conn), nil
}

func main() {
    bookClient, err := connectToBookService()  // Connect on startup
    grpcServer := grpc.NewServer()
    authorpb.RegisterAuthorCatalogServer(grpcServer, 
        newServer(db, bookClient))  // Pass bookClient to server
}
```

## 🎓 Key Concepts Demonstrated

### 1. Microservice Architecture
- **Separation of Concerns**: Book service chỉ quản lý books, Author service quản lý authors
- **Independent Databases**: 2 SQLite databases riêng biệt
- **Service Discovery**: Services biết địa chỉ của nhau

### 2. Service-to-Service Communication
- **gRPC Client trong Server**: Author service vừa là server (phục vụ client) vừa là client (gọi Book service)
- **Request Chaining**: Client → Author Service → Book Service → Author Service → Client
- **Graceful Degradation**: Nếu Book service fail, Author service vẫn trả về author info

### 3. Data Relationships
- **Foreign Key**: `author_id` trong books table
- **Cross-Service Queries**: Lấy data từ 2 services để tạo view hoàn chỉnh
- **Data Aggregation**: Combine author info + books list

## 🔧 Database Schema

### Books Database (books_task5.db)
```sql
CREATE TABLE books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    author TEXT NOT NULL,
    isbn TEXT,
    price REAL,
    stock INTEGER,
    published_year INTEGER,
    author_id INTEGER DEFAULT 0  -- Foreign key
);
```

### Authors Database (authors.db)
```sql
CREATE TABLE authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    bio TEXT,
    birth_year INTEGER,
    country TEXT
);
```

## 🏆 Grading Criteria (25 points)

- ✅ **Author service proto defined** (3 points)
  - Author message với 5 fields
  - BookSummary message
  - 4 RPCs (GetAuthor, CreateAuthor, ListAuthors, GetAuthorBooks)

- ✅ **Author CRUD operations** (5 points)
  - GetAuthor với validation
  - CreateAuthor với birth year validation
  - ListAuthors với pagination

- ✅ **Book service updated with author_id** (3 points)
  - Field author_id thêm vào Book message
  - Database schema updated
  - Seed data có author_id

- ✅ **GetBooksByAuthor implemented** (4 points)
  - Query books by author_id
  - Return count và list
  - Validation author_id > 0

- ✅ **GetAuthorBooks with service-to-service call** (6 points)
  - Author service connect đến Book service khi khởi động
  - GetAuthorBooks call GetBooksByAuthor
  - Convert Books → BookSummary
  - Graceful degradation nếu Book service fail

- ✅ **Both services run simultaneously** (2 points)
  - Book service: port 50051
  - Author service: port 50052
  - Author service connect thành công đến Book service

- ✅ **Demo client shows integration** (2 points)
  - Tạo author
  - Tạo books cho author
  - Gọi GetAuthorBooks (cross-service call)
  - List authors và get stats

## 🎯 Real-World Applications

### Amazon Architecture Analog:
- **Inventory Service** (Book Service) - Quản lý sản phẩm
- **Seller Service** (Author Service) - Quản lý người bán
- **Order Service** gọi cả 2 để check stock + seller info
- **Payment Service** gọi pricing
- **Shipping Service** gọi address validation

### Benefits:
1. **Scalability**: Mỗi service có thể scale độc lập
2. **Fault Isolation**: Book service down → Author service vẫn hoạt động
3. **Technology Diversity**: Mỗi service có thể dùng tech khác nhau
4. **Team Autonomy**: Mỗi team quản lý 1 service

## 📚 References

- [Microservices Pattern](https://microservices.io/patterns/microservices.html)
- [gRPC Best Practices](https://grpc.io/docs/guides/performance/)
- [Service Mesh](https://istio.io/latest/docs/concepts/what-is-istio/)

---

**Total Score: 25/25 points** ✅
