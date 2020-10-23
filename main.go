package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	db "github.com/cufee/botto-reactive/database"
	"github.com/cufee/botto-reactive/handlers"

	"github.com/Necroforger/dgrouter/exmiddleware"
	"github.com/Necroforger/dgrouter/exrouter"
	"github.com/bwmarrin/discordgo"
)

var (
	token  string
	prefix string = "!"
)

func init() {
	var err error
	token, err = loadToken("token.dat")
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Event handlers for Reactive Roles
	dg.AddHandler(handlers.ReactionRoleCheck)
	dg.AddHandler(handlers.VerificationReaction)
	dg.AddHandler(handlers.GuildRoleUpdate)
	dg.AddHandler(handlers.GuildRoleDelete)

	// Dev events
	dg.AddHandler(handlers.GuildCreate)
	dg.AddHandler(handlers.GuildDelete)

	// Router
	router := exrouter.New()

	// Command groups
	router.Group(func(r *exrouter.Route) {
		// Category
		r.Cat("setup")

		// Set cooldown
		r.Use(
			exmiddleware.UserCooldown(time.Second*10, exmiddleware.CatchReply("This command is on cooldown...")),
			modCheck,
		)

		// Commands
		r.On("setup", (handlers.SetupHandler)).Desc("Setup Reactive Roles.")
		r.On("add", (handlers.AddHandler)).Desc("Add a role to the RR list.")
		r.On("remove", (handlers.RemoveHandler)).Desc("Remove a role from the RR list.")
	})

	// Help command
	router.Group(func(r *exrouter.Route) {
		// Category
		r.Cat("help")

		// Set cooldown
		r.Use(exmiddleware.UserCooldown(time.Second*5, exmiddleware.CatchReply("This command is on cooldown...")))

		// Help command
		r.Default = r.On("help", func(ctx *exrouter.Context) {
			// Get longest command for formatting
			var longestCmd int
			for _, v := range router.Routes {
				if len(v.Name) > longestCmd {
					longestCmd = len(v.Name)
				}
			}

			var text = ""
			for _, v := range router.Routes {
				spceCnt := longestCmd - len(v.Name)
				space := ""
				for i := 0; i <= spceCnt; i++ {
					space = space + " "
				}
				text += prefix + v.Name + space + ": " + v.Description + "\n"
			}
			ctx.Reply("```" + text + "```")
		}).Desc("Prints this help menu.")
	})

	// Add message handler
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		router.FindAndExecute(dg, prefix, s.State.User.ID, m.Message)
	})

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Botto is now running. Press CTRL-C to exit.")
	// Wait for the user to cancel the process
	defer func() {
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
		<-sc
		dg.Close()
		// Close DB connection
		db.CloseDB()
	}()
}

func modCheck(fn exrouter.HandlerFunc) exrouter.HandlerFunc {
	return func(ctx *exrouter.Context) {
		userPerms, err := ctx.Ses.UserChannelPermissions(ctx.Msg.Author.ID, ctx.Msg.ChannelID)
		if err != nil {
			log.Printf("failed to check user perms: %v", err)
		}

		if userPerms&discordgo.PermissionManageRoles == discordgo.PermissionManageRoles {
			fn(ctx)
			return
		}

		ctx.Reply("You don't have permission to use this command")
	}
}

// Loads a discord token from filename
func loadToken(filename string) (string, error) {
	// Open file
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	// Scan for token
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s := scanner.Text()
		if strings.TrimSpace(s) != "" {
			return s, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	// Token not found
	return "", fmt.Errorf("%v did not contain a token", filename)
}
