package defender

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const Factor = 50

type client struct {
	limiter *rate.Limiter
	expire  time.Time
	banned  bool
}

type Defender struct {
	tokenBuckets map[string]*client

	Duration    time.Duration
	BanDuration time.Duration
	Max         int

	sync.Mutex
}

func New(max int, duration, banDuration time.Duration) *Defender {
	return &Defender{
		tokenBuckets: map[string]*client{},
		Duration:     duration,
		BanDuration:  banDuration,
		Max:          max,
	}
}

func (d *Defender) Banned(key string) bool {
	d.Lock()
	defer d.Unlock()
	now := time.Now()
	if _, found := d.tokenBuckets[key]; !found {
		d.tokenBuckets[key] = &client{
			limiter: rate.NewLimiter(rate.Every(d.Duration), d.Max),
			expire:  now.Add(d.Duration * Factor),
		}
	}

	client := d.tokenBuckets[key]
	if client.banned {
		return true
	}
	client.expire = now.Add(d.Duration * Factor)
	banned := !client.limiter.AllowN(now, 1)
	if banned {
		client.banned = true
		client.expire = now.Add(d.BanDuration)
	}
	return banned
}

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
