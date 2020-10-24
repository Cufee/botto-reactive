package handlers

import (
	"fmt"
	"log"
	"strings"

	"regexp"

	"github.com/Necroforger/dgrouter/exrouter"
	"github.com/bwmarrin/discordgo"
	"github.com/cufee/botto-reactive/config"
	db "github.com/cufee/botto-reactive/database"
)

// AddHandler - just a test
func AddHandler(ctx *exrouter.Context) {
	// Delete messae
	ctx.Ses.ChannelMessageDelete(ctx.Msg.ChannelID, ctx.Msg.ID)

	// Get guild settings
	guildSettings, err := db.GetGuildSettings(ctx.Msg.GuildID)
	if err != nil {
		replyDel(ctx, fmt.Sprintf("An error occured while fetching guild settings.\n```%v```", err), 15)
		return
	}

	// Get channel and message
	channel := guildSettings.EnabledChannels[ctx.Msg.ChannelID]
	msg, err := ctx.Ses.ChannelMessage(ctx.Msg.ChannelID, channel.MessageID)
	if err != nil {
		log.Print(err)
		replyDel(ctx, "Something did not work.", 15)
		return
	}
	if msg == nil {
		replyDel(ctx, "It looks like there is no Reactive Roles message in this channel. Try !setup.", 15)
		return
	}

	// Get role
	roleArg := ctx.Args.Get(1)
	if roleArg == "" || !strings.HasPrefix(roleArg, "<@&") {
		replyDel(ctx, "Make sure the role you want to add is a mention and is the first argument after this command.", 15)
		return
	}

	// Regex to get role ID
	re := regexp.MustCompile(`[<@&>]`)
	roleID := re.ReplaceAllString(roleArg, "")

	guildRoles, err := ctx.Ses.GuildRoles(guildSettings.ID)
	if err != nil {
		replyDel(ctx, "I was not able to get a list of roles on this server.", 15)
		return
	}

	// Find role in guild roles
	var gldRole *discordgo.Role
	for _, r := range guildRoles {
		if r.ID == roleID {
			gldRole = r
			break
		}
	}

	// Role not found
	if gldRole == nil {
		replyDel(ctx, "I was not able to find the role on this server.", 15)
		return
	}

	// Get verification type
	verArg := ctx.Args.Get(2)
	if !sliceContains(config.VerTypes, verArg) {
		replyDel(ctx, "Make sure the verification type is either `auto` or `manual` and is the second argument after this command.", 15)
		return
	}

	// Get reaction name
	emojiArgRaw := ctx.Args.Get(3)
	emojiArg := re.ReplaceAllString(emojiArgRaw, "")
	if emojiArg == "" {
		replyDel(ctx, "Make sure the emote you want to bind is the third argument after this command.", 15)
		return
	}

	// Set role settings
	var role db.Role
	role.ID = gldRole.ID
	role.Reaction = emojiArgRaw
	role.ReactionFixed = emojiArg
	role.Name = gldRole.Name
	role.Verification = verArg

	// Check for nil map
	if guildSettings.EnabledChannels[channel.ID].Roles == nil {
		channel.Roles = make(map[string]*db.Role)
		guildSettings.EnabledChannels[channel.ID] = channel
	}

	// Add role reaction to settings
	guildSettings.EnabledChannels[channel.ID].Roles[role.ID] = &role
	err = db.UpdateGuildSettings(guildSettings)
	if err != nil {
		replyDel(ctx, "Failed to update channel settings.", 15)
		return
	}

	// Update RR message
	err = updateReactiveMsg(ctx, guildSettings.EnabledChannels[channel.ID].Roles, channel.ID, channel.MessageID)
	if err != nil {
		replyDel(ctx, "Failed to update the Reative Roles message.", 15)
		return
	}

	// Add reaction
	err = ctx.Ses.MessageReactionAdd(ctx.Msg.ChannelID, channel.MessageID, emojiArg)
	if err != nil {
		replyDel(ctx, fmt.Sprintf("Failed to add a reaction.\n```%v```", err), 5)
		return
	}

	replyDel(ctx, fmt.Sprintf("Done! `%v` is now a Reactive Role in this channel.", role.Name), 15)
}

