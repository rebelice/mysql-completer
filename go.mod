module github.com/rebelice/mysql-completer

go 1.21.1

require (
	github.com/antlr4-go/antlr/v4 v4.13.0
	github.com/bytebase/mysql-parser v0.0.0-20231027071737-2b1ee7eca26c
	github.com/stretchr/testify v1.8.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/exp v0.0.0-20230515195305-f3d0a9c9a5cc // indirect
)

replace github.com/antlr4-go/antlr/v4 => github.com/rebelice/antlr/v4 v4.0.0-20231025084258-3010199da4f1
