package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
	telebot "gopkg.in/telebot.v3"
)

func main() {
	godotenv.Load()
	telegram_token := os.Getenv("TELEGRAM_TOKEN")
	proxy_url := os.Getenv("PROXY_URL")
	
	proxyURL, err := url.Parse(proxy_url) // bring link to the type *url.URL (for Client)
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
		Token: telegram_token,
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