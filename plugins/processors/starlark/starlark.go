package starlark

import (
	"errors"
	"fmt"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/processors"
	"go.starlark.net/starlark"
)

const (
	description  = ""
	sampleConfig = `
`
)

type Starlark struct {
	Source string          `toml:"source"`
	Log    telegraf.Logger `toml:"-"`

	thread    *starlark.Thread
	applyFunc *starlark.Function
	args      starlark.Tuple
}

func (s *Starlark) Init() error {
	s.thread = &starlark.Thread{
		Name:  "processor.starlark",
		Print: func(_ *starlark.Thread, msg string) { s.Log.Info(msg) },
	}

	predeclared := starlark.StringDict{}

	_, program, err := starlark.SourceProgram("processor.starlark", s.Source, predeclared.Has)
	if err != nil {
		return nil
	}

	// Execute source
	globals, err := program.Init(s.thread, predeclared)
	if err != nil {
		return nil
	}

	// The source should define an apply function.
	apply := globals["apply"]

	if apply == nil {
		return errors.New("apply not found")
	}

	var ok bool
	if s.applyFunc, ok = apply.(*starlark.Function); !ok {
		return errors.New("apply is not a function")
	}

	if s.applyFunc.NumParams() != 1 {
		return errors.New("apply function must take one parameter")
	}

	s.args = make(starlark.Tuple, 1)
	return nil
}

func (s *Starlark) SampleConfig() string {
	return sampleConfig
}

func (s *Starlark) Description() string {
	return description
}

func (s *Starlark) Apply(metrics ...telegraf.Metric) []telegraf.Metric {
	var results []telegraf.Metric
	for _, m := range metrics {
		s.args[0] = &Metric{m}
		rv, err := starlark.Call(s.thread, s.applyFunc, s.args, nil)
		if err != nil {
			// FIXME What do? toss metric/keep metric
			s.Log.Errorf("Error calling apply function: %v", err)
			continue
		}

		switch rv := rv.(type) {
		case *starlark.List:
			iter := rv.Iterate()
			defer iter.Done()
			var v starlark.Value
			for iter.Next(&v) {
				switch v := v.(type) {
				case *Metric:
					results = append(results, v.Metric)
				default:
					fmt.Printf("error, rv: %T\n", v)
				}
			}
		case *Metric:
			results = append(results, rv.Metric)
		default:
			fmt.Printf("error, rv: %T\n", rv)
		}
	}

	return results
}

func init() {
	processors.Add("starlark", func() telegraf.Processor {
		return &Starlark{}
	})
}
