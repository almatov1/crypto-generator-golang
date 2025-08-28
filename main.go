package main

import (
	"context"
	"database/sql"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/mattn/go-sqlite3"
)

var rpcs = []string{
	"https://1rpc.io/eth",
	"https://eth-mainnet.public.blastapi.io/",
	"https://go.getblock.us/27eb23f40b964c9bb71b62f721e594e7",
	"https://0xrpc.io/eth",
	"https://ethereum.publicnode.com",
	"https://api.blockeden.xyz/eth/67nCBdZQSH9z3YqDDjdm",
	"https://ethereum.blockpi.network/v1/rpc/public",
	"https://cloudflare-eth.com/v1/mainnet",
	"https://rpc.flashbots.net/",
	"https://public-eth.nownodes.io/",
	"https://eth.api.onfinality.io/public",
	"https://ethereum-rpc.polkachu.com/",
	"https://eth-mainnet.reddio.com/",
}

func connectRPC() *ethclient.Client {
	for {
		for _, url := range rpcs {
			client, err := ethclient.Dial(url)
			if err != nil {
				log.Println("Ошибка подключения к RPC:", url, err)
				continue
			}
			log.Println("Подключено к RPC:", url)
			return client
		}
		log.Println("Не удалось подключиться ни к одной RPC. Ждем 5 секунд...")
		time.Sleep(5 * time.Second)
	}
}

func main() {
	db, err := sql.Open("sqlite3", "./wallets.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS wallets (
        address BLOB PRIMARY KEY
    )`)
	if err != nil {
		log.Fatal(err)
	}

	client := connectRPC()
	defer client.Close()

	maxRows := 200_000_000
	lastBlockHeader, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	startBlock := lastBlockHeader.Number.Uint64() // начинаем с последнего блока

	for i := startBlock; i > 0; i-- {
		block, err := client.BlockByNumber(context.Background(), big.NewInt(int64(i)))
		if err != nil {
			log.Println("Ошибка получения блока:", err)
			client = connectRPC()
			continue
		}

		inserted := 0
		for _, tx := range block.Transactions() {
			// from
			var from common.Address
			chainID := tx.ChainId()
			if chainID != nil && chainID.Sign() != 0 {
				from, err = types.Sender(types.LatestSignerForChainID(chainID), tx)
				if err != nil {
					continue
				}
			} else {
				// fallback для "старых" или странных tx
				from, err = types.Sender(types.HomesteadSigner{}, tx)
				if err != nil {
					continue
				}
			}

			if err == nil {
				_, _ = db.Exec("INSERT OR IGNORE INTO wallets(address) VALUES(?)", from.Bytes())
				inserted++
			}

			// to
			if tx.To() != nil {
				_, _ = db.Exec("INSERT OR IGNORE INTO wallets(address) VALUES(?)", tx.To().Bytes())
				inserted++
			}
		}

		var count int
		_ = db.QueryRow("SELECT COUNT(*) FROM wallets").Scan(&count)
		log.Printf("Блок %d обработан, новых адресов: %d, всего адресов: %d", i, inserted, count)

		if count >= maxRows {
			log.Println("Достигнуто максимальное количество строк:", maxRows)
			return
		}

		time.Sleep(100 * time.Millisecond)
	}

}
