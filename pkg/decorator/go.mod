module github.com/architeacher/nomos-technical-challenge/pkg/decorator

go 1.26

replace github.com/architeacher/nomos-technical-challenge/pkg/logger => ../logger

require (
	github.com/architeacher/nomos-technical-challenge/pkg/logger v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/otel v1.40.0
	go.opentelemetry.io/otel/trace v1.40.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
