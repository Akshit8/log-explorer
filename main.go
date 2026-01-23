package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/Akshit8/log-explorer/config"
	log_parser "github.com/Akshit8/log-explorer/log-parser"
	"github.com/Akshit8/log-explorer/tail"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Config Error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var wg sync.WaitGroup

	clients := []*tail.TailLogClient{}

	for _, svc := range cfg.Services {
		wg.Add(1)
		go func() {
			log.Printf("Service: %s, Account ID: %s, API Token: %s, Env: %v", svc.Name, svc.AccountID, svc.APIToken, svc.Env)

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
					parsedLog, err := log_parser.Parse(logEntry)
					if err != nil {
						log.Printf("Failed to parse log: %v", err)
						continue
					}

					for _, log := range parsedLog {
						fmt.Printf("[%s] %s | %d | %s -> %s\n",
							log.Time.Format("15:04:05"),
							log.Level,
							log.Status,
							log.URL,
							log.Message)
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
}
