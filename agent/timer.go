package agent

import (
	"time"

	"github.com/influxdata/telegraf/internal"
)

// Timer represents a single future event.
type Timer interface {
	// Elapsed returns a channel that can be read from when the event as
	// elapsed.  The value is the scheduled time.
	Elapsed() <-chan time.Time

	// Reset reschedules the event again.  The previous elapsed time should be
	// passed in to ensure proper rescheduling.
	//
	// If the event has not been read you must Stop the Timer and read the
	// event, similar to time.Timer.
	Reset(prev time.Time)

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
	d := internal.AlignDuration(start, interval) +
		internal.RandomDuration(jitter)

	return &AlignedTimer{
		interval: interval,
		jitter:   jitter,
		timer:    time.NewTimer(d),
	}
}

// could we skip intervals?  it should be impossible to do this without the
// warning.  the next interval should be based on the previous interval, not
// now().
//
// what about negative durations?  what about zero durations?
//
// prev thing was a bad idea, what if write took longer than an interval, it would expire immediately.  we do want to skip intervals

func (t *AlignedTimer) Elapsed() <-chan time.Time {
	return t.timer.C
}

func (t *AlignedTimer) Reset(prev time.Time) {
	t.timer.Reset(t.nextDuration(prev))
}

func (t *AlignedTimer) Stop() bool {
	return t.timer.Stop()
}

func (t *AlignedTimer) Interval() time.Duration {
	return t.interval
}

func (t *AlignedTimer) nextDuration(prev time.Time) time.Duration {
	return internal.AlignDuration(prev, t.interval) +
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

func (t *UnalignedTimer) Reset(time.Time) {
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
