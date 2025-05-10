module github.com/jaxxiy/newforum/auth_service

go 1.23.0

toolchain go1.23.6

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gorilla/mux v1.8.1
	github.com/jaxxiy/newforum/core v0.0.0
	github.com/stretchr/testify v1.9.0
	github.com/swaggo/swag v1.16.3
	golang.org/x/crypto v0.38.0
	google.golang.org/grpc v1.64.1
)

require (
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/spec v0.20.15 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/swaggo/files v0.0.0-20220610200504-28940afbdbfe // indirect
	github.com/swaggo/http-swagger v1.3.4 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240513163218-0867130af1f8 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/jaxxiy/newforum/core => ../core
