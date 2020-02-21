package starlark

import (
	"testing"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/testutil"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		input    telegraf.Metric
		expected []telegraf.Metric
	}{
		{
			name: "noop",
			source: `
def apply(metric):
	return metric
			`,
			input: testutil.MustMetric(
				"cpu",
				map[string]string{},
				map[string]interface{}{
					"time_idle": 42,
				},
				time.Unix(0, 0),
			),
			expected: []telegraf.Metric{
				testutil.MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{
						"time_idle": 42,
					},
					time.Unix(0, 0),
				),
			},
		},
		{
			name: "iterate tags",
			source: `
def apply(metric):
	for k, v in metric.tags.items():
		metric.fields[k] = v
	return metric
			`,
			input: testutil.MustMetric(
				"cpu",
				map[string]string{
					"host": "example.org",
					"cpu":  "cpu0",
				},
				map[string]interface{}{
					"time_idle": 42,
				},
				time.Unix(0, 0),
			),
			expected: []telegraf.Metric{
				testutil.MustMetric(
					"cpu",
					map[string]string{
						"host": "example.org",
						"cpu":  "cpu0",
					},
					map[string]interface{}{
						"time_idle": 42,
						"host":      "example.org",
						"cpu":       "cpu0",
					},
					time.Unix(0, 0),
				),
			},
		},
		{
			name: "set name",
			source: `
def apply(metric):
	metric.name = "cpu2"
	return metric
			`,
			input: testutil.MustMetric(
				"cpu",
				map[string]string{},
				map[string]interface{}{
					"time_idle": 42,
				},
				time.Unix(0, 0),
			),
			expected: []telegraf.Metric{
				testutil.MustMetric(
					"cpu2",
					map[string]string{},
					map[string]interface{}{
						"time_idle": 42,
					},
					time.Unix(0, 0),
				),
			},
		},
		{
			name: "set time",
			source: `
def apply(metric):
	metric.time -= metric.time % 100000000
	return metric
			`,
			input: testutil.MustMetric(
				"cpu",
				map[string]string{},
				map[string]interface{}{
					"time_idle": 42,
				},
				time.Unix(42, 42).UTC(),
			),
			expected: []telegraf.Metric{
				testutil.MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{
						"time_idle": 42,
					},
					time.Unix(42, 0).UTC(),
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := &Starlark{
				Source: tt.source,
				Log:    testutil.Logger{},
			}
			err := plugin.Init()
			require.NoError(t, err)

			actual := plugin.Apply(tt.input)
			testutil.RequireMetricsEqual(t, tt.expected, actual)
		})
	}
}

func BenchmarkNoop(b *testing.B) {
	plugin := &Starlark{
		Source: `
def apply(metric):
	return metric
`,
		Log: testutil.Logger{},
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

func TestRename(t *testing.T) {
	plugin := &Starlark{
		Source: `
def apply(metric):
	metric.name = "howdy"
	return metric
`,
		Log: testutil.Logger{},
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

	expected := []telegraf.Metric{
		testutil.MustMetric(
			"howdy",
			map[string]string{},
			map[string]interface{}{
				"time_idle": 42.0,
			},
			time.Unix(0, 0),
		),
	}

	actual := plugin.Apply(metrics...)

	testutil.RequireMetricsEqual(t, expected, actual)
}

func BenchmarkRename(b *testing.B) {
	plugin := &Starlark{
		Source: `
def apply(metric):
	metric.name = "howdy"
	return metric
`,
		Log: testutil.Logger{},
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
		plugin.Apply(metrics...)
	}
}

func TestSetTime(t *testing.T) {
	plugin := &Starlark{
		Source: `
def apply(metric):
	metric.time = 42
	return metric
`,
		Log: testutil.Logger{},
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

	expected := []telegraf.Metric{
		testutil.MustMetric(
			"cpu",
			map[string]string{},
			map[string]interface{}{
				"time_idle": 42.0,
			},
			time.Unix(0, 42),
		),
	}

	actual := plugin.Apply(metrics...)

	testutil.RequireMetricsEqual(t, expected, actual)
}

func TestGetTag(t *testing.T) {
	plugin := &Starlark{
		Source: `
def apply(metric):
	print(metric.tags['host'])
	return metric
`,
		Log: testutil.Logger{},
	}
	err := plugin.Init()
	if err != nil {
		panic(err)
	}

	metrics := []telegraf.Metric{
		testutil.MustMetric(
			"cpu",
			map[string]string{
				"host": "example.org",
			},
			map[string]interface{}{
				"time_idle": 0,
			},
			time.Unix(0, 0),
		),
	}

	_ = plugin.Apply(metrics...)
}

func TestTagMapping(t *testing.T) {
	plugin := &Starlark{
		Source: `
def apply(metric):
	for k in metric.tags:
		print(k)
	return metric
`,
		Log: testutil.Logger{},
	}
	err := plugin.Init()
	if err != nil {
		panic(err)
	}

	metrics := []telegraf.Metric{
		testutil.MustMetric(
			"cpu",
			map[string]string{
				"host": "example.org",
				"cpu":  "cpu0",
			},
			map[string]interface{}{
				"time_idle": 0,
			},
			time.Unix(0, 0),
		),
	}

	actual := plugin.Apply(metrics...)
	_ = actual
}

func TestTagMappingItems(t *testing.T) {
	plugin := &Starlark{
		Source: `
def apply(metric):
	print(dir(dict()))
	for k, v in {'x': 1}.items():
		print('items: ', k, v)
	print(type({'x': 1}.items))
	print(dir(metric.tags))
	for k in metric.tags.keys():
		print(k)
	for k,v in metric.tags.items():
		print('items:',k,v)
	return metric
`,
		Log: testutil.Logger{},
	}
	err := plugin.Init()
	if err != nil {
		panic(err)
	}

	metrics := []telegraf.Metric{
		testutil.MustMetric(
			"cpu",
			map[string]string{
				"host": "example.org",
				"cpu":  "cpu0",
			},
			map[string]interface{}{
				"time_idle": 0,
			},
			time.Unix(0, 0),
		),
	}

	actual := plugin.Apply(metrics...)
	_ = actual
}
