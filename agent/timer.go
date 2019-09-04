package agent

import (
	"time"

	"github.com/influxdata/telegraf/internal"
)

// Timer represents a single future event, the Timer can be reset to reschedule
// the event.
//
// This timer encapsulates the jitter handling and compensates for timer drift.
type Timer interface {
	// Elapsed returns a channel that can be read from when the event as
	// elapsed.  The value is the scheduled time.
	Elapsed() <-chan time.Time

	// Reset reschedules the event again based on the current time and the
	// initially set duration.
	//
	// If the event has not been read you must Stop the Timer and read the
	// event, similar to time.Timer.
	Reset()

	// Stop the timer; returns true if the timer was stopped before sending the
	// event.  If false, you must read any values in the elapsed channel.
	Stop() bool

	// Returns the Timer interval.
	Interval() time.Duration
}

// AlignedTimer is a Timer that is aligned to the interval.
type AlignedTimer struct {
	interval time.Duration
	jitter   time.Duration
	timer    *time.Timer
}

func NewAlignedTimer(start time.Time, interval, jitter time.Duration) *AlignedTimer {
	t := &AlignedTimer{
		interval: interval,
		jitter:   jitter,
	}
	d := t.nextDuration(start)
	t.timer = time.NewTimer(d)
	return t
}

// what about negative durations?  what about zero durations?
//
// could we skip intervals?  it should be impossible to do this without the
// warning.  the next interval should be based on the previous interval, not
// now().
//
// prev thing was a bad idea, what if write took longer than an interval, it
// would expire immediately.  we do want to skip intervals
//
// would be nice to guarantee log or run interval; because prev was reverted right now its racy

func (t *AlignedTimer) Elapsed() <-chan time.Time {
	return t.timer.C
}

func (t *AlignedTimer) Reset() {
	t.timer.Reset(t.nextDuration(time.Now()))
}

func (t *AlignedTimer) Stop() bool {
	return t.timer.Stop()
}

func (t *AlignedTimer) Interval() time.Duration {
	return t.interval
}

func (t *AlignedTimer) nextDuration(now time.Time) time.Duration {
	return internal.AlignDuration(now, t.interval) +
		internal.RandomDuration(t.jitter)
}

func NewUnalignedTimer(interval, jitter time.Duration) *UnalignedTimer {
	t := &UnalignedTimer{
		interval: interval,
		jitter:   jitter,
	}

	t.timer = time.NewTimer(t.nextDuration())
	return t
}

// UnalignedTimer is a Timer that is not aligned to the interval.
type UnalignedTimer struct {
	interval time.Duration
	jitter   time.Duration
	timer    *time.Timer
}

func (t *UnalignedTimer) Elapsed() <-chan time.Time {
	return t.timer.C
}

func (t *UnalignedTimer) Reset() {
	t.timer.Reset(t.nextDuration())
}

func (t *UnalignedTimer) Stop() bool {
	return t.timer.Stop()
}

func (t *UnalignedTimer) Interval() time.Duration {
	return t.interval
}

func (t *UnalignedTimer) nextDuration() time.Duration {
	return t.interval + internal.RandomDuration(t.jitter)
}