// RemoveHandler - just a test
func RemoveHandler(ctx *exrouter.Context) {
	// Delete messae
	ctx.Ses.ChannelMessageDelete(ctx.Msg.ChannelID, ctx.Msg.ID)

	// Get guild settings
	guildSettings, err := db.GetGuildSettings(ctx.Msg.GuildID)
	if err != nil {
		replyDel(ctx, fmt.Sprintf("An error occured while fetching guild settings.\n```%v```", err), 15)
		return
	}

	// Get role
	roleArg := ctx.Args.Get(1)
	if roleArg == "" || !strings.HasPrefix(roleArg, "<@&") {
		replyDel(ctx, "Make sure the role you want to add is a mention and is the first argument after this command.", 15)
		return
	}

	// Regex to get role ID
	re := regexp.MustCompile(`[<@&>]`)
	roleID := re.ReplaceAllString(roleArg, "")

	guildRoles, err := ctx.Ses.GuildRoles(guildSettings.ID)
	if err != nil {
		replyDel(ctx, "I was not able to get a list of roles on this server.", 15)
		return
	}

	// Find role in guild roles
	var gldRole *discordgo.Role
	for _, r := range guildRoles {
		if r.ID == roleID {
			gldRole = r
			break
		}
	}

	// Role not found
	if gldRole == nil {
		replyDel(ctx, "I was not able to find the role on this server.", 15)
		return
	}

	// Check if role is in map
	role, ok := guildSettings.EnabledChannels[ctx.Msg.ChannelID].Roles[gldRole.ID]
	if !ok {
		replyDel(ctx, fmt.Sprintf("`%v` is not a reactive role in this channel.", gldRole.Name), 15)
		return
	}

	// Delete role
	delete(guildSettings.EnabledChannels[ctx.Msg.ChannelID].Roles, gldRole.ID)
	err = db.UpdateGuildSettings(guildSettings)
	if err != nil {
		replyDel(ctx, "Failed to update channel settings.", 15)
		return
	}

	// Update RR message
	err = updateReactiveMsg(ctx, guildSettings.EnabledChannels[ctx.Msg.ChannelID].Roles, ctx.Msg.ChannelID, guildSettings.EnabledChannels[ctx.Msg.ChannelID].MessageID)
	if err != nil {
		replyDel(ctx, "Failed to update the Reative Roles message.", 15)
		return
	}
	replyDel(ctx, fmt.Sprintf("Done! `%v` is no longer a Reactive Role in this channel.", role.Name), 15)
	return
}

