package storage

import (
	"fmt"
	"time"

	"github.com/grafana/loki-client-go/loki"
	"github.com/prometheus/common/model"
)

var client *loki.Client

func InitLoki(url string) error {
	config, err := loki.NewDefaultConfig(url)
	if err != nil {
		return err
	}

	config.BatchWait = 2 * time.Second
	config.BatchSize = 1024 * 1024 // 1MB batching
	
	c, err := loki.New(config)
	if err != nil {
		return err
	}
	
	client = c
	return nil
}

// SendLogs accepts the raw JSON byte slice and pushes it to Loki
func SendLogs(logData []byte, name string, t time.Time, serviceEnv *string) {
	if client == nil {
		fmt.Println("Loki client not initialized")
		return
	}

	env := "nil"
	if serviceEnv != nil {
		env = *serviceEnv
	}

	// Set static labels for the stream
	labels := model.LabelSet{
		"service_name": model.LabelValue(name),
		"env": model.LabelValue(env),
	}

	err := client.Handle(labels, t, string(logData))
	if err != nil {
		fmt.Printf("Error pushing to Loki: %v\n", err)
	}
}

func Close() {
	if client != nil {
		client.Stop()
	}
}