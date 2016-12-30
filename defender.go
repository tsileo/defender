package defender

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const Factor = 50

type Client struct {
	limiter *rate.Limiter
	expire  time.Time
	banned  bool
	key     interface{}
}

func (c *Client) Key() interface{}  { return c.key }
func (c *Client) Banned() bool      { return c.banned }
func (c *Client) Expire() time.Time { return c.expire }

// Defender keep tracks if the `Client`s and maintains the banlist
type Defender struct {
	tokenBuckets map[interface{}]*Client

	Duration    time.Duration
	BanDuration time.Duration
	Max         int

	BanHook func(interface{}) error

	sync.Mutex
}

// New initializes a Defender instance that will limit `max` event maximum per `duration` before banning the client for `banDuration`
func New(max int, duration, banDuration time.Duration) *Defender {
	return &Defender{
		tokenBuckets: map[interface{}]*Client{},
		Duration:     duration,
		BanDuration:  banDuration,
		Max:          max,
	}
}

// BanList returns the list of banned clients
func (d *Defender) BanList() []*Client {
	l := []*Client{}
	for _, client := range d.tokenBuckets {
		if client.banned {
			l = append(l, client)
		}
	}
	return l
}

// Banned returns true if the client is banned
func (d *Defender) Banned(key interface{}) (bool, error) {
	d.Lock()
	defer d.Unlock()
	now := time.Now()

	// Try to retrieve the
	if _, found := d.tokenBuckets[key]; !found {
		d.tokenBuckets[key] = &Client{
			key:     key,
			limiter: rate.NewLimiter(rate.Every(d.Duration), d.Max),
			expire:  now.Add(d.Duration * Factor),
		}
	}

	client := d.tokenBuckets[key]

	// Check if the client is not banned anymore and the cleanup hasn't been run yet
	if client.banned && now.After(client.expire) {
		client.banned = false
	}

	// Check if client is banned
	if client.banned {
		return true, nil
	}

	// Update the client expiration
	client.expire = now.Add(d.Duration * Factor)

	// Check the rate limiter
	banned := !client.limiter.AllowN(time.Now(), 1)

	if banned {
		// Set the client as banned
		client.banned = true

		// Check if a hook is registered
		if d.BanHook != nil {
			if err := d.BanHook(key); err != nil {
				return false, err
			}
		}
		// Set the ban duration
		client.expire = now.Add(d.BanDuration)
	}

	return banned, nil
}

// Cleanup should be used if you want to manage the cleanup yourself, looks for CleanupTask for an automatic way
func (d *Defender) Cleanup() {
	d.Lock()
	defer d.Unlock()
	now := time.Now()
	for key, client := range d.tokenBuckets {
		if now.After(client.expire) {
			delete(d.tokenBuckets, key)
		}
	}
}

// CleanupTask should be run in a goroutime
func (d *Defender) CleanupTask(quit <-chan struct{}) {
	c := time.Tick(d.Duration * Factor)
	for {
		select {
		case <-quit:
			break
		case <-c:
			d.Cleanup()
		}
	}
}
