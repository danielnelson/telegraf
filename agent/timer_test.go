package agent

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal/models"
	"github.com/influxdata/telegraf/testutil"
)

func NewTestingTimer() *TestingTimer {
	return &TestingTimer{
		C: make(chan time.Time, 1),
	}
}

type TestingTimer struct {
	C       chan time.Time
	expired bool
}

func (t *TestingTimer) Elapsed() <-chan time.Time {
	return t.C
}

func (t *TestingTimer) Signal() {
	t.expired = true
	t.C <- time.Now()
}

func (t *TestingTimer) Reset() {
	select {
	case <-t.C:
		panic("timer not stopped and channel unread")
	default:
	}
}

func (t *TestingTimer) Stop() bool {
	return !t.expired
}

func (t *TestingTimer) Interval() time.Duration {
	return 10 * time.Second
}

type FakeOutput struct {
	ConnectF      func() error
	CloseF        func() error
	DescriptionF  func() string
	SampleConfigF func() string
	WriteF        func(metrics []telegraf.Metric) error
}

func (o *FakeOutput) Connect() error {
	return o.ConnectF()
}

func (o *FakeOutput) Close() error {
	return o.CloseF()
}

func (o *FakeOutput) Description() string {
	return o.DescriptionF()
}

func (o *FakeOutput) SampleConfig() string {
	return o.SampleConfigF()
}

func (o *FakeOutput) Write(metrics []telegraf.Metric) error {
	return o.WriteF(metrics)
}

var testMetric = testutil.MustMetric(
	"cpu",
	map[string]string{},
	map[string]interface{}{
		"time_idle": 42,
	},
	time.Unix(0, 0),
)

func TestFlush(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	output := &FakeOutput{}
	ro := models.NewRunningOutput(
		"fake", output, &models.OutputConfig{}, 10, 10)

	timer := NewTestingTimer()

	cancel()
	flush(ctx, ro, timer)
}

func TestFlush2(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	output := &FakeOutput{
		WriteF: func(metrics []telegraf.Metric) error {
			fmt.Println(10)
			cancel()
			return nil
		},
	}
	ro := models.NewRunningOutput(
		"fake", output, &models.OutputConfig{}, 10, 10)
	ro.AddMetric(testMetric)

	timer := NewTestingTimer()
	timer.Signal()
	flush(ctx, ro, timer)
}

func TestFlush3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	output := &FakeOutput{
		WriteF: func(metrics []telegraf.Metric) error {
			fmt.Println(10)
			cancel()
			return nil
		},
	}
	ro := models.NewRunningOutput(
		"fake", output, &models.OutputConfig{}, 10, 10)
	ro.AddMetric(testMetric)

	timer := NewTestingTimer()
	ro.BatchReady <- time.Now()
	flush(ctx, ro, timer)
}

func TestFlush4(t *testing.T) {
	timer := NewTestingTimer()
	call := 0
	ctx, cancel := context.WithCancel(context.Background())
	output := &FakeOutput{
		WriteF: func(metrics []telegraf.Metric) error {
			fmt.Println("write", call)
			if call == 0 {
				call++
				// this gets wiped out by the timer.reset
				timer.Signal()
				fmt.Println("error")
				return errors.New("error")
			}

			if call == 1 {
				call++
				cancel()
				return nil
			}

			return nil
		},
	}
	ro := models.NewRunningOutput(
		"fake", output, &models.OutputConfig{}, 10, 10)
	ro.AddMetric(testMetric)

	ro.BatchReady <- time.Now()
	flush(ctx, ro, timer)
}
