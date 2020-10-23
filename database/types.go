package database

import (
	"time"
)

// GuildSettings - DB record for guild settings
type GuildSettings struct {
	ID               string
	VerificationChan string
	EnabledChannels  map[string]ReactiveChannel
}

// ReactiveChannel - Channel record
type ReactiveChannel struct {
	ID        string
	MessageID string
	Roles     map[string]*Role
}

// Role - Role record
type Role struct {
	ID       string
	Reaction string
	Name     string
	// Verification type auto / manual
	Verification string
}

// User - Discord user db record
type User struct {
	ID string
	// map[cahnnelID][roleID]UerReaction
	RoleRequests map[string]map[string]Request
	IsSoftBanned bool
}

// Request - User reaction obj
type Request struct {
	Active    bool
	Timestamp time.Time
}

// VerificationMessage - message sent to verification channel struct
type VerificationMessage struct {
	ID            string
	GuildID       string
	UserID        string
	RequestChanID string
	RoleRequested *Role
	Timestamp     time.Time
}
