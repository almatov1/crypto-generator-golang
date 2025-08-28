package main

import (
	"bufio"
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tyler-smith/go-bip39"
)

const (
	defaultDB  = "database.db"
	defaultLog = "log.txt"
)

var (
	counter   uint64
	stopFlag  uint32
	startTime = time.Now()
)

func main() {
	maxThreads := runtime.NumCPU()
	fmt.Printf("В системе обнаружено %d логических процессоров.\n", maxThreads)

	var userThreads int
	fmt.Println("Введите желаемое количество потоков (минимум 2):")
	fmt.Scan(&userThreads)
	if userThreads < 2 {
		userThreads = 2
	}
	if userThreads > maxThreads {
		userThreads = maxThreads
	}

	// Открываем базу
	db, err := sql.Open("sqlite3", defaultDB)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Включаем оптимизации SQLite
	db.Exec("PRAGMA journal_mode=WAL;")
	db.Exec("PRAGMA synchronous=NORMAL;")
	db.Exec("PRAGMA temp_store=MEMORY;")

	// Готовим statement для проверки адресов
	stmt, err := db.Prepare("SELECT 1 FROM addresses WHERE address = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	// Готовим файл логов
	logFile, err := os.OpenFile(defaultLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	writer := bufio.NewWriter(logFile)

	// Скорость
	go func() {
		for atomic.LoadUint32(&stopFlag) == 0 {
			total := atomic.LoadUint64(&counter)
			mins := time.Since(startTime).Minutes()
			speed := float64(total) / (mins + 1e-9)
			fmt.Printf("\rСкорость: %.2f адресов/мин", speed)
			time.Sleep(time.Second)
		}
	}()

	// Ctrl+C для завершения
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		atomic.StoreUint32(&stopFlag, 1)
	}()

	// Запускаем воркеров
	var wg sync.WaitGroup
	for i := 0; i < userThreads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for atomic.LoadUint32(&stopFlag) == 0 {
				entropy, _ := bip39.NewEntropy(128)
				mnemonic, _ := bip39.NewMnemonic(entropy)
				seed := bip39.NewSeed(mnemonic, "")

				// Приватный ключ
				privKey, err := crypto.ToECDSA(seed[:32])
				if err != nil {
					continue
				}
				pubKey := privKey.Public().(*ecdsa.PublicKey)
				addr := crypto.PubkeyToAddress(*pubKey)

				atomic.AddUint64(&counter, 1)

				// Проверка адреса через prepared statement
				if checkAddress(stmt, addr.Bytes()) {
					saveToFile(writer, mnemonic, addr.Hex(), "0x"+hex.EncodeToString(privKey.D.Bytes()))
				}
			}
		}()
	}
	wg.Wait()

	fmt.Println("\nПрограмма завершена.")
}

// Проверка адреса в БД через prepared statement
func checkAddress(stmt *sql.Stmt, addr []byte) bool {
	row := stmt.QueryRow(addr)
	var dummy int
	err := row.Scan(&dummy)
	return err == nil
}

// Сохранение в лог
func saveToFile(w *bufio.Writer, phrase, addr, priv string) {
	w.WriteString(fmt.Sprintf("Seed: %s\n", phrase))
	w.WriteString(fmt.Sprintf("Address: %s\n", addr))
	w.WriteString(fmt.Sprintf("Private: %s\n\n", priv))
	w.Flush()
}
