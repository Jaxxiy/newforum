module github.com/jaxxiy/newforum/core

go 1.23.0

toolchain go1.23.6

require (
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/golang-migrate/migrate/v4 v4.18.3
	github.com/lib/pq v1.10.9
)

require (
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	go.uber.org/atomic v1.7.0 // indirect
)

// Note: This module is designed to be used as a standalone package
// Import it in your projects using:
// go get github.com/jaxxiy/newforum/core
