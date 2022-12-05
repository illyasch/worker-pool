module github.com/illyasch/worker-pool/examples/password-bcrypt-service

go 1.19

require (
	github.com/ardanlabs/conf/v3 v3.1.3
	github.com/illyasch/worker-pool/pool v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.8.1
	golang.org/x/crypto v0.3.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/illyasch/worker-pool/pool => ../../pool
