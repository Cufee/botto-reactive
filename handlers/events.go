package handlers

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/cufee/botto-reactive/config"
	db "github.com/cufee/botto-reactive/database"
)

// ReactionRoleCheck - Check if a reaction is on a RR message and give role / submit request
func ReactionRoleCheck(s *discordgo.Session, e *discordgo.MessageReactionAdd) {
	// Ignore self
	if e.UserID == s.State.User.ID {
		return
	}

	// Get request data
	reqMsg, err := s.ChannelMessage(e.ChannelID, e.MessageID)
	if err != nil {
		log.Print(err)
		return
	}

	if reqMsg.Author.ID != s.State.User.ID {
		return
	}

	// Get guild settings
	guildSettings, err := db.GetGuildSettings(e.GuildID)
	if err != nil {
		eventReplyDel(s, e.ChannelID, fmt.Sprintf("An error occured while fetching guild settings.\n```%v```", err), 5)
		return
	}

	// Check if this channel and message are enabled
	channelSettings, ok := guildSettings.EnabledChannels[e.ChannelID]
	if !ok || e.MessageID != channelSettings.MessageID {
		return
	}

	// Check if reaction matched a role
	validReaction := false
	var role *db.Role
	for _, r := range guildSettings.EnabledChannels[e.ChannelID].Roles {
		if strings.Contains(r.Reaction, e.Emoji.APIName()) {
			role = r

			// Get member obj
			member, err := s.GuildMember(e.GuildID, e.UserID)
			if err != nil {
				log.Printf("failed to get member for %v", e.UserID)
				return
			}

			// Check if member has this role
			hasRole := false
			for _, mRole := range member.Roles {
				if role.ID == mRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				// DM user
				validReaction = false
				dmChan, err := s.UserChannelCreate(e.UserID)
				if err != nil {
					// User DMs closed
					return
				}
				s.MessageReactionRemove(e.ChannelID, e.MessageID, e.Emoji.APIName(), e.UserID)
				s.ChannelMessageSend(dmChan.ID, fmt.Sprintf("You already have the %s role.", role.Name))
				return
			}
			validReaction = true
			break
		}
	}

	// Remove user reaction
	defer func() {
		if config.RemoveReactions || !validReaction {
			err := s.MessageReactionRemove(e.ChannelID, e.MessageID, e.Emoji.APIName(), e.UserID)
			if err != nil {
				log.Printf("failed to remove a reaction, %v", err.Error())
			}
		}
	}()

	// Return if no valid reaction
	if !validReaction {
		return
	}

	// Get user reations data
	userData, err := db.GetUserData(e.UserID)
	if err != nil {
		log.Printf("failed to get user data: %v", err)
		return
	}

	// Check if user has a valid pending role request
	activeReq := userData.RoleRequests[e.ChannelID][role.ID]
	_, err = s.ChannelMessage(guildSettings.VerificationChan, activeReq.VerificationID)

	if activeReq.Active && err == nil {
		validReaction = false
		dmChan, err := s.UserChannelCreate(e.UserID)
		if err != nil {
			// User DMs closed
			return
		}
		s.ChannelMessageSend(dmChan.ID, fmt.Sprintf("You have a pending request for %s role already.", role.Name))
		return
	}

	switch role.Verification {
	case "auto":
		err := s.GuildMemberRoleAdd(e.GuildID, e.UserID, role.ID)
		if err != nil {
			s.ChannelMessageSend(guildSettings.VerificationChan, fmt.Sprintf("Failed to automatically assign %v role to %v. Please do it manually and report this issue.\n```%v```.\n*This is most likely due to Botto role being below the role it is trying to assign.*", role.Name, fmt.Sprintf("<@%v>", e.UserID), err.Error()))
			return
		}
		return

	case "manual":
		// Check for nil map
		if userData.IsSoftBanned == nil {
			userData.IsSoftBanned = make(map[string]bool)
			userData.IsSoftBanned[e.GuildID] = false
		}

		// Check if user is soft banned
		if userData.IsSoftBanned[e.GuildID] {
			log.Printf("%v - user is soft banned", e.UserID)
			return
		}

		// Get channel info
		channel, err := s.Channel(channelSettings.ID)
		if err != nil {
			log.Print(err)
			channel.Name = "(failed to get channel name)"
		}

		// Construct message
		finalMsg := "**New role request:**\n" + fmt.Sprintf("%v requested by %v", role.Name, fmt.Sprintf("<@%v>", e.UserID))

		// Send Embed
		msg, err := s.ChannelMessageSend(guildSettings.VerificationChan, finalMsg)
		if err != nil {
			log.Printf("failed to send a message to the verification channel: %v", err)
			return
		}

		// Add Verification message to DB
		var verMsg db.VerificationMessage
		verMsg.ID = msg.ID
		verMsg.GuildID = msg.GuildID
		verMsg.RoleRequested = role
		verMsg.Timestamp = time.Now()
		verMsg.UserID = e.UserID
		verMsg.RequestChanID = e.ChannelID

		err = db.AddVerificationMsg(verMsg)
		if err != nil {
			log.Print("failed to generate a verification message")
			return
		}

		// Add a request to user
		newReq := db.Request{Active: true, Timestamp: time.Now(), VerificationID: msg.ID}

		// Check for nil maps
		if userData.RoleRequests == nil {
			chanMap := make(map[string]map[string]db.Request)
			chanMap[e.ChannelID] = make(map[string]db.Request)
			userData.RoleRequests = chanMap
		}
		if userData.RoleRequests[e.ChannelID] == nil {
			reqMap := make(map[string]db.Request)
			userData.RoleRequests[e.ChannelID] = reqMap
		}

		// Update user data
		userData.RoleRequests[e.ChannelID][role.ID] = newReq
		err = db.UpdateUserData(userData)
		if err != nil {
			log.Printf("failed to update user request status: %v", err)
		}

		// Add Reactions
		s.MessageReactionAdd(msg.ChannelID, msg.ID, config.ApproveReaction)
		s.MessageReactionAdd(msg.ChannelID, msg.ID, config.DenyReaction)
		return

	default:
		log.Print("bad verification method")
		return
	}

}

