package persistence

import (
	"fmt"
	"testing"
)

// To see the info messages, run as:
//
//	go test -test.v

func TestPersistence(t *testing.T) {
	db, err := Open()
	if err != nil {
		t.Fatalf("Failed opening DB: %v", err)
	}

	userID := "testuser@testdomain.xyz"

	//////////////////////// User subtests ////////////////////////
	t.Run("User_Tests", func(t *testing.T) {
		// Get all users when we have none. Should have no errors, and 0 users.
		users, err := db.GetAllUsers()
		if err != nil {
			t.Fatalf("Failed getting all users: %v", err)
		} else {
			numUsers := len(users)
			if numUsers != 0 {
				t.Fatalf("Expected 0 users in empty DB but have %d", numUsers)
			}
		}

		// Let's add a valid user. Should succeed.
		user, err := db.AddUser(userID, "cafed00d")
		if err != nil {
			t.Fatalf("Failed adding a valid user: %v", err)
		} else {
			fmt.Println("TEST PERSISTENCE: USER: ADDED user:", user)
		}
		// Test getting the user we just added. Should succeed.
		user, err = db.GetUserByID(userID)
		if err != nil {
			t.Fatalf("Error getting user '%s' whom we just added: %v", userID, err)
		} else {
			fmt.Println("TEST PERSISTENCE: USER: Got existing user:", user)
		}

		// Add the user again with the same ID. Should error.
		user, err = db.AddUser(userID, "decafbad")
		if err == nil {
			t.Fatalf("Should have encountered an error adding the same user again, but didn't")
		}

		// Add a second temp user
		user, err = db.AddUser("tempuser", "abcedf")
		if err != nil {
			t.Fatalf("Failed adding a valid temp user: %v", err)
		} else {
			fmt.Println("TEST PERSISTENCE: USER: ADDED TEMP user:", user)
		}

		// Test getting a non-existent user. Should not succeed.
		_, err = db.GetUserByID("nonexistent")
		if err == nil {
			t.Fatal("Should have encountered an error getting a non-existent user, but didn't")
		}

		// Test getting empty/blank users. Should not succeed.
		_, err = db.GetUserByID("")
		if err == nil {
			t.Fatal("Should have encountered an error getting user with empty string as ID, but didn't")
		}
		_, err = db.GetUserByID("   ")
		if err == nil {
			t.Fatal("Should have encountered an error getting user with ID containing only spaces, but didn't")
		}

		// Now let's try to get the list of users.
		// Should have exactly two - the second one being the temp user
		users, err = db.GetAllUsers()
		if err != nil {
			t.Fatalf("Error getting all users: %v", err)
		} else {
			numUsers := len(users)
			if numUsers != 2 {
				t.Fatalf("Expected 2 users in DB but have %d", numUsers)
			}
			for _, user := range users {
				fmt.Println("TEST PERSISTENCE: USER: ALL USERS: got User:", user)
			}
		}
	})

	//////////////////////// Note subtests ////////////////////////
	t.Run("Note_Tests", func(t *testing.T) {
		// Test adding a note with empty userID. Should error out.
		_, err := db.AddNoteForUser("", "a note")
		if err == nil {
			t.Fatal("Should have encountered an error adding a note with empty userID and noteID, but didn't")
		}

		// Update a nonexisting note. Should fail.
		_, err = db.UpdateNoteForUser(userID, "bogusNoteID", "")
		if err == nil {
			t.Fatal("Should have encountered an error updating a nonexistent noteID, but didn't")
		}

		// Add a note with valid userID but empty note. Should fail.
		_, err = db.AddNoteForUser(userID, "")
		if err == nil {
			t.Fatalf("Should have encountered an error adding a note with empty text for user %s, but didn't", userID)
		}

		// Add a valid note. Should succeed
		theNote, err := db.AddNoteForUser(userID, "a note")
		if err != nil {
			t.Fatalf("Error adding valid note for user %s: %s", userID, err.Error())
		}
		noteID := theNote.NoteID

		// Get the note we just added. Should succeed
		theNote, err = db.GetNoteForUser(userID, noteID)
		if err != nil {
			t.Fatalf("%s", err.Error())
		} else {
			fmt.Println("TEST PERSISTENCE: NOTES: Got note:", theNote)
		}

		// Update the note we just added to have some text in the note. Should succeed.
		multilineNoteText := `Multiline note text line 1 of 3
Multiline note text line 2 of 3
Multiline note text line 3 of 3`
		theNote, err = db.UpdateNoteForUser(userID, noteID, multilineNoteText)
		if err != nil {
			t.Errorf("%s", err.Error())
		} else {
			fmt.Println("TEST PERSISTENCE: NOTES: Updated previous note:", theNote)
		}
		// Now get the note we just updated. Should succeed
		theNote, err = db.GetNoteForUser(userID, noteID)
		if err != nil {
			t.Errorf("%s", err.Error())
		} else {
			fmt.Println("TEST PERSISTENCE: NOTES: Got UPDATED note:", theNote)
		}

		// Add a note for a user who does not exist in the system. Should fail.
		_, err = db.AddNoteForUser("nonexistent_user", "a note")
		if err == nil {
			t.Error("Should have encountered an error adding a note with nonexistent userID, but didn't")
		}

		// Get a note for a different user but with noteID being the one for userID. Should fail.
		// Let's add another user first.
		userID2 := "user2@mmhmm.com"
		user2, err := db.AddUser(userID2, "c001d00d")
		if err != nil {
			t.Fatalf("Failed adding user '%s' for note test: %v", userID2, err)
		} else {
			fmt.Println("TEST PERSISTENCE: NOTES: ADDED user2:", user2)
		}
		_, err = db.GetNoteForUser(userID2, noteID)
		if err == nil {
			t.Fatalf("Should have encountered an error getting note having existing noteID '%s' for the WRONG user, but didn't",
				noteID)
		}

		// Get all notes for a user who has no notes. Should not error out, but the note list should have 0 length
		noteList, err := db.GetAllNotesForUser(userID2)
		if err != nil {
			t.Errorf("Error getting all notes for a user '%s' with no notes (should have not errored): %v",
				userID2, err)
		} else {
			nlen := len(noteList)
			if nlen != 0 {
				t.Fatalf("User '%s' should have had 0 notes, but we got %d", userID2, nlen)
			}
		}

		// Now add a note for our new user, with a new note ID
		theNote, err = db.AddNoteForUser(userID2, fmt.Sprintf("note for user '%s'", userID2))
		if err != nil {
			t.Fatalf("Error adding note for user '%s': %v", userID2, err)
		} else {
			fmt.Printf("TEST PERSISTENCE: NOTES: Added note for new user '%s':\n%v\n",
				userID2, theNote)
		}

		// Get all notes for a user who has one note
		noteList, err = db.GetAllNotesForUser(userID2)
		if err != nil {
			t.Fatalf("Error getting all notes for user '%s' with 1 note (should have not errored): %v",
				userID2, err)
		} else {
			nlen := len(noteList)
			// Expect exactly 1 note for this user
			if nlen != 1 {
				t.Fatalf("User '%s' should have had 1 note, but we got %d", userID2, nlen)
			}

			for _, note := range noteList {
				fmt.Printf("TEST PERSISTENCE: NOTES: ALL NOTES for user '%s': Got Note: %v\n",
					userID2, note)
			}
		}

		// Now let's add another note for userID2 and then get all notes again
		theNote, err = db.AddNoteForUser(userID2, fmt.Sprintf("NOTE TWO for user '%s'", userID2))
		if err != nil {
			t.Fatalf("Error adding second note for user '%s': %v", userID2, err)
		} else {
			fmt.Printf("TEST PERSISTENCE: NOTES: Added second note for user '%s':\n%v\n",
				userID2, theNote)
		}

		// Get all notes for a user who has > 1 note
		noteList, err = db.GetAllNotesForUser(userID2)
		if err != nil {
			t.Fatalf("Error getting all notes for user '%s' with > 1 note (should have not errored): %v",
				userID2, err)
		} else {
			nlen := len(noteList)
			// Expect exactly 2 notes for this user
			if nlen != 2 {
				t.Fatalf("User '%s' should have had 2 notes, but we got %d", userID2, nlen)
			}

			for _, note := range noteList {
				fmt.Printf("TEST PERSISTENCE: NOTES: ALL NOTES for user '%s': Got Note: %v\n",
					userID2, note)
			}
		}

		// Delete note for user. We'll delete a note for a user with exactly 1 note,
		// then ensure that GetAllNotesForUser() shows 0 notes
		numDel, err := db.DeleteNoteForUser(userID, noteID)
		if err != nil {
			t.Fatalf("Error deleting note ID '%s' for user '%s' (user has 1 note): %s",
				noteID, userID, err.Error())
		} else {
			fmt.Printf("TEST PERSISTENCE: NOTES: Deleted note ID '%s' for user '%s': %v\n",
				noteID, userID, numDel)
		}
		noteList, err = db.GetAllNotesForUser(userID)
		if err != nil {
			t.Fatalf("Error getting all notes for user '%s' with 0 notes (should have not errored): %v",
				userID, err)
		} else {
			nlen := len(noteList)
			// Expect exactly 0 notes for this user
			if nlen != 0 {
				t.Fatalf("User '%s' should have had 0 notes, but we got %d", userID, nlen)
			}
		}

		// Now try deleting a deleted note. This should NOT error out.
		// Reason being, txn.DeleteAll does not error out in this case
		// which makes sense because the DB stays consistent - nothing
		// has changed.
		numDel, err = db.DeleteNoteForUser(userID, noteID)
		if err != nil {
			t.Fatalf("Unexpected error deleting already-deleted note ID '%s' for user '%s': %v",
				noteID, userID, err)
		} else {
			fmt.Printf("TEST PERSISTENCE: NOTES: Deleted already deleted note ID '%s' for user '%s': %v\n",
				noteID, userID, numDel)
		}

		// Now delete all the notes for userID2.
		numDel, err = db.DeleteAllNotesForUser(userID2)
		if err != nil {
			t.Fatalf("Error deleting all notes for user %s: %v", userID2, err)
		} else {
			fmt.Printf("TEST PERSISTENCE: NOTES: Deleted ALL notes for user %s. Num deleted: %d\n", userID2, numDel)
		}
		// Now get all notes for userID2 and ensure that the count is 0
		noteList, err = db.GetAllNotesForUser(userID2)
		if err != nil {
			t.Fatalf("Error getting all notes for user '%s' with 0 notes (should have not errored): %v",
				userID2, err)
		} else {
			nlen := len(noteList)
			// Expect exactly 0 notes for this user
			if nlen != 0 {
				t.Fatalf("User '%s' should have had 0 notes, but we got %d", userID2, nlen)
			}
		}

		// Now delete all the notes for userID2 AGAIN
		numDel, err = db.DeleteAllNotesForUser(userID2)
		if err != nil {
			t.Fatalf("Error deleting all notes for user %s: %v", userID2, err)
		} else {
			fmt.Printf("TEST PERSISTENCE: NOTES: Deleted ALL notes for user %s. Num deleted: %d\n", userID2, numDel)
		}
		// Now get all notes for userID2 and ensure that the count is 0
		noteList, err = db.GetAllNotesForUser(userID2)
		if err != nil {
			t.Fatalf("Error getting all notes for user '%s' with 0 notes (should have not errored): %v",
				userID2, err)
		} else {
			nlen := len(noteList)
			// Expect exactly 0 notes for this user
			if nlen != 0 {
				t.Fatalf("User '%s' should have had 0 notes, but we got %d", userID2, nlen)
			}
		}
	})
}
