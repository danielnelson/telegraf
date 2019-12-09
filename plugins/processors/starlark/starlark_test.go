package starlark

import (
	"testing"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/testutil"
)

func BenchmarkNoop(b *testing.B) {
	plugin := &Starlark{
		Source: `
def apply(metric):
	return metric
`,
	}
	err := plugin.Init()
	if err != nil {
		panic(err)
	}

	metrics := []telegraf.Metric{
		testutil.MustMetric(
			"cpu",
			map[string]string{},
			map[string]interface{}{
				"time_idle": 42.0,
			},
			time.Unix(0, 0),
		),
	}

	for n := 0; n < b.N; n++ {
		_ = plugin.Apply(metrics...)
	}
}
