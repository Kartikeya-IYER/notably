package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"

	"github.com/gin-gonic/gin"

	"notably/internal/model"
	"notably/internal/platform/persistence"
	ourutils "notably/internal/utils"
)

// For the code that magically gets the DB connection from the current Gin context,
// see: https://github.com/gin-gonic/gin/issues/420#issuecomment-233893183

// There's going to be a lot of duplicated logic here, and I can't think of a good
// way to refactor it to remove the copypastas, because these are request handlers
// and hence are independent.

// Registers a new user.
// Uses POST with the following items in the POST body:
//   - The user ID as an email address.
//   - The password (plainText) to be used for logging in this user.
//
// The client must set the "Content-Type: application/json" header and pass a valid
// JSON body in the request.
func AddUser(c *gin.Context) {
	var reqUser model.RequestUser
	var message string

	// Get the body into the REQUEST DTO
	if err := c.BindJSON(&reqUser); err != nil {
		message = "Potentially malformed POST body."
		message += " Please ensure that the body is valid JSON and contains all relevant fields ('id', 'password')."
		message += fmt.Sprintf(" Error: %s", err.Error())

		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	// Sanity checks on the request DTO
	userID := reqUser.ID
	plaintextPassword := reqUser.Password

	userID, ok := ourutils.ValidateStringNotempty(userID)
	if !ok {
		message = "Request 'id' field is empty or blank"
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	// Now validate that the userID is a proper email address
	_, err := mail.ParseAddress(userID)
	if err != nil {
		message = "Request 'id' field does not appear to be a valid email address"
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	plaintextPassword, ok = ourutils.ValidateStringNotempty(plaintextPassword)
	if !ok {
		message = "Request 'password' field is empty or blank"
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	hashedPassword := ourutils.SHA256Hash(plaintextPassword)
	hashedPassword, ok = ourutils.ValidateStringNotempty(hashedPassword)
	if !ok {
		message = "Failed to hash Request 'password' field"
		c.IndentedJSON(http.StatusUnprocessableEntity, gin.H{"error": message})
		return
	}

	// Get the DB connection from our context.
	// See:
	db := c.MustGet("DB").(*persistence.NotablyDB)

	// Now we can finally call our persistence function.
	aUser, err := db.AddUser(userID, hashedPassword)
	if err != nil {
		respErr := http.StatusInternalServerError
		// Check whether the error has "not found" in it
		if ourutils.StrContainsInsensitive(err.Error(), "already exists") {
			respErr = http.StatusConflict
		}

		c.IndentedJSON(respErr, gin.H{"error": err.Error()})
		return
	}

	// Phew! Looks like we added the user.
	// For some weird reason I can't fathom, making this bit a utils function
	// causes weird things to happen. TODO: Figure out why.
	// Redact password hash
	aUser.PasswordHash = "REDACTED"
	respData, err := json.Marshal(aUser)
	if err != nil {
		message := fmt.Sprintf("Error marshalling model user to JSON: %s", err.Error())
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": message})
		return
	}

	r := json.RawMessage(string(respData))

	c.IndentedJSON(http.StatusCreated, gin.H{"message": r})
}

// "Logs in" an already registered user.
// If the login is successful, it sets a cookie (with expiry) to simulate session security.
// If the cookie expires, the user needs to log in again before they are able to use the API.
//
// Uses POST with the following items in the POST body:
//   - The user ID as an email address.
//   - The password (plainText) to be used for logging in this user.
//
// The client must set the "Content-Type: application/json" header and pass a valid
// JSON body in the request.
func LoginUser(c *gin.Context) {
	var reqUser model.RequestUser
	var message string

	// Get the body into the REQUEST DTO
	if err := c.BindJSON(&reqUser); err != nil {
		message = "Potentially malformed POST body."
		message += " Please ensure that the body is valid JSON and contains all relevant fields ('id', 'password')."
		message += fmt.Sprintf(" Error: %s", err.Error())
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	// Sanity checks on the request DTO
	userID := reqUser.ID
	plaintextPassword := reqUser.Password

	userID, ok := ourutils.ValidateStringNotempty(userID)
	if !ok {
		message = "Request 'id' field is empty or blank"
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	// Now validate that the userID is a proper email address
	_, err := mail.ParseAddress(userID)
	if err != nil {
		message = "Request 'id' field does not appear to be a valid email address"
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	plaintextPassword, ok = ourutils.ValidateStringNotempty(plaintextPassword)
	if !ok {
		message = "Request 'password' field is empty or blank"
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	hashedPassword := ourutils.SHA256Hash(plaintextPassword)
	// Trust, but verify ;-)
	hashedPassword, ok = ourutils.ValidateStringNotempty(hashedPassword)
	if !ok {
		message = "Failed to hash Request 'password' field"
		c.IndentedJSON(http.StatusUnprocessableEntity, gin.H{"error": message})
		return
	}

	// Get the DB connection from our context.
	// See: https://github.com/gin-gonic/gin/issues/932
	db := c.MustGet("DB").(*persistence.NotablyDB)

	// Now we can finally call our persistence function.
	aUser, err := db.GetUserByID(userID)
	if err != nil {
		respErr := http.StatusInternalServerError
		// Check whether the error has "not found" in it
		if ourutils.StrContainsInsensitive(err.Error(), "not found") {
			respErr = http.StatusNotFound
		}

		c.IndentedJSON(respErr, gin.H{"error": err.Error()})
		return
	}

	// If we got here, we have a user. Check whether it's the same one who made the request.
	aUserID := aUser.UserID
	aUserHashedPassword := aUser.PasswordHash
	if aUserID != userID || aUserHashedPassword != hashedPassword {
		message = "Forbidden. Terminating login due to user verification failure"
		c.IndentedJSON(http.StatusForbidden, gin.H{"error": message})
		return
	}

	// If we got here, we have a validated user. Let's set a cookie in the context.
	// We will use this cookie in subsequent calls to backend functions
	// which need a "logged in" user.
	// The signature of gin.Context.SetCookie() is:
	//    SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool)
	// where "maxAge" is in seconds (the docs don't mention this, but the source does)
	loginCookieMaxAgeSecs := c.MustGet(LoginCookieMaxAgeKey).(int)

	// Paranoia? Perhaps...
	if loginCookieMaxAgeSecs <= 0 {
		loginCookieMaxAgeSecs = DefaultLoginCookieMaxAgeSecs
	}

	c.SetCookie(LoginCookieName, aUserID, loginCookieMaxAgeSecs, "/", "localhost", false, true)

	message = fmt.Sprintf("OK, user %s logged in", userID)
	c.IndentedJSON(http.StatusOK, gin.H{"message": message})
}

// For a logged-in user, shows that user their own details (the password hash is redacted).
// This is a GET request handler
// The user ID will be a URL-encoded query parameter having the key as specified by the
// UserIDQueryParamKey defined in the handler infrastructure source file.
// Attempts by a user to view the info of any user other than themself will result in disappointment.
func GetUserById(c *gin.Context) {
	if !c.Request.URL.Query().Has(UserIDQueryParamKey) {
		// Any route where this is set as middleware MUST have the key.
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "Bad Request. Did not find the user ID key in the query params",
		})
		return
	}

	// Query().Get() is nice enough to URL-decode the encoded things for us.
	userID := c.Request.URL.Query().Get(UserIDQueryParamKey)
	userID, ok := ourutils.ValidateStringNotempty(userID)
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "Bad Request. User ID value in the query params was empty",
		})
		return
	}

	// Now that we have a valid user ID, let's call the backend DB method.
	db := c.MustGet("DB").(*persistence.NotablyDB)
	aUser, err := db.GetUserByID(userID)
	if err != nil {
		respErr := http.StatusInternalServerError
		// Check whether the error has "not found" in it
		if ourutils.StrContainsInsensitive(err.Error(), "not found") {
			respErr = http.StatusNotFound
		}

		c.IndentedJSON(respErr, gin.H{"error": err.Error()})
		return
	}
	if aUser.UserID != userID {
		// Hmmm...how did this happen? At any rate, we need to set a 404
		message := fmt.Sprintf("User '%s' not found", userID)
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": message})
		return
	}

	// For some weird reason I can't fathom, making this bit a utils function
	// causes weird things to happen. TODO: Figure out why.
	// Redact password hash
	aUser.PasswordHash = "REDACTED"
	respData, err := json.Marshal(aUser)
	if err != nil {
		message := fmt.Sprintf("Error marshalling model user to JSON: %s", err.Error())
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": message})
		return
	}

	r := json.RawMessage(string(respData))

	c.IndentedJSON(http.StatusOK, gin.H{"message": r})
}
