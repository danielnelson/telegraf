package agent

import (
	"fmt"
	"testing"
	"time"
)

func TestA(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	start := time.Now()
	interval := 50 * time.Millisecond
	jitter := 0 * time.Second

	ticker := NewAlignTicker(start, interval, jitter)
	for {
		select {
		case tm := <-ticker.Elapsed():
			fmt.Printf("tick\t\t\t\t\t\t\t\t\t\t\t\t%s\n", tm)
		}
	}
}
