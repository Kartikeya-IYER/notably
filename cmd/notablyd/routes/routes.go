package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"

	"github.com/gin-gonic/gin"

	"notably/cmd/notablyd/routes/handlers"
	"notably/internal/platform/persistence"
	ourutils "notably/internal/utils"
)

// middlewareCookieMonster is router middleware which handles checking whether the
// request user has a valid login/session cookie.
func middlewareCookieMonster() gin.HandlerFunc {
	log.Println("Setting up the middleware cookie monster (om nom nom nom)...")
	return func(c *gin.Context) {
		// Get the login cookie
		message := ""
		if cookieValue, err := c.Cookie(handlers.LoginCookieName); err == nil {
			if cookieValue != "" {
				// Get the active user ID from the context
				// Since we use this for users as well as notes, and they each have
				// different DTOs, we need to use the trick of unmarshalling to a map[string]interface{}
				bodyMap := make(map[string]interface{})
				userID := ""

				if c.Request.Method == http.MethodPost {
					// POST methods have the user ID in the body.
					// If we access the body in the middleware, we will consume it.
					// So let's make a copy of it first, and then work with the copy.
					// This will still consume the request body, so we'll need to set it back.
					// NOTE: can't use ShouldBindBodyWith() to avoid this, because all
					// subsequent calls to bind to the body must also use the same function.
					// Fun, eh?
					// The code below has been adapted from: https://stackoverflow.com/a/72680426
					bodyCopy := new(bytes.Buffer)
					_, err := io.Copy(bodyCopy, c.Request.Body) // Read the whole body.
					if err != nil {
						message = fmt.Sprintf("Login Verification Error (POST request): Error copying request body: %s",
							err.Error())
						log.Printf("ERROR: LOGIN COOKIE ROUTER MIDDLEWARE: %s\n", message)
						c.IndentedJSON(http.StatusInternalServerError, gin.H{
							"error": message,
						})
						c.Abort()
						return
					}
					bodyData := bodyCopy.Bytes()

					err = json.Unmarshal(bodyData, &bodyMap)
					if err != nil {
						message = fmt.Sprintf("Login Verification Error (POST request): Error parsing copied request body: %s",
							err.Error())
						log.Printf("ERROR: LOGIN COOKIE ROUTER MIDDLEWARE: %s\n", message)
						c.IndentedJSON(http.StatusInternalServerError, gin.H{
							"error": message,
						})
						c.Abort()
						return
					}

					// Needle, meet haystack. We need to find the userID which could be named
					// either 'id' (RequestUser DTO) or 'user_id' (RequestNote DTO)
					if bodyMap["user_id"] != nil {
						userID = bodyMap["user_id"].(string) // type assertion needed because interface{}
					} else if bodyMap["id"] != nil {
						// we'll erroneously reach this case if a note-related request was
						// missing the user_id field. But we'll check to see if it is an
						// email ID, so no worries.
						userID = bodyMap["id"].(string) // type assertion needed because interface{}
					}

					userID, ok := ourutils.ValidateStringNotempty(userID)
					if !ok {
						message = "Login Verification Error (POST request): User ID is missing or empty"
						log.Printf("ERROR: LOGIN COOKIE ROUTER MIDDLEWARE: %s\n", message)
						c.IndentedJSON(http.StatusBadRequest, gin.H{
							"error": message,
						})
						c.Abort()
						return
					}
					// Now check if it is an email ID
					_, err = mail.ParseAddress(userID)
					if err != nil {
						message = fmt.Sprintf("Login Verification Error (POST request): Error parsing user ID: %s",
							err.Error())
						log.Printf("ERROR: LOGIN COOKIE ROUTER MIDDLEWARE: %s\n", message)
						c.IndentedJSON(http.StatusBadRequest, gin.H{
							"error": message,
						})
						c.Abort()
						return
					}

					// FINALLY, now we can check the userID.
					if userID != cookieValue {
						message = fmt.Sprintf("Login Verification Error (POST request): User '%s' is not logged in",
							userID)
						log.Printf("ERROR: LOGIN COOKIE ROUTER MIDDLEWARE: %s\n", message)
						c.IndentedJSON(http.StatusForbidden, gin.H{
							"error": message,
						})
						c.Abort()
						return
					}

					// Replace the body - without this, the request EOFs when parsing the body.
					c.Request.Body = io.NopCloser(bytes.NewReader(bodyData))
				} else {
					// Not POST, userID will be a URL-encoded query parameter having key handlers.UserIDQueryParamKey
					if !c.Request.URL.Query().Has(handlers.UserIDQueryParamKey) {
						// Any route where this is set as middleware MUST have the key.
						message = fmt.Sprintf("Login Verification Error: Request URL Query Params do not contain User ID field: %s",
							handlers.UserIDQueryParamKey)
						log.Printf("ERROR: LOGIN COOKIE ROUTER MIDDLEWARE: %s\n", message)
						c.IndentedJSON(http.StatusBadRequest, gin.H{
							"error": message,
						})
						c.Abort()
						return
					}

					// Query().Get() is nice enough to URL-decode the encoded things for us.
					userID := c.Request.URL.Query().Get(handlers.UserIDQueryParamKey)
					userID, ok := ourutils.ValidateStringNotempty(userID)
					if !ok {
						message = "Login Verification Error: User ID is missing or empty"
						log.Printf("ERROR: LOGIN COOKIE ROUTER MIDDLEWARE: %s\n", message)
						c.IndentedJSON(http.StatusBadRequest, gin.H{
							"error": message,
						})
						c.Abort()
						return
					}

					// FINALLY, now we can check the userID.
					if userID != cookieValue {
						message = fmt.Sprintf("Login Verification Error: User '%s' is not logged in",
							userID)
						log.Printf("ERROR: LOGIN COOKIE ROUTER MIDDLEWARE: %s\n", message)
						c.IndentedJSON(http.StatusForbidden, gin.H{
							"error": message,
						})
						c.Abort()
						return
					}
				}

				// If we made it here without returning, we're good to go
				c.Next()
				return
			}
		}

		// If we got here, Cookie verification failed, usually because there is no cookie.
		// TODO When proper login is implemented, send a WWW-Authenticate header in the response.
		message = "Login Verification Error: No user logged in, or failure verifying valid login session"
		log.Printf("ERROR: LOGIN COOKIE ROUTER MIDDLEWARE: %s\n", message)
		c.IndentedJSON(http.StatusUnauthorized, gin.H{
			"error": message,
		})
		c.Abort()
	}
}

// middlewareSetupRouter is middleware which sets up the DB connection to pass to route handlers.
// Also passes the max age (in seconds) of the login session cookie which gets
// set on a successful login.
func middlewareSetupRouter(rc RouterConfig) gin.HandlerFunc {
	// Get the max age of the login cookie.
	loginCookieMaxAgeSecs := rc.LoginCookieMaxAgeSecs
	if loginCookieMaxAgeSecs <= 0 {
		// Set it to the default
		loginCookieMaxAgeSecs = handlers.DefaultLoginCookieMaxAgeSecs
	}
	log.Printf("Router middleware setup: Login cookie max age (seconds): %d\n", loginCookieMaxAgeSecs)

	// Calisthenics to pass the DB connection to the route handlers.
	// Adapted from: https://github.com/gin-gonic/gin/issues/420
	// We pass the DB connection object to the handlers via the gin context.
	log.Println("Router middleware setup: Opening DB connection...")
	db, err := persistence.Open()
	if err != nil {
		// No option but to panic and die
		panic(err)
	}

	log.Println("Router middleware setup done, returning with context settings for required things.")
	// Now we set our router context with the things we want in it.
	return func(c *gin.Context) {
		c.Set("DB", db)
		c.Set(handlers.LoginCookieMaxAgeKey, loginCookieMaxAgeSecs)
		c.Next()
	}
}

// NewRouter creates a new Gin router.
func NewRouter(rc RouterConfig) *gin.Engine {
	log.Println("Creating router...")
	r := gin.Default()

	log.Println("Setting up router middleware...")
	r.Use(middlewareSetupRouter(rc))

	log.Println("Setting up routes and their associated handlers...")
	// Specify an API v1 group.
	// In case we ever make breaking changes in the future, those changes can
	// go into a v2 API group, and so on.
	// The handler functions are defined in the "handlers" subdirectory.
	v1 := r.Group("/api/v1")
	{
		// Are we alive? How are we doing?
		v1.GET("/health", handlers.GetHealth)

		// NOTE: For anything other than a POST, and which requires the user ID,
		// the userID will be a query parameter called "userid", with the value URL-encoded.

		// User APIs
		// TODOs:
		//  - Administrative routes for an admin user.
		//  - Allow users to modify themselves.
		//  - Allow users to delete themselves (GDPR!)
		v1.POST("/register", handlers.AddUser)
		v1.POST("/login", handlers.LoginUser)                            // Will set a cookie with the username.
		v1.PUT("/logout", handlers.LogoutUser)                           // Deletes an existing login cookie.
		v1.GET("/user", middlewareCookieMonster(), handlers.GetUserById) // Get our own info. Needs the cookie from login.

		// Note APIs.
		// Life would be MUCH simpler if GET and DELETE requests had been designed with bodies.
		// These all check/use the cookie created by the user login route.
		v1.POST("/note", middlewareCookieMonster(), handlers.AddNoteForUser)

		// Note ID needs to be in path param as well as body
		v1.POST("/note/:id", middlewareCookieMonster(), handlers.UpdateNoteByNoteIDForUser)

		v1.GET("/note/:id", middlewareCookieMonster(), handlers.GetOrDeleteNoteByNoteIDForUser)
		v1.GET("/note", middlewareCookieMonster(), handlers.GetOrDeleteAllNotesForUser)
		v1.DELETE("/note/:id", middlewareCookieMonster(), handlers.GetOrDeleteNoteByNoteIDForUser)
		v1.DELETE("/note", middlewareCookieMonster(), handlers.GetOrDeleteAllNotesForUser)
	}

	log.Println("Router creation completed successfully")
	return r
}
