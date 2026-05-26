package dbtype

import "strings"

const (
	MySQL    = "mysql"
	Postgres = "postgres"
	Mongo    = "mongo"
	Redis    = "redis"
)

var supported = []string{MySQL, Postgres, Mongo, Redis}

// Supported returns the database proxy types supported by capture, replay, and diff.
func Supported() []string {
	return append([]string(nil), supported...)
}

// IsSupported reports whether dbType is a supported database proxy type.
func IsSupported(dbType string) bool {
	for _, supportedType := range supported {
		if dbType == supportedType {
			return true
		}
	}
	return false
}

// Names returns a human-readable supported type list.
func Names() string {
	return strings.Join(supported, ", ")
}
