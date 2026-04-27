package monitor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/famevex/eth-wallet-watcher/internal/db"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var transferTopic = common.HexToHash(
    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
)

type Alert struct {
    ChatID  int64
    Type    string  // "in" или "out"
    Asset   string  // "ETH" или "USDC"
    Amount  string
    From    string
    To      string
    TxHash  string
}

type Monitor struct {
	client *ethclient.Client
	db     *sql.DB
	alerts chan<- Alert
	usdc   common.Address
	chainID *big.Int
}

type NewHeadsNotification struct {
    Params struct {
        Result struct {
            Number     string `json:"number"`     // блок в hex: "0x123456"
            Hash       string `json:"hash"`       // хэш блока
            Timestamp  string `json:"timestamp"`  // время в hex
        } `json:"result"`
    } `json:"params"`
}

func NewMonitor(client *ethclient.Client, db *sql.DB, alerts chan<- Alert, usdc common.Address) *Monitor {
	// get chainID for signer in Eth transaction
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatal("cannot get chainID:", err)
	}
	return &Monitor{
		client: client,
		db: db,
		alerts: alerts,
		usdc: usdc,
		chainID: chainID,
	}
}

func (m *Monitor) Start (ctx context.Context) {
	godotenv.Load()
	wssURL := os.Getenv("ALCHEMY_WSS_URL")

	conn, _, err := websocket.DefaultDialer.Dial(wssURL, nil)
	if err != nil {
		log.Fatal("Connection error:", err)
	}
	defer conn.Close()
	go func() {
        <-ctx.Done() // wait for someone to call cancel()
        fmt.Println("Context cancelled, closing connection...")
        conn.Close() // forcefully break the socket
    }()

	subscribeMsg := `{
		"jsonrpc":"2.0",
		"id":1,
		"method":"eth_subscribe",
		"params":["newHeads"]
	}`

	// sending a subscription message
	err = conn.WriteMessage(websocket.TextMessage, []byte(subscribeMsg))
	if err != nil {
		log.Fatal("write error:", err)
	}

	for {
		// receiving messages from the server.
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Ошибка соединения с сервером Alchemy")
			return
		}

		// collect the necessary data about the new block
		var notification NewHeadsNotification
        if err := json.Unmarshal(message, &notification); err != nil {
            log.Printf("JSON parse error: %v", err)
            continue
        }
		if notification.Params.Result.Hash == "" {
			continue
		}

		go m.processETH(ctx, notification.Params.Result.Hash)

		blockNumHex := strings.TrimPrefix(notification.Params.Result.Number, "0x")
		blockNum := new(big.Int)
		blockNum.SetString(blockNumHex, 16)
		go m.processUSDC(ctx, blockNum)
	}
}

func (m *Monitor) processETH(ctx context.Context, blockHash string) {
	hash := common.HexToHash(blockHash)

	block, err := m.client.BlockByHash(ctx, hash)
    if err != nil {
        log.Printf("Failed to fetch block %s: %v", blockHash, err)
        return
    }

	subscriptions, err := m.getSubscriptions() // SELECT chat_id, wallet_address FROM subscriptions
    if err != nil {
        log.Printf("DB error: %v", err)
        return
    }

	signer := types.NewLondonSigner(m.chainID)
	
	for _, tx := range block.Transactions() {
        if tx.To() == nil { continue } // skip creating contracts

        from, err := types.Sender(signer, tx)
        if err != nil { continue }
        
        to := tx.To()

        for _, sub := range subscriptions {
            if strings.EqualFold(sub.Address, from.Hex()) {
                m.sendAlert(Alert {
					ChatID: sub.ChatID,
					Type: "out",
					Asset: "ETH",
					Amount: weiToEth(tx.Value()),
					From: from.Hex(),
					To: to.Hex(),
					TxHash: tx.Hash().Hex(),
				})
            }
            if strings.EqualFold(sub.Address, to.Hex()) {
               m.sendAlert(Alert {
					ChatID: sub.ChatID,
					Type: "in",
					Asset: "ETH",
					Amount: weiToEth(tx.Value()),
					From: from.Hex(),
					To: to.Hex(),
					TxHash: tx.Hash().Hex(),
				})
            }
        }
    }
}

func (m *Monitor) processUSDC(ctx context.Context, blockNum *big.Int) {
    query := ethereum.FilterQuery{
        FromBlock: blockNum,
        ToBlock:   blockNum,
        Addresses: []common.Address{m.usdc},
        Topics:    [][]common.Hash{{transferTopic}},
    }

    logs, err := m.client.FilterLogs(ctx, query)
    if err != nil {
		log.Printf("Get logs error: %v", err)
        return
	}
    subs, err := m.getSubscriptions()
    if err != nil {
		log.Printf("Get subscriptions error: %v", err)
        return
	}

    for _, logEntry := range logs {
        if len(logEntry.Topics) < 3 { continue }

        from := common.HexToAddress(logEntry.Topics[1].Hex())
        to   := common.HexToAddress(logEntry.Topics[2].Hex())
        
        amount := new(big.Int).SetBytes(logEntry.Data)
        txHash := logEntry.TxHash.Hex()

        for _, sub := range subs {
            if strings.EqualFold(sub.Address, from.Hex()) {
                m.sendAlert(Alert {
					ChatID: sub.ChatID,
					Type: "out",
					Asset: "USDC",
					Amount: usdcToFloat(amount),
					From: from.Hex(),
					To: to.Hex(),
					TxHash: txHash,
				})
            }
            if strings.EqualFold(sub.Address, to.Hex()) {
               m.sendAlert(Alert {
					ChatID: sub.ChatID,
					Type: "in",
					Asset: "USDC",
					Amount: usdcToFloat(amount),
					From: from.Hex(),
					To: to.Hex(),
					TxHash: txHash,
				})
            }
        }
    }
}

func (m *Monitor) getSubscriptions () ([]db.Subscription, error) {
	subscriptions, err := db.GetAllSubscriptions(m.db)
	if err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (m *Monitor) sendAlert (alert Alert){
	log.Printf("ALERT: chat=%d, %s %s, %s → %s, tx=%s",
        alert.ChatID,
        alert.Type,
        alert.Asset,
        shortAddr(alert.From),
        shortAddr(alert.To),
        shortAddr(alert.TxHash),
    )
	m.alerts <- alert
}

// 1 ETH = 1_000_000_000_000_000_000 wei (10^18)
func weiToEth(wei *big.Int) string {
    eth := new(big.Float).Quo(
        new(big.Float).SetInt(wei),
        new(big.Float).SetFloat64(1e18),
    )
    return fmt.Sprintf("%.6f ETH", eth)
}

// 1 USDC = 1_000_000 raw amount
func usdcToFloat(raw *big.Int) string {
    f := new(big.Float).Quo(
        new(big.Float).SetInt(raw),
        new(big.Float).SetFloat64(1e6),
    )
    return fmt.Sprintf("%.2f USDC", f)
}
func shortAddr(addr string) string {
    if len(addr) < 10 { return addr }
    return addr[:6] + "..." + addr[len(addr)-4:]
}