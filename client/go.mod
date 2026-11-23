module github.com/0x4D67/tenantzero/client

go 1.26.5

replace github.com/0x4D67/tenantzero/shared => ../shared

replace github.com/0x4D67/tenantzero/server => ../server

require (
	github.com/0x4D67/tenantzero/server v0.0.0-00010101000000-000000000000
	github.com/0x4D67/tenantzero/shared v0.0.0-00010101000000-000000000000
	github.com/gorilla/websocket v1.5.3
	golang.org/x/term v0.48.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/jung-kurt/gofpdf v1.16.2 // indirect
	golang.org/x/sys v0.50.0 // indirect
)
