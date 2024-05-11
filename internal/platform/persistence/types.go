package persistence

import (
	"github.com/hashicorp/go-memdb"
)

// This will be used as a method receiver for persistence operations.
// It wraps the valid opened database object as an anonymous member to allow
// us to pretend as if we added new methods directly to the DB object, which
// we can't otherwise do because it comes from a module external to this project.
// And it can't be put in the model because it will then become non-local to
// this package, which means we can't use it as a method receiver to the
// persistence methods.
type NotablyDB struct {
	*memdb.MemDB
}
