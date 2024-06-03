// Utilities which can be used across the entire project.
package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/segmentio/ksuid"
)

// ValidateStringNotempty checks whether a string is empty.
// If the given string is non-empty, returns the space-trimmed string and true.
// If the given string is empty, returns the empty string and false.
func ValidateStringNotempty(theString string) (string, bool) {
	if theString == "" {
		return theString, false
	}

	theString = strings.TrimSpace(theString)
	// Check again
	if theString == "" {
		return theString, false
	}

	// If we got here, the string is non-empty and space-trimmed.
	return theString, true
}

// ValidateUserIDAndNoteID validates a noteID and userID pair, since we use that quite a lot.
// If there were no errors, returns the space-stripped userID and noteID (in that order).
// On errors, an error object is returned, along with EMPTY STRINGS for userID and noteID.
func ValidateUserIDAndNoteID(userID, noteID string) (uid string, nid string, err error) {
	userID, ok := ValidateStringNotempty(userID)
	if !ok {
		return "", "", errors.New("userID is empty/blank")
	}
	noteID, ok = ValidateStringNotempty(noteID)
	if !ok {
		return "", "", errors.New("noteID is empty/blank")
	}

	// At this point, we have space-stripped IDs
	return userID, noteID, nil
}

// GenerateKsuid() generates a UUID-like entity using ksuid.
// https://github.com/segmentio/ksuid/blob/master/README.md
//
// We use ksuid instead of a canonical UUID because a ksuid offers certain
// benefits over using a UUID:
//   - It is lexically sortable because it has an inbuilt time component which
//     helps with the lexical sort.
//   - It is base62 so it is alphanumeric and thus lends itself well to being
//     used in JSON and RESTful APIs
//
// This function generates a raw ksuid in case we need it to use it in
// different forms. See https://github.com/segmentio/ksuid/blob/master/README.md#plays-well-with-others
func GenerateKsuid() (ksuid.KSUID, error) {
	// ksuid.New() may panic in rare cases. To handle errors gracefully without panic(),
	// we call ksuid.NewRandom(), which is what ksuid.New() calls internally.
	return ksuid.NewRandom()
}

// GenerateKsuidAsString is a convenience function to get a ksuid in string form.
func GenerateKsuidAsString() (string, error) {
	ks, err := GenerateKsuid()
	if err != nil {
		return "", fmt.Errorf("failed generating KSUID: %s", err.Error())
	}
	return ks.String(), nil
}

// SHA256Hash creates a one-way-hash of the given plainText. We do NOT store plaintext passwords.
// The return string is a hex representation of the hashed plaintext.
func SHA256Hash(plainText string) string {
	h := sha256.New()
	h.Write([]byte(plainText))
	hSum := h.Sum(nil)
	return hex.EncodeToString(hSum)
}

// StrContainsInsensitive is a case-insensitive Contains function for strings.
// Strangely, I ended up with the same implementation as https://stackoverflow.com/a/44601360
// when I went searching to see if there was a better way to do this :-)
func StrContainsInsensitive(a, b string) bool {
	return strings.Contains(
		strings.ToLower(a),
		strings.ToLower(b),
	)
}
