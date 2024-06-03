package persistence

import (
	"errors"
	"fmt"
	"time"

	"notably/internal/model"
	ourutils "notably/internal/utils"
)

// All note functions return the Note DTO to allow the GUI layer to display the note in question.

// TODO: method to get all notes for all users. This would be an admin user functionality

// Private helper function.
func (db *NotablyDB) addOrUpdateNoteForUser(userID, noteID, noteText string, update bool) (*model.Note, error) {
	var creationTimestamp, updateTimestamp int64
	var err error
	var ok bool

	if noteText == "" {
		return nil, errors.New("cannot create/update a note when the note text is empty")
	}

	// Calisthenics necessitated by txn.Insert() actually being an upsert. Oh, go-memdb...
	if update {
		// Sanity checks for update
		userID, noteID, err := ourutils.ValidateUserIDAndNoteID(userID, noteID)
		if err != nil {
			return nil, fmt.Errorf("cannot add note: %s", err.Error())
		}

		// Ensure that the noteID exists, since this is an update to an ostensibly existing note.
		tempNote, err := db.GetNoteForUser(userID, noteID)
		if err != nil {
			return nil, fmt.Errorf("note not found: Error finding note to update for userID='%s', noteID='%s': %s",
				userID, noteID, err.Error())
		}

		// If we got here, the update can proceed.
		// Set the timestamps accordingly.
		creationTimestamp = tempNote.CreationTimestamp
		updateTimestamp = time.Now().Unix() // seconds since Unix epoch
	} else {
		// This is an add.
		// Sanity check the userID.
		// If we were called in "add" mode, we would not (should not) have been passed a noteID.
		userID, ok = ourutils.ValidateStringNotempty(userID)
		if !ok {
			return nil, fmt.Errorf("need a user ID to add note")
		}

		// Generate a note ID using our inbuilt utility function.
		noteID, err = ourutils.GenerateKsuidAsString()
		if err != nil {
			return nil, fmt.Errorf("failed generating noteID: %v", err)
		}

		creationTimestamp = time.Now().Unix() // seconds since Unix epoch
	}

	// Ensure that the given userID exists in the system.
	// We need this because I haven't found a way to do table joins with go-memdb,
	// or even know if that's possible in go-memdb.
	_, err = db.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("cannot add note with ID '%s' for user '%s', user was not found",
			noteID, userID)
	}

	// If we got here, we can proceed with the create or update.
	// The note text will be added as-is.
	theNote := model.Note{
		NoteID:            noteID,
		NoteUserID:        userID,
		CreationTimestamp: creationTimestamp,
		UpdateTimestamp:   updateTimestamp,
		Note:              noteText,
	}

	txn := db.Txn(true) // Create a write transaction
	err = txn.Insert(notesTableName, theNote)
	if err != nil {
		txn.Abort() // go-memdb should have called this method Rollback() to be in line with database/sql. Oh well.
		return nil, fmt.Errorf("failed adding note with ID '%s' for user '%s': %s",
			noteID, userID, err.Error())
	}

	txn.Commit()
	return &theNote, nil
}

func (db *NotablyDB) AddNoteForUser(userID, noteText string) (*model.Note, error) {
	return db.addOrUpdateNoteForUser(userID, "", noteText, false)
}

func (db *NotablyDB) UpdateNoteForUser(userID, noteID, noteText string) (*model.Note, error) {
	return db.addOrUpdateNoteForUser(userID, noteID, noteText, true)
}

