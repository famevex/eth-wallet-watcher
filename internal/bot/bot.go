package bot

import (
	"database/sql"
	"fmt"

	"github.com/famevex/eth-wallet-watcher/internal/db"
	"github.com/famevex/eth-wallet-watcher/internal/monitor"
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

func (b *Bot) Register() {
    b.tg.Handle("/start", b.HandleStart)
    b.tg.Handle("/watch", b.HandleWatch)
    b.tg.Handle("/unwatch", b.HandleUnwatch)
}

func (b *Bot) Start() {
    b.tg.Start()
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
	if !isValidAddress(address) {
		return c.Send("Invalid address")
	}
	err := db.AddSubscription(b.db, chatID, address)
	if err != nil {
		if err.Error() == "already exists" {
			return c.Send("You're already watching this address!")
		}

		return c.Send("An error occurred while saving to the database.")
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

func isValidAddress(address string) bool {
    return len(address) == 42 && address[:2] == "0x"
}

func (b *Bot) ListenAlerts(alerts <-chan monitor.Alert) {
    for alert := range alerts {
        msg := fmt.Sprintf(
            "🔔 *%s Transaction*\nType: %s\nAmount: %s\nFrom: `%s`\nTo: `%s`\nEtherscan: https://etherscan.io/tx/%s",
            alert.Asset,
            alert.Type,
            alert.Amount,
            alert.From,
            alert.To,
            alert.TxHash,
        )
        chat := &telebot.Chat{ID: alert.ChatID}
        b.tg.Send(chat, msg, telebot.ModeMarkdown)
    }
}