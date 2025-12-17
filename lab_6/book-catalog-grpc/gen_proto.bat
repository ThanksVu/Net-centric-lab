@echo off
set PATH=%PATH%;C:\Users\hung1\go\bin
protoc --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative proto\book_service.proto
echo Proto files generated successfully!
