package persistence

import (
	"fmt"

	"github.com/hashicorp/go-memdb"
)

const (
	usersTableName = "users"
	notesTableName = "notes"
)

// In real life, this would be an sql.Open() call to an existing DB from an ACID-compliant database.
func Open() (*NotablyDB, error) {
	// Table "DDL" for go-memdb.
	// Although go-memdb is an in-memory database and doesn't support the "D" in "ACID",
	// it has the advantage of not needing a database client installed on the target machine.
	usersTable := &memdb.TableSchema{
		Name: usersTableName,
		Indexes: map[string]*memdb.IndexSchema{
			// id = model.User.UserID is an email address
			"id": &memdb.IndexSchema{
				Name:    "id",
				Unique:  true,
				Indexer: &memdb.StringFieldIndex{Field: "UserID"},
			},

			// The hash of the user's password, to avoid storing raw passwords.
			"passwordHash": &memdb.IndexSchema{
				Name:    "passwordHash",
				Unique:  false,
				Indexer: &memdb.StringFieldIndex{Field: "PasswordHash"},
			},

			// The timestamp (since Unix epoch) of when the user was initially created.
			"creationTimestamp": &memdb.IndexSchema{
				Name:    "creationTimestamp",
				Unique:  false,
				Indexer: &memdb.IntFieldIndex{Field: "CreationTimestamp"},
			},

			// TODO: Other fields: updateTimestamp, name, isAdmin, etc
		},
	}

	notesTable := &memdb.TableSchema{
		Name: notesTableName,
		Indexes: map[string]*memdb.IndexSchema{
			// "id" is akin to a composite primary key.
			"id": &memdb.IndexSchema{
				Name:         "id",
				Unique:       true,
				AllowMissing: false,
				Indexer: &memdb.CompoundIndex{
					Indexes: []memdb.Indexer{
						&memdb.StringFieldIndex{Field: "NoteID"},
						&memdb.StringFieldIndex{Field: "NoteUserID"},
					},
				},
			},

			"noteUserID": &memdb.IndexSchema{
				Name:         "noteUserID",
				Unique:       false,
				AllowMissing: false,
				Indexer:      &memdb.StringFieldIndex{Field: "NoteUserID"},
			},

			// The timestamp (since Unix epoch) of when the note was initially created.
			"creationTimestamp": &memdb.IndexSchema{
				Name:    "creationTimestamp",
				Unique:  false,
				Indexer: &memdb.IntFieldIndex{Field: "CreationTimestamp"},
			},

			// The timestamp (since Unix epoch) of when the note was most recently modified.
			"updateTimestamp": &memdb.IndexSchema{
				Name:    "updateTimestamp",
				Unique:  false,
				Indexer: &memdb.IntFieldIndex{Field: "UpdateTimestamp"},
			},

			// The contents of the note item
			// A user can save an empty note if they wish, although it would be pointless, eh...
			"note": &memdb.IndexSchema{
				Name:         "note",
				Unique:       false,
				Indexer:      &memdb.StringFieldIndex{Field: "Note"},
				AllowMissing: false,
			},
		},
	}

	// The main DB schema
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			usersTableName: usersTable,
			notesTableName: notesTable,
		},
	}

	theDB, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, fmt.Errorf("Failed to open DB: %s", err.Error())
	}

	ourDB := NotablyDB{theDB}
	return &ourDB, nil
}
