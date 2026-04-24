package main

import (
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/famevex/eth-wallet-watcher/internal/config"
	"github.com/famevex/eth-wallet-watcher/internal/db"
	telebot "gopkg.in/telebot.v3"
)

func main() {
	conf, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	dbConnection, err := db.Connect(conf.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer dbConnection.Close()

	if err := db.RunMigrations(dbConnection); err != nil {
		log.Fatal(err)
	}
	

	proxyURL, err := url.Parse(conf.ProxyURL) // bring link to the type *url.URL (for Client)
	if err != nil {
		log.Fatal(err)
	}

	// сreate a client that works through a proxy
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	// set up the bot settings
	pref := telebot.Settings{
		Token: conf.TelegramToken,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		Client: client,
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}


	// /start
	b.Handle("/start", func(c telebot.Context) error {
		return c.Send("Hi, I'm eth-wallet-watcher-bot")
	})

	// plain text
	b.Handle(telebot.OnText, func(c telebot.Context) error {
		return c.Send("You wrote: " + c.Text())
	})


	log.Println("Bot started...")
	b.Start()
}