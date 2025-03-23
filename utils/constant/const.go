package constant

type DBMode string

const (
	SQLITE     = "sqlite"
	MYSQL      = "mysql"
	POSTGRESQL = "postgres"
)

const (
	READ  DBMode = "read"
	WRITE DBMode = "write"
)
