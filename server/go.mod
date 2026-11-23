module github.com/0x4D67/tenantzero/server

go 1.26.5

replace github.com/0x4D67/tenantzero/shared => ../shared

require (
	github.com/0x4D67/tenantzero/shared v0.0.0-00010101000000-000000000000
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/jung-kurt/gofpdf v1.16.2
	github.com/lib/pq v1.12.3
	golang.org/x/crypto v0.59.0
	gopkg.in/yaml.v3 v3.0.1
)