// SetupHandler - just a test
func SetupHandler(ctx *exrouter.Context) {
	// Check bot perms
	if ok := permsCheck(ctx, ctx.Msg.ChannelID); !ok {
		replyDel(ctx, "I do not have proper perms in this channel for Reactive Roles to work.", 15)
		return
	}

	// Elevate Bot role

	// Delete messae
	ctx.Ses.ChannelMessageDelete(ctx.Msg.ChannelID, ctx.Msg.ID)

	// Get guild settings
	guildSettings, err := db.GetGuildSettings(ctx.Msg.GuildID)
	if err != nil {
		replyDel(ctx, (fmt.Sprintf("An error occured while fetching guild settings.\n```%v```", err)), 15)
		return
	}

	// Setup guild verification channel
	if guildSettings.VerificationChan == "" {
		// Get verification channel
		verArg := ctx.Args.Get(1)
		if verArg == "" {
			replyDel(ctx, "Make sure to specify the #verification channel you would like to use as the first argument after this command.\nYour verification channel needs to be visible to the moderators only.", 15)
			return
		}

		// Regex to get role ID
		re := regexp.MustCompile(`[<#>]`)
		verChanID := re.ReplaceAllString(verArg, "")

		// Find the channel privided
		guildChans, err := ctx.Ses.GuildChannels(ctx.Msg.GuildID)
		if err != nil {
			replyDel(ctx, "I was not able to get a list of channels on this server.", 15)
			return
		}
		for _, c := range guildChans {
			if c.ID == verChanID {
				guildSettings.VerificationChan = c.ID
				break
			}
		}

		// Channel not found/not valid
		if guildSettings.VerificationChan == "" {
			replyDel(ctx, "I was not able to find that channel on this server.", 15)
			return
		}

		// Check verification channel perms
		permsBool := permsCheck(ctx, verChanID)
		if !permsBool {
			replyDel(ctx, "It looks like I do not have proper perms in the verification channel provided.", 15)
			return
		}

	}

	// Check if this channel is already enabled
	channel, msg := getChanAndMsg(guildSettings, ctx)
	if msg != nil {
		replyDel(ctx, "This channel is already enabeled.", 15)
		return
	}

	// Set channel data if not set
	channel.ID = ctx.Msg.ChannelID
	// Send a new message and save ID
	newMsg, err := ctx.Reply("Do not delete this message. It will be edited each time you add or remove a reactive role.")
	if err != nil {
		log.Print(err)
		// Just in case this was a one time error
		replyDel(ctx, fmt.Sprintf("Failed to send a message to this channel."), 15)
		return
	}
	channel.MessageID = newMsg.ID

	// Update guild settings
	if guildSettings.EnabledChannels == nil {
		// Create new map
		guildSettings.EnabledChannels = make(map[string]db.ReactiveChannel)
	}
	guildSettings.EnabledChannels[channel.ID] = channel

	err = db.UpdateGuildSettings(guildSettings)
	if err != nil {
		replyDel(ctx, "Failed to update guild settings. Please try again later.", 15)
		return
	}

	replyDel(ctx, "Setup complete. You can add reactive roles with `!add @role verification-type emote`.\n*Available verification types: auto, manual*", 15)
	return
}

func updateReactiveMsg(ctx *exrouter.Context, roles map[string]*db.Role, cid string, mid string) error {
	// Clear message reactions
	ctx.Ses.MessageReactionsRemoveAll(cid, mid)

	// Delete message if there are no roles
	if len(roles) == 0 {
		_, err := ctx.Ses.ChannelMessageEdit(cid, mid, "There are no Reactive Roles in this channel. You can delete this message.")
		return err
	}

	// Compile new message
	var newMsg string = fmt.Sprintf("%s\n", config.ReactiveMsgHeader)

	var i int
	for _, r := range roles {
		newMsg = newMsg + fmt.Sprintf("%v - %v", r.Reaction, r.Name)
		// Add newline unless it's last line
		if i < len(roles) {
			newMsg = newMsg + "\n"
		}
		i++

		// Add reaction
		ctx.Ses.MessageReactionAdd(cid, mid, r.ReactionFixed)
	}

	// Edit the message
	_, err := ctx.Ses.ChannelMessageEdit(cid, mid, newMsg)
	return err
}

func permsCheck(ctx *exrouter.Context, chanID string) bool {
	// Check bot perms
	perms, err := ctx.Ses.UserChannelPermissions(ctx.Ses.State.User.ID, chanID)
	if err != nil || perms < config.PermsCode {
		log.Print(err)
		return false
	}
	return true
}

func getChanAndMsg(gs db.GuildSettings, ctx *exrouter.Context) (channel db.ReactiveChannel, msg *discordgo.Message) {
	// Loop through all channels
	for _, c := range gs.EnabledChannels {
		if c.ID == ctx.Msg.ChannelID {
			channel = c
			msg, _ = ctx.Ses.ChannelMessage(c.ID, c.MessageID)
			break
		}
	}
	return channel, msg
}

func sliceContains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
