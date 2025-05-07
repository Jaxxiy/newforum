module github.com/jaxxiy/newforum/forum_service

go 1.21

require (
	github.com/golang-migrate/migrate/v4 v4.18.3
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/jaxxiy/newforum/core v0.0.0
	github.com/lib/pq v1.10.9
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.72.0
)

require (
	github.com/golang-jwt/jwt/v5 v5.2.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/protobuf v1.36.5 // indirect
)

replace github.com/jaxxiy/newforum/core => ../core
