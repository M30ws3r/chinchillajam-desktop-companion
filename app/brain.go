package main

import "time"

// Sprite identifies one character pose.
type Sprite int

const (
	SpIdle Sprite = iota
	SpLeft
	SpRight
	SpFastA
	SpFastB
	SpPhone
	SpClick
	SpCoffee
	SpPatted
	spCount
)

var spriteFiles = [spCount]string{"idle", "left", "right", "fast_a", "fast_b", "phone", "click_heart", "coffee", "patted"}

// Config holds every timing value (same meaning as assets/chinchilla/config.json).
type Config struct {
	HitMs            int
	FastSwapMs       int
	ClickMs          int
	CoffeeAfterMs    int
	FastWindowMs     int
	FastThreshold    int
	AlternateResetMs int
	PhoneEvery       int
	PhoneHoldMs      int
	PatHoldMs        int
	SquashMs         int
	SquashScale      float64
}

func DefaultConfig() Config {
	return Config{
		HitMs: 90, FastSwapMs: 70, ClickMs: 450, CoffeeAfterMs: 10000,
		FastWindowMs: 1000, FastThreshold: 6, AlternateResetMs: 1000,
		PhoneEvery: 10, PhoneHoldMs: 1200, PatHoldMs: 1500, SquashMs: 60, SquashScale: 0.97,
	}
}

// Action tells the UI which sprite to show and when to go back to idle.
type Action struct {
	Show        Sprite
	Squash      bool
	RevertAfter time.Duration
}

// Brain is the platform-independent state machine.
type Brain struct {
	cfg       Config
	presses   []time.Time
	lastPress time.Time
	nextLeft  bool
	fastIdx   int
	count     int
	holdUntil time.Time
}

func NewBrain(c Config) *Brain { return &Brain{cfg: c, nextLeft: true} }

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

// Holding reports whether a sticky pose (phone) is still on screen.
func (b *Brain) Holding(now time.Time) bool { return now.Before(b.holdUntil) }

// Key handles one key press. ok=false means "keep the current pose".
func (b *Brain) Key(now time.Time) (Action, bool) {
	if b.Holding(now) {
		return Action{}, false
	}
	if b.lastPress.IsZero() || now.Sub(b.lastPress) > ms(b.cfg.AlternateResetMs) {
		b.nextLeft = true // after a pause, always start with the left paw
	}
	b.lastPress = now

	cut := now.Add(-ms(b.cfg.FastWindowMs))
	kept := b.presses[:0]
	for _, t := range b.presses {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	b.presses = append(kept, now)

	b.count++
	if b.cfg.PhoneEvery > 0 && b.count%b.cfg.PhoneEvery == 0 {
		b.holdUntil = now.Add(ms(b.cfg.PhoneHoldMs))
		return Action{Show: SpPhone, Squash: true, RevertAfter: ms(b.cfg.PhoneHoldMs)}, true
	}

	if len(b.presses) >= b.cfg.FastThreshold {
		b.fastIdx ^= 1
		sp := SpFastA
		if b.fastIdx == 1 {
			sp = SpFastB
		}
		b.nextLeft = !b.nextLeft
		return Action{Show: sp, Squash: true, RevertAfter: ms(b.cfg.HitMs + b.cfg.FastSwapMs)}, true
	}

	sp := SpRight
	if b.nextLeft {
		sp = SpLeft
	}
	b.nextLeft = !b.nextLeft
	return Action{Show: sp, Squash: true, RevertAfter: ms(b.cfg.HitMs)}, true
}

// Click handles a mouse click anywhere on screen.
func (b *Brain) Click(now time.Time) (Action, bool) {
	if b.Holding(now) {
		return Action{}, false
	}
	return Action{Show: SpClick, Squash: true, RevertAfter: ms(b.cfg.ClickMs)}, true
}

// Pat handles a left click on the character itself (always wins, sticky for PatHoldMs).
func (b *Brain) Pat(now time.Time) Action {
	b.holdUntil = now.Add(ms(b.cfg.PatHoldMs))
	return Action{Show: SpPatted, Squash: true, RevertAfter: ms(b.cfg.PatHoldMs)}
}
