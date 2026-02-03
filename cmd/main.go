package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/Akshit8/log-explorer/internal/parser"
	"github.com/Akshit8/log-explorer/internal/storage"
	"github.com/Akshit8/log-explorer/internal/tail"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Config Error: %v", err)
	}

	err = storage.InitLoki("http://localhost:3100/loki/api/v1/push")
	if err != nil {
		log.Fatalf("Failed to init Loki: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var wg sync.WaitGroup

	clients := []*tail.TailLogClient{}

	for _, svc := range cfg.Services {
		wg.Add(1)
		go func() {
			client := tail.NewTailLogClient(svc.AccountID, svc.APIToken, svc.Name, nil)
			clients = append(clients, client)

			session, err := client.CreateSession(ctx)
			if err != nil {
				log.Fatal(err)
			}

			logs, err := client.StreamLogs(ctx, session.Result.URL)
			if err != nil {
				log.Fatal(err)
			}

			go func() {
				defer wg.Done()
				for logEntry := range logs {
					parsedLog, err := parser.Parse(logEntry)
					if err != nil {
						log.Printf("Failed to parse log: %v", err)
						continue
					}

					for _, l := range parsedLog {
						fmt.Printf("[%s] %s | %d | %s -> %s\n",
							l.Time.Format("15:04:05"),
							l.Level,
							l.Status,
							l.URL,
							l.Message)

						rawLog, err := json.Marshal(l)
						if err != nil {
							log.Printf("Failed to marshal log: %v", err)
							continue
						}

						fmt.Println(string(rawLog), svc.Name)
						storage.SendLogs(rawLog, svc.Name, l.Time, svc.Env)
					}
				}
			}()
		}()
	}

	<-ctx.Done()
	fmt.Println("Interrupted, shutting down...")

	for _, client := range clients {
		client.Close()
	}

	wg.Wait()

	storage.Close()
}
