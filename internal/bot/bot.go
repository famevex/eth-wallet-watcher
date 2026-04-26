package bot

import (
	"database/sql"

	"github.com/famevex/eth-wallet-watcher/internal/db"
	"gopkg.in/telebot.v3"
)

type Bot struct {
	db *sql.DB
	tg *telebot.Bot
}

func NewBot(db *sql.DB, tg *telebot.Bot) *Bot {
	return &Bot{
		db: db,
		tg: tg,
	}
}

func (*Bot) HandleStart(c telebot.Context) error {
	return c.Send("Hi, I'm eth-wallet-watcher-bot. Commands: /watch, /unwatch")
}

func (b *Bot) HandleWatch(c telebot.Context) error {
	chatID := c.Chat().ID
	address := c.Message().Payload
	if address == "" {
        return c.Send("Please include the address after the command. For example: /watch 0x...")
    }

	err := db.AddSubscription(b.db, chatID, address)
	if err != nil {
        return c.Send("Error adding subscription.")
    }

	return c.Send("Subscription added!")
}

func (b *Bot) HandleUnwatch(c telebot.Context) error {
	chatID := c.Chat().ID
	address := c.Message().Payload
	if address == "" {
        return c.Send("Please include the address after the command. For example: /watch 0x...")
    }

	err := db.RemoveSubscription(b.db, chatID, address)
	if err != nil {
        return c.Send("Error deleting subscription.")
    }

	return c.Send("Subscription deleted!")
}