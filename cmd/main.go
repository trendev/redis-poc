package main

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/trendev/redis-poc/internal/data"
)

func main() {
	msg := "Hello trendev"
	var wg sync.WaitGroup

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			fmt.Printf("🚀 client#%.3d started\n", i)
			v := fmt.Sprintf("%s - client#%.3d", msg, i)
			for {
				m := data.Message{Value: v}
				b, err := json.Marshal(m)
				if err != nil {
					panic(err)
				}
				buf := bytes.NewBuffer(b)
				resp, err := http.Post(fmt.Sprintf("http://localhost:808%d/jsie", i%2), "application/json", buf)
				if err != nil || resp.StatusCode != http.StatusOK {
					fmt.Printf("❌ request of client#%.3d failed\n💀 client#%.3d is dead\n", i, i)
					return
				}
				time.Sleep(3 * time.Second)
			}
		}(i)

	}

	wg.Wait() // wait all go routines over
	fmt.Println("🙌 redis client is stopped")
}
