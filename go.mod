module github.com/rebelice/mysql-completer

go 1.21.1

require (
	github.com/antlr4-go/antlr/v4 v4.13.0
	github.com/bytebase/mysql-parser v0.0.0-20231013095254-61b0903123c6
)

require golang.org/x/exp v0.0.0-20230515195305-f3d0a9c9a5cc // indirect

replace github.com/antlr4-go/antlr/v4 => github.com/rebelice/antlr/v4 v4.0.0-20231025084258-3010199da4f1
