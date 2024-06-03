package persistence

import (
	"errors"
	"fmt"
	"time"

	"notably/internal/model"
	ourutils "notably/internal/utils"
)

// TODO: Other user ops for production: modify, delete, admin user operations...
// NOTE: When replacing go-memdb with a real DB, the method receivers will all
// need to be *sql.DB (assuming the use of the database/sql package)
// NOTE: IMPORTANT: in go-memdb, txn.Insert() is actually an upsert.

func (db *NotablyDB) AddUser(userID, passwordHash string) (*model.User, error) {
	// TODO: When implementing user update/modify properly, refactor this method to be like AddOrUpdateNoteForUser()
	// Sanity checks
	userID, ok := ourutils.ValidateStringNotempty(userID)
	if !ok {
		return nil, errors.New("cannot add user because userID is empty/blank")
	}

	passwordHash, ok = ourutils.ValidateStringNotempty(passwordHash)
	if !ok {
		return nil, errors.New("cannot add user because password hash is empty/blank")
	}

	// Check whether user already exists
	_, err := db.GetUserByID(userID)
	if err == nil {
		// Uh oh...
		return nil, fmt.Errorf("user '%s' already exists", userID)
	}

	// Set the creation timestamp to  the current time, as seconds after Unix epoch
	creationTimestamp := time.Now().Unix()

	user := model.User{UserID: userID, PasswordHash: passwordHash, CreationTimestamp: creationTimestamp}
	txn := db.Txn(true) // Create a write transaction
	err = txn.Insert(usersTableName, user)
	if err != nil {
		txn.Abort() // go-memdb should have called this method Rollback() to be in line with database/sql. Oh well.
		return nil, fmt.Errorf("failed adding user '%s': %s", userID, err.Error())
	}

	txn.Commit()
	return &user, nil
}

func (db *NotablyDB) GetUserByID(userID string) (*model.User, error) {
	// Sanity checks
	userID, ok := ourutils.ValidateStringNotempty(userID)
	if !ok {
		return nil, errors.New("cannot search for user because userID is empty")
	}

	// Create read-only transaction.
	// For go-memdb, RO txn abort is basically a no-op, and one does not commit a RO txn.
	txn := db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First(usersTableName, "id", userID)
	if err != nil {
		return nil, fmt.Errorf("error getting user with ID '%s': %s", userID, err.Error())
	}

	if raw == nil {
		return nil, fmt.Errorf("user not found: Nil result from DB for user '%s'", userID)
	}

	user := raw.(model.User) // The go-memdb example is wrong here.
	return &user, nil
}

func (db *NotablyDB) GetAllUsers() ([]*model.User, error) {
	// See note above on read-only txns
	txn := db.Txn(false)
	defer txn.Abort()

	iter, err := txn.Get(usersTableName, "id")
	if err != nil {
		return nil, fmt.Errorf("failed getting all users: %s", err.Error())
	}

	var userList []*model.User
	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		// The go-memdb example is wrong here - the iterator object is NOT a pointer.
		// So we need to perform a runtime type assertion. See https://go.dev/ref/spec#Type_assertions
		user := obj.(model.User)
		userList = append(userList, &user)
	}

	return userList, nil
}
