package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"notably/internal/model"
	"notably/internal/platform/persistence"
	ourutils "notably/internal/utils"
)

// For the code that magically gets the DB connection from the current Gin context,
// see: https://github.com/gin-gonic/gin/issues/420#issuecomment-233893183

// Duplicate code as copypastas will be refactored to be saner after I've figured
// out the best way to deal with doing this in a route handler.

// ALL ROUTE HANDLERS FOR NOTES functionality require that the user be "logged in"
// and have a valid authenticated "session". If the session times out, the API calls
// will error out until the user "logs in" again.

// Adds a new note for the given user.
// This is a POST handler, with the JSON POST body requiring the following fields:
//   - user_id : The email ID of the logged-in user.
//   - note : The note contents as a string.
//
// On success, will return the JSON object representing the note.
func AddNoteForUser(c *gin.Context) {
	var reqNote model.RequestNote
	var message string

	// Get the body into the REQUEST DTO
	if err := c.BindJSON(&reqNote); err != nil {
		message = "Potentially malformed POST body."
		message += " Please ensure that the body is valid JSON and"
		message += " contains all relevant fields ('user_id', 'note')."
		message += fmt.Sprintf(" Error: %s", err.Error())
		log.Printf("ERROR: ADD NOTE: %s\n", message)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	// Sanity check. The only thing we check is the user ID, since this is a new note.
	// Empty notes are allowed.
	userID := reqNote.UserID
	userID, ok := ourutils.ValidateStringNotempty(userID)
	if !ok {
		message = "Request 'user_id' field is empty or blank"
		log.Printf("ERROR: ADD NOTE: %s\n", message)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	noteText := reqNote.Note
	// We don't space-trim notes; we want them as the user entered them.
	// So we just check if it is altogether empty.
	if noteText == "" {
		message = fmt.Sprintf("Request 'note' field is empty for user '%s'", userID)
		log.Printf("ERROR: ADD NOTE: %s\n", message)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	// middlewareCookieMonster() should have already taken care of ensuring that the
	// user in the request body and the logged in user are the same.

	// Get the DB connection from our context.
	// See: https://github.com/gin-gonic/gin/issues/932
	db := c.MustGet("DB").(*persistence.NotablyDB)

	// Now we call our persistence function to create the note.
	aNote, err := db.AddNoteForUser(userID, noteText)
	if err != nil {
		respErr := http.StatusInternalServerError
		// Check whether the error has "not found" in it
		if ourutils.StrContainsInsensitive(err.Error(), "already exists") {
			respErr = http.StatusConflict
		}

		message := fmt.Sprintf("Error adding note for user '%s': %s", userID, err.Error())
		log.Printf("ERROR: ADD NOTE: %s\n", message)
		c.IndentedJSON(respErr, gin.H{"error": message})
		return
	}

	respData, err := json.Marshal(aNote)
	if err != nil {
		message := fmt.Sprintf("Error adding note for user '%s': %s", userID, err.Error())
		log.Printf("ERROR: ADD NOTE: %s\n", message)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{
			"error": message,
		})
		return
	}

	n := json.RawMessage(string(respData))
	c.IndentedJSON(http.StatusCreated, gin.H{"message": n})
}

// For a logged-in user, get a note by its note ID.
// This is a GET handler as well as a DELETE handler, with the note ID being a path param
// and the user ID // being a URL-encoded query param having the key specified by the
// UserIDQueryParamKey defined in the handler infrastructure source file.
// Attempts by a user to view or delete the notes of any user other than themself
// will result in disappointment.
func GetOrDeleteNoteByNoteIDForUser(c *gin.Context) {
	reqMethod := c.Request.Method
	if !c.Request.URL.Query().Has(UserIDQueryParamKey) {
		// Any route where this is set as middleware MUST have the key.
		message := fmt.Sprintf("Bad Request for %s Note. Did not find the user ID key in the query params",
			reqMethod)
		log.Printf("ERROR: %s NOTE: %s\n", reqMethod, message)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": message,
		})
		return
	}

	// Query().Get() is nice enough to URL-decode the encoded things for us.
	userID := c.Request.URL.Query().Get(UserIDQueryParamKey)
	noteID := c.Param("id")
	userID, noteID, err := ourutils.ValidateUserIDAndNoteID(userID, noteID)
	if err != nil {
		message := fmt.Sprintf("Bad Request for %s Note. Request %s param error: %s",
			reqMethod, UserIDQueryParamKey, err.Error())
		log.Printf("ERROR: %s NOTE: %s\n", reqMethod, message)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	// middlewareCookieMonster() should have done its job
	db := c.MustGet("DB").(*persistence.NotablyDB)

	var aNote *model.Note
	var numDeleted int
	var isGET bool

	// Now call the appropriate DB method depending on the request method.
	if reqMethod == "" || reqMethod == http.MethodGet {
		isGET = true
		aNote, err = db.GetNoteForUser(userID, noteID)
	} else {
		numDeleted, err = db.DeleteNoteForUser(userID, noteID)
	}
	if err != nil {
		message := fmt.Sprintf("Error %s note for user '%s': %s", reqMethod, userID, err.Error())
		log.Printf("ERROR: %s NOTE: %s\n", reqMethod, message)
		if isGET {
			respErr := http.StatusInternalServerError
			// Check whether the error has "not found" in it
			if ourutils.StrContainsInsensitive(err.Error(), "not found") {
				respErr = http.StatusNotFound
			}

			c.IndentedJSON(respErr, gin.H{"error": message})
		} else {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": message})
		}
		return
	}

	if isGET {
		// Return the Note DTO in the response.
		// Really got to figure out why the weirdness is happening so I can
		// get rid of this copypasta. It offends my sensibilities.
		respData, err := json.Marshal(aNote)
		if err != nil {
			message := fmt.Sprintf("Error getting note for user '%s': %s", userID, err.Error())
			log.Printf("ERROR: GET NOTE: %s", message)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{
				"error": message,
			})
			return
		}

		n := json.RawMessage(string(respData))
		c.IndentedJSON(http.StatusOK, gin.H{
			"message": n,
		})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"message": numDeleted})
}

