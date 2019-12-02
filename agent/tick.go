package agent

import (
	"context"
	"sync"
	"time"

	"github.com/influxdata/telegraf/internal"
)

// AlignTicker is a ticker whose interval is aligned as specified by the
// round_interval option.  This ticker uses absolute time to avoid drift when
// used for long periods of time.
type AlignTicker struct {
	interval time.Duration
	jitter   time.Duration
	ch       chan time.Time

	timer      *time.Timer
	wg         sync.WaitGroup
	cancelFunc context.CancelFunc
}

func NewAlignTicker(start time.Time, interval, jitter time.Duration) *AlignTicker {
	t := &AlignTicker{
		interval: interval,
		jitter:   jitter,
		ch:       make(chan time.Time),
	}

	d := t.nextDuration(start)
	t.timer = time.NewTimer(d)

	ctx, cancel := context.WithCancel(context.Background())
	t.cancelFunc = cancel

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.relayTime(ctx)
	}()

	return t
}

func (t *AlignTicker) nextDuration(now time.Time) time.Duration {
	next := internal.AlignTime(now, t.interval)
	d := next.Sub(now)
	d += internal.RandomDuration(t.jitter)
	return d
}

func (t *AlignTicker) relayTime(ctx context.Context) {
	for {
		select {
		case now := <-t.timer.C:
			select {
			case t.ch <- now:
			default:
			}

			d := t.nextDuration(now)
			t.timer.Reset(d)
		case <-ctx.Done():
			t.timer.Stop()
			return
		}
	}
}

func (t *AlignTicker) Elapsed() <-chan time.Time {
	return t.ch
}

func (t *AlignTicker) Stop() {
	t.cancelFunc()
	t.wg.Wait()
}

type UnalignTicker struct {
	interval time.Duration
	jitter   time.Duration
}
