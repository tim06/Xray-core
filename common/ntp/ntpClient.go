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
    ntpServers       []string
    offset           time.Duration
    mutex            sync.Mutex
    lastSynced       time.Time
    synced           atomic.Bool
    syncAttemptsDone atomic.Bool
}

func NewNTPClient(ntpServers ...string) *NTPClient {
    client := &NTPClient{
        ntpServers: ntpServers,
    }
    go client.UpdateTime()
    return client
}

func (c *NTPClient) UpdateTime() error {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    if c.syncAttemptsDone.Load() {
        return nil
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
            err = response.Validate()
            if err != nil {
                log.Printf("Validation error from NTP server %s: %v", server, err)
                return
            }
            c.offset = response.ClockOffset
            c.lastSynced = time.Now()
            c.synced.Store(true)
            c.syncAttemptsDone.Store(true)
            log.Printf("Time synchronized with NTP server %s", server)
        }(server)
    }
    wg.Wait()

    if !c.synced.Load() {
        return fmt.Errorf("failed to synchronize time with all provided NTP servers")
    }
    return nil
}

func (c *NTPClient) Now() time.Time {
    if !c.synced.Load() && !c.syncAttemptsDone.Load() {
        go c.UpdateTime()
        if !c.synced.Load() {
            return time.Now()
        }
    }

    return time.Now().Add(c.offset)
}