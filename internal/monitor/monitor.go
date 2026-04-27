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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/famevex/eth-wallet-watcher/internal/db"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
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
        <-ctx.Done() // Ждем, пока кто-то вызовет cancel()
        fmt.Println("Контекст отменен, закрываем соединение...")
        conn.Close() // Принудительно рвем сокет
    }()

	subscribeMsg := `{
		"jsonrpc":"2.0",
		"id":1,
		"method":"eth_subscribe",
		"params":["newHeads"]
	}`

	err = conn.WriteMessage(websocket.TextMessage, []byte(subscribeMsg))
	if err != nil {
		log.Fatal("write error:", err)
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Ошибка соединения с сервером Alchemy")
			return
		}

		var notification NewHeadsNotification
        if err := json.Unmarshal(message, &notification); err != nil {
            log.Printf("JSON parse error: %v", err)
            continue
        }

		go m.processETH(ctx, notification.Params.Result.Hash)
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
        if tx.To() == nil { continue } // пропускаем создание контрактов

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

func shortAddr(addr string) string {
    if len(addr) < 10 { return addr }
    return addr[:6] + "..." + addr[len(addr)-4:]
}