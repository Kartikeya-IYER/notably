// THIS IS NOT A ROUTE HANDLER. Hence the xxxx prefix.
// It is local infrastructure for the router and route handlers.

package handlers

const (
	DefaultLoginCookieMaxAgeSecs = 28800 // 28800 seconds = 8 hours

	// The name of the login cookie we set on successful user login.
	LoginCookieName = "lcmas"

	// The name of the router context variable used to get the max age setting.
	LoginCookieMaxAgeKey = "LoginCookieMaxAgeSecs"

	// When a user ID is passed as a (URL-encoded) query param, this is the key it will have.
	UserIDQueryParamKey = "userid"
)
