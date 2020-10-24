package config

// PermsCode - Minimal pems code for bot to work
const PermsCode int = 67628097

// VerTypes - Valid varification types
var VerTypes []string = []string{"auto", "manual"}

// RemoveReactions used to check if user reactions need to be removed on Reactive messaged
const RemoveReactions bool = true

// Emojis for verification

// ReactiveMsgHeader - Header for all reactive messages generated
const ReactiveMsgHeader string = "React for a role!"

// ApproveReaction - Approve verification request reaction
const ApproveReaction string = "approve:769004759259545610"

// DenyReaction - Deny verifiacation request reaction
const DenyReaction string = "deny:769004759482499124"

// MessageOnDeny - If the bot should message a user on role request denial
const MessageOnDeny bool = true

// MessageOnApprove - If the bot should message a user on role request approval
const MessageOnApprove bool = true
