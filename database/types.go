package database

// GuildSettings - DB record for guild settings
type GuildSettings struct {
	ID              string
	EnabledChannels map[string]ReactiveChannel
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