func (db *NotablyDB) GetNoteForUser(userID, noteID string) (*model.Note, error) {
	// Sanity checks
	userID, noteID, err := ourutils.ValidateUserIDAndNoteID(userID, noteID)
	if err != nil {
		return nil, fmt.Errorf("cannot get note: %s", err.Error())
	}

	// Ensure that the given userID exists in the system.
	// We need this because I haven't found a way to do table joins with go-memdb,
	// or even know if that's possible in go-memdb.
	_, err = db.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("cannot add note with ID '%s' for user '%s', user does not exist",
			noteID, userID)
	}

	// Create read-only transaction.
	// For go-memdb, RO txn abort is basically a no-op, and one does not commit a RO txn.
	txn := db.Txn(false)
	defer txn.Abort()

	// the "id" index is a compound index comprising the noteID string index and the
	// userID string index
	raw, err := txn.First(notesTableName, "id", noteID, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting note with ID '%s' for user '%s': %s", noteID, userID, err.Error())
	}

	if raw == nil {
		return nil, fmt.Errorf("note not found: Nil result from DB for note with id '%s' for user '%s'", noteID, userID)
	}

	theNote := raw.(model.Note) // The go-memdb example is wrong here.

	// Now validate that the NoteUserID is the same as userID
	if theNote.NoteUserID != userID {
		return nil, fmt.Errorf("note user ID mismatch for note ID '%s', expected user '%s' but got '%s'",
			noteID, userID, theNote.NoteUserID)
	}

	// If we got here, we have the note.
	return &theNote, nil
}

func (db *NotablyDB) GetAllNotesForUser(userID string) ([]*model.Note, error) {
	// Sanity
	userID, ok := ourutils.ValidateStringNotempty(userID)
	if !ok {
		// This is not OK (heh heh)
		return nil, errors.New("cannot get all notes for blank/empty user")
	}

	// Ensure that the given userID exists in the system.
	_, err := db.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("cannot get all notes for user '%s', error getting user: %s", userID, err.Error())
	}

	var noteList []*model.Note

	txn := db.Txn(false) // RO txn
	defer txn.Abort()

	iter, err := txn.Get(notesTableName, "noteUserID", userID)
	if err != nil {
		return nil, fmt.Errorf("cannot get all notes for user '%s', error in DB txn: %s", userID, err.Error())
	}
	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		note := obj.(model.Note) // Runtime type assertion. See https://go.dev/ref/spec#Type_assertions
		fmt.Println("DEBUG: GET ALL NOTES:", userID, ":", note)
		if note.NoteUserID != userID {
			continue
		}

		// If we got here, the note's user matches what we wanted. Add it to the slice
		noteList = append(noteList, &note)
	}

	return noteList, nil
}

// Returns the number of notes deleted. This is usually 1 or 0 (negative when there re errors).
func (db *NotablyDB) DeleteNoteForUser(userID, noteID string) (int, error) {
	// Sanity checks
	userID, noteID, err := ourutils.ValidateUserIDAndNoteID(userID, noteID)
	if err != nil {
		return -1, fmt.Errorf("cannot delete note due to userID/noteID validation failure: %s", err.Error())
	}

	txn := db.Txn(true) // Write txn

	// Ensure that the given userID exists in the system.
	_, err = db.GetUserByID(userID)
	if err != nil {
		return -1, fmt.Errorf("cannot get all notes for user '%s', error getting user: %s", userID, err.Error())
	}

	// NOTE: txn.DeleteAll(notesTableName, "id", noteID, userID)
	// does NOT error out when we try to delete a deleted note.
	// This makes sense, because from the DB's perspective, deleting nothing
	// still leaves the DB consistent :-)
	numDel, err := txn.DeleteAll(notesTableName, "id", noteID, userID)
	if err != nil {
		txn.Abort()
		return -1, fmt.Errorf("error deleting note for user '%s' noteID '%s': %s", userID, noteID, err.Error())
	}

	txn.Commit()
	return numDel, nil
}

// Returns the number of notes deleted, which will be negative on errors.
func (db *NotablyDB) DeleteAllNotesForUser(userID string) (int, error) {
	// Sanity
	userID, ok := ourutils.ValidateStringNotempty(userID)
	if !ok {
		// This is not OK (heh heh)
		return -1, errors.New("cannot delete all notes for blank/empty user")
	}

	// Ensure that the given userID exists in the system.
	_, err := db.GetUserByID(userID)
	if err != nil {
		return -1, fmt.Errorf("cannot get all notes for user '%s', error getting user: %s", userID, err.Error())
	}

	txn := db.Txn(true) // Write txn
	numDeleted, err := txn.DeleteAll(notesTableName, "noteUserID", userID)
	if err != nil {
		txn.Abort()
		return -1, fmt.Errorf("error deleting all notes for user '%s': %s", userID, err.Error())
	}

	txn.Commit()
	return numDeleted, nil
}
