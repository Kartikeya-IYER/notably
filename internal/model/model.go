package model

// The data structures used by the persistence layer
type User struct {
	UserID            string `json:"user_id"`
	PasswordHash      string `json:"password_hash"`
	CreationTimestamp int64  `json:creation_timestamp"`
}

type Note struct {
	NoteID            string `json:"note_id"`
	NoteUserID        string `json:"note_user_id"`
	CreationTimestamp int64  `json:"creation_timestamp"`
	UpdateTimestamp   int64  `json:"update_timestamp"`
	Note              string `json:"note"`
}

// The RESPONSE Data Transfer Object (DTO) for operations on users.
type ResponseUser struct {
	*User
}

// The RESPONSE Data Transfer Object (DTO) for operations on notes.
type ResponseNote struct {
	*Note
}

// The REQUEST DTO used in the route handler for user ops.
type RequestUser struct {
	ID       string `json:"id"`
	Password string `json:"password"`
}

// The REQUEST DTO used in the route handler for note ops
type RequestNote struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	Note   string `json:"note"`
}