// VerificationReaction - Handle reaction event on verification message
func VerificationReaction(s *discordgo.Session, e *discordgo.MessageReactionAdd) {
	// Ignore self
	if e.UserID == s.State.User.ID {
		return
	}

	if !strings.Contains(config.ApproveReaction, e.Emoji.APIName()) || !strings.Contains(config.DenyReaction, e.Emoji.APIName()) {
		// Ignore all other reactions
		return
	}

	// Get request data
	reqMsg, err := s.ChannelMessage(e.ChannelID, e.MessageID)
	if err != nil {
		log.Print(err)
		return
	}

	if reqMsg.Author.ID != s.State.User.ID {
		return
	}

	perms, err := s.UserChannelPermissions(e.UserID, e.ChannelID)
	if err != nil {
		eventReplyDel(s, e.ChannelID, fmt.Sprintf("Failed to check your perms.\n```%v```", err), 5)
	}

	if perms&discordgo.PermissionManageRoles != discordgo.PermissionManageRoles {
		eventReplyDel(s, e.ChannelID, "You need to have Manage Roles perms to approve/deny this request.", 5)
		return
	}

	// Get guild settings
	guildSettings, err := db.GetGuildSettings(e.GuildID)
	if err != nil {
		eventReplyDel(s, e.ChannelID, fmt.Sprintf("An error occured while fetching guild settings.\n```%v```", err), 5)
		return
	}

	// Check if this is the verification channel
	if guildSettings.VerificationChan != e.ChannelID {
		return
	}

	requestData, err := db.GetVerificationMsg(e.MessageID)
	if err != nil {
		eventReplyDel(s, e.ChannelID, "Failed to find a valid request for this message.", 15)
		return
	}

	// Check reaction and message user / assign role
	if strings.Contains(config.ApproveReaction, e.Emoji.APIName()) {
		// Set request as complete
		db.CompleteUserRequest(requestData.UserID, requestData.RequestChanID, requestData.RoleRequested.ID)

		// Give role to user
		err := s.GuildMemberRoleAdd(e.GuildID, requestData.UserID, requestData.RoleRequested.ID)
		if err != nil {
			eventReplyDel(s, e.ChannelID, fmt.Sprintf("Failed to give the role. Please do it manually and report this issue.\n```%v```", err), 15)
		}

		// Edit request message
		s.ChannelMessageEdit(e.ChannelID, e.MessageID, fmt.Sprintf("%s\nAPPROVED by <@%v>", reqMsg.Content, e.UserID))
		s.MessageReactionsRemoveAll(e.ChannelID, e.MessageID)

		// Message user
		if config.MessageOnDeny {
			dmChan, err := s.UserChannelCreate(requestData.UserID)
			if err != nil {
				// User DMs closed
				return
			}
			s.ChannelMessageSend(dmChan.ID, fmt.Sprintf("Your request for %v role was approved.", requestData.RoleRequested.Name))
		}

		// Delete verification mesasge data from db
		db.DelVerificationMsg(requestData)
		return
	}
	if strings.Contains(config.DenyReaction, e.Emoji.APIName()) {
		// Set request as complete
		db.CompleteUserRequest(requestData.UserID, requestData.RequestChanID, requestData.RoleRequested.ID)

		// Edit request message
		s.ChannelMessageEdit(e.ChannelID, e.MessageID, fmt.Sprintf("%s\nDENIED by <@%v>", reqMsg.Content, e.UserID))
		s.MessageReactionsRemoveAll(e.ChannelID, e.MessageID)

		// Message user
		if config.MessageOnDeny {
			dmChan, err := s.UserChannelCreate(requestData.UserID)
			if err != nil {
				// User DMs closed
				return
			}
			s.ChannelMessageSend(dmChan.ID, fmt.Sprintf("Your request for %v role was denied.", requestData.RoleRequested.Name))
		}

		// Delete verification mesasge data from db
		db.DelVerificationMsg(requestData)
		return
	}
	log.Print("invalid reaction")
}

// GuildRoleUpdate - Handle role update event
func GuildRoleUpdate(s *discordgo.Session, e *discordgo.GuildRoleUpdate) {
	//
}

// GuildRoleDelete - Handle role deletion event
func GuildRoleDelete(s *discordgo.Session, e *discordgo.GuildRoleDelete) {
	//
}

// GuildCreate - Handle new guild joined event
func GuildCreate(s *discordgo.Session, e *discordgo.GuildCreate) {
	//
}

// GuildDelete - Handle kicked from guild event
func GuildDelete(s *discordgo.Session, e *discordgo.GuildDelete) {
	log.Print("Kicked")
}

func eventReplyDel(s *discordgo.Session, cid, msg string, timer time.Duration) error {
	newMsg, err := s.ChannelMessageSend(cid, msg)
	defer func() {
		time.Sleep(time.Second * timer)
		s.ChannelMessageDelete(cid, newMsg.ID)
	}()
	return err
}
