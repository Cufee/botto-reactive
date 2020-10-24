package handlers

import (
	"fmt"
	"log"
	"time"

	"github.com/Necroforger/dgrouter/exrouter"
	db "github.com/cufee/botto-reactive/database"
)

// SoftBanHandler - Soft ban a user
func SoftBanHandler(ctx *exrouter.Context) {
	users := ctx.Msg.Mentions
	if len(users) == 0 {
		ctx.Reply("Please include at least one user mention.")
		return
	}

	updateErrors := make(chan error, len(users))

	// Loop through users and ban
	for _, u := range users {
		// Get user data
		userData, err := db.GetUserData(u.ID)
		if err != nil {
			replyDel(ctx, fmt.Sprintf("Failed to get user data\n```%v```", err), 15)
			return
		}

		// Check for nil map
		if userData.IsSoftBanned == nil {
			userData.IsSoftBanned = make(map[string]bool)
		}
		// Set to banned
		userData.IsSoftBanned[ctx.Msg.GuildID] = true
		err = db.UpdateUserData(userData)
		if err != nil {
			updateErrors <- err
		}
	}
	close(updateErrors)
	if len(updateErrors) != 0 {
		// Log errors
		for e := range updateErrors {
			log.Print(e)
		}
		replyDel(ctx, fmt.Sprintf("%v errors occured while updating user settings, some or all user bans have failed.", len(updateErrors)), 15)
		return
	}
	replyDel(ctx, fmt.Sprintf("Banned %v users.", len(users)), 15)
	return
}

// SoftUnbanHandler - Remove a soft ban for user
func SoftUnbanHandler(ctx *exrouter.Context) {
	users := ctx.Msg.Mentions
	if len(users) == 0 {
		ctx.Reply("Please include at least one user mention.")
		return
	}
	updateErrors := make(chan error, len(users))

	// Loop through users and unban
	for _, u := range users {
		// Get user data
		userData, err := db.GetUserData(u.ID)
		if err != nil {
			replyDel(ctx, fmt.Sprintf("Failed to get user data\n```%v```", err), 15)
			return
		}

		// Check for nil map
		if userData.IsSoftBanned == nil {
			return
		}

		// Set to banned
		delete(userData.IsSoftBanned, ctx.Msg.GuildID)
		err = db.UpdateUserData(userData)
		if err != nil {
			updateErrors <- err
		}
	}
	close(updateErrors)
	if len(updateErrors) != 0 {
		// Log errors
		for e := range updateErrors {
			log.Print(e)
		}
		replyDel(ctx, fmt.Sprintf("%v errors occured while updating user settings, some or all user unbans have failed.", len(updateErrors)), 15)
		return
	}
	replyDel(ctx, fmt.Sprintf("Lifted a ban from %v users.", len(users)), 15)
	return
}

func replyDel(ctx *exrouter.Context, msg string, timer time.Duration) error {
	newMsg, err := ctx.Reply(msg)
	defer func() {
		time.Sleep(time.Second * timer)
		ctx.Ses.ChannelMessageDelete(ctx.Msg.ChannelID, newMsg.ID)
	}()
	return err
}