// GET Handler as well as DELETE handler for all notes for a logged-in user.
func GetOrDeleteAllNotesForUser(c *gin.Context) {
	reqMethod := c.Request.Method
	if !c.Request.URL.Query().Has(UserIDQueryParamKey) {
		// Any route where this is set as middleware MUST have the key.
		message := fmt.Sprintf("Bad Request for %s All Notes. Did not find the user ID key in the query params",
			reqMethod)
		log.Printf("ERROR: %s ALL NOTES: %s\n", reqMethod, message)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": message,
		})
		return
	}

	// Query().Get() is nice enough to URL-decode the encoded things for us.
	userID := c.Request.URL.Query().Get(UserIDQueryParamKey)
	userID, ok := ourutils.ValidateStringNotempty(userID)
	if !ok {
		message := fmt.Sprintf("Bad Request for %s All Notes: User ID is empty or missing",
			reqMethod)
		log.Printf("ERROR: %s ALL NOTES: %s\n", reqMethod, message)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	// middlewareCookieMonster() should have done its job
	db := c.MustGet("DB").(*persistence.NotablyDB)

	var manyNotes []*model.Note
	var err error
	var numDeleted int
	var isGET bool

	// Now call the appropriate DB method depending on the request method.
	if reqMethod == "" || reqMethod == http.MethodGet {
		isGET = true
		manyNotes, err = db.GetAllNotesForUser(userID)
	} else {
		numDeleted, err = db.DeleteAllNotesForUser(userID)
	}
	if err != nil {
		message := fmt.Sprintf("Error %s all notes for user '%s': %s", reqMethod, userID, err.Error())
		log.Printf("ERROR: %s ALL NOTES: %s\n", reqMethod, message)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": message,
		})
		return
	}

	if isGET {
		// Now we convert the slice of our DTOs to raw JSON
		respData, err := json.Marshal(manyNotes)
		if err != nil {
			message := fmt.Sprintf("Error getting all notes for user '%s': %s", userID, err.Error())
			log.Printf("ERROR: GET ALL NOTES: %s\n", message)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{
				"error": message,
			})
			return
		}

		n := json.RawMessage(string(respData))
		c.IndentedJSON(http.StatusOK, gin.H{
			"message": n,
		})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"message": numDeleted})
}

// For a logged-in user, updates the given note ID with new data.
// This is a POST handler.
func UpdateNoteByNoteIDForUser(c *gin.Context) {
	var reqNote model.RequestNote
	var message string

	// Get the body into the REQUEST DTO
	if err := c.BindJSON(&reqNote); err != nil {
		message = "Potentially malformed POST body."
		message += " Please ensure that the body is valid JSON and contains"
		message += " all relevant fields ('id', 'user_id', 'note')."
		message += fmt.Sprintf(" Error: %s", err.Error())
		log.Printf("ERROR: UPDATE SINGLE NOTE: %s\n", err.Error())
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	// Sanity checks on the request DTO
	noteID := reqNote.ID
	userID := reqNote.UserID

	userID, noteID, err := ourutils.ValidateUserIDAndNoteID(userID, noteID)
	if err != nil {
		message = fmt.Sprintf("Bad Request. Missing required field(s): %s", err.Error())
		log.Printf("ERROR: UPDATE SINGLE NOTE: %s\n", message)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	// middlewareCookieMonster() should have done its job

	noteText := reqNote.Note
	// We don't space-trim notes; we want them as the user entered them.
	// So we just check if it is altogether empty.
	if noteText == "" {
		message = "Bad Request. Request 'note' field is empty"
		log.Printf("ERROR: UPDATE SINGLE NOTE for user '%s': %s\n", userID, message)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	db := c.MustGet("DB").(*persistence.NotablyDB)
	aNote, err := db.UpdateNoteForUser(userID, noteID, noteText)
	if err != nil {
		log.Printf("ERROR: UPDATE SINGLE NOTE for user '%s': %s\n", userID, err.Error())
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	respData, err := json.Marshal(aNote)
	if err != nil {
		log.Printf("ERROR: UPDATE SINGLE NOTE for user '%s': %s\n", userID, err.Error())
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	n := json.RawMessage(string(respData))
	c.IndentedJSON(http.StatusOK, gin.H{"message": n})
}

// Maybe allow users to update ALL their notes in one shot?
// If there are many notes, might need to page the notes...
