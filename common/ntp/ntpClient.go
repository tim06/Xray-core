package ntp

import (
	"fmt"
	"github.com/beevik/ntp"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type NTPClient struct {
	ntpServers      []string
	offset          time.Duration
	mutex           sync.RWMutex
	lastSynced      time.Time
	synced          atomic.Bool
	syncAttemptsDone atomic.Bool
}

func NewNTPClient(ntpServers ...string) *NTPClient {
	client := &NTPClient{
		ntpServers: ntpServers,
	}
	go client.UpdateTime()
	return client
}

func (c *NTPClient) UpdateTime() {
	if c.syncAttemptsDone.Load() {
		return
	}

	var wg sync.WaitGroup
	for _, server := range c.ntpServers {
		wg.Add(1)
		go func(server string) {
			defer wg.Done()
			response, err := ntp.Query(server)
			if err != nil {
				log.Printf("Error querying NTP server %s: %v", server, err)
				return
			}
			if err := response.Validate(); err != nil {
				log.Printf("Validation error from NTP server %s: %v", server, err)
				return
			}

			c.mutex.Lock()
			defer c.mutex.Unlock()
			c.offset = response.ClockOffset
			c.lastSynced = time.Now()
			c.synced.Store(true)
			c.syncAttemptsDone.Store(true)
			log.Printf("Time synchronized with NTP server %s, offset: %v", server, c.offset)
		}(server)
	}

	wg.Wait()

	if !c.synced.Load() {
		log.Println("Failed to synchronize time with all NTP servers")
	}
}

func (c *NTPClient) Now() time.Time {
	if !c.synced.Load() && !c.syncAttemptsDone.Load() {
		go c.UpdateTime()
		return time.Now()
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return time.Now().Add(c.offset)
}
