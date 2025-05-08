module github.com/jaxxiy/newforum/auth_service

go 1.23.0

toolchain go1.23.6

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gorilla/mux v1.8.1
	github.com/jaxxiy/newforum/core v0.0.0
	github.com/lib/pq v1.10.9
	golang.org/x/crypto v0.38.0
	google.golang.org/grpc v1.64.1
)

require (
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240513163218-0867130af1f8 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
)

replace github.com/jaxxiy/newforum/core => ../core
