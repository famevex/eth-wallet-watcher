# eth-wallet-watcher

A Telegram bot that monitors Ethereum wallet addresses and sends real-time notifications for incoming and outgoing ETH and USDC transactions.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat&logo=postgresql)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker)

## Features

- Subscribe to any Ethereum wallet address via Telegram
- Real-time notifications for **ETH** transfers using WebSocket (`eth_subscribe newHeads`)
- Real-time notifications for **USDC** transfers using `eth_getLogs`
- Direct Etherscan link in every notification
- Multiple users can watch the same address independently
- Clean package structure following Go conventions

## Tech Stack

- **Go** — core language
- **go-ethereum** — Ethereum client and types
- **telebot v3** — Telegram Bot API
- **PostgreSQL** — storing user subscriptions
- **Alchemy** — Ethereum node provider (WebSocket)
- **Docker Compose** — running PostgreSQL locally

## Project Structure

```
eth-wallet-watcher/
├── cmd/
│   └── bot/
│       └── main.go        # Entry point
├── internal/
│   ├── bot/
│   │   └── bot.go         # Telegram handlers and alert listener
│   ├── config/
│   │   └── config.go      # Environment config loader
│   ├── db/
│   │   └── db.go          # PostgreSQL connection and queries
│   └── monitor/
│       └── monitor.go     # Ethereum block monitor (ETH + USDC)
├── docker-compose.yml
├── .env.example
└── README.md
```

## Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- [Docker](https://www.docker.com/) and Docker Compose
- [Alchemy](https://www.alchemy.com/) account — create an app on **Ethereum Mainnet**, copy the **WebSocket** URL
- Telegram bot token from [@BotFather](https://t.me/BotFather)

## Setup

**1. Clone the repository**
```bash
git clone https://github.com/your-username/eth-wallet-watcher.git
cd eth-wallet-watcher
```

**2. Configure environment**
```bash
cp .env.example .env
```

Fill in `.env`:
```env
TELEGRAM_TOKEN=your_telegram_bot_token
PROXY_URL=socks5://127.0.0.1:10808   # optional, required if Telegram is blocked in your region
DATABASE_URL=postgres://postgres:postgres@localhost:5432/wallet_watcher?sslmode=disable
ALCHEMY_WSS_URL=wss://eth-mainnet.g.alchemy.com/v2/your_api_key
USDC_CONTRACT=0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48
```
> **Note:** `DATABASE_URL` uses `postgres` as the host — this is the Docker service name, not `localhost`.
>
> **Note:** `host.docker.internal` points to your host machine from inside the container. On Linux you may need to add `extra_hosts: ["host.docker.internal:host-gateway"]` to the bot service in `docker-compose.yml`.

**3. Run everything**
```bash
docker compose up --build
```

That's it. Docker will start PostgreSQL first, wait until it's healthy, then start the bot.

## Usage

| Command | Description |
|---|---|
| `/start` | Show available commands |
| `/watch 0x...` | Start monitoring a wallet address |
| `/unwatch 0x...` | Stop monitoring a wallet address |

**Example notification:**
```
🔔 ETH Transaction
Type: out
Amount: 1.250000 ETH
From: 0x28C6c06298d514Db089934071355E5743bf21d60
To: 0x9307f94A478f53b7e632198c756F1f4019A1b537
Etherscan: https://etherscan.io/tx/0x3a83...
```

## How It Works

1. Bot connects to Alchemy via WebSocket and subscribes to `newHeads` — a stream of new Ethereum block headers
2. For each new block, two goroutines run in parallel:
   - `processETH` — fetches the full block and scans all transactions for watched addresses
   - `processUSDC` — calls `eth_getLogs` filtered by the USDC contract and `Transfer` event topic
3. When a match is found, an `Alert` is sent to a buffered channel
4. `ListenAlerts` reads from the channel and sends a Telegram message to the subscriber

## License

MIT
