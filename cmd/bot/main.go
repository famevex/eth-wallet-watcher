package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/famevex/eth-wallet-watcher/internal/bot"
	"github.com/famevex/eth-wallet-watcher/internal/config"
	"github.com/famevex/eth-wallet-watcher/internal/db"
	"github.com/famevex/eth-wallet-watcher/internal/monitor"
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

	apiKey := os.Getenv("API_KEY")
	ethurl := "wss://eth-mainnet.g.alchemy.com/v2/" + apiKey

    ethClient, err := ethclient.Dial(ethurl)
    if err != nil {
        log.Fatalf("Ошибка подключения к ноде: %v", err)
    }

	alertChannel := make(chan monitor.Alert, 100)
	m := monitor.NewMonitor (
		ethClient,
		dbConnection,
		alertChannel,
		common.HexToAddress(conf.UsdcContract),
	)

	// сreate a context and a function to close the goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.Start(ctx)
	

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

	// create bot
	tbBot, err := telebot.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}
	appBot := bot.NewBot(dbConnection, tbBot)
	appBot.Register()

	
	log.Println("Bot started...")
	appBot.Start()
}