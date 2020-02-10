package starlark

import (
	"errors"
	"fmt"
	"strings"

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
	results   []telegraf.Metric
}

func (s *Starlark) Init() error {
	s.thread = &starlark.Thread{
		Name:  "processor.starlark",
		Print: func(_ *starlark.Thread, msg string) { s.Log.Debug(msg) },
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
	met := &Metric{}
	s.args[0] = met

	s.results = make([]telegraf.Metric, 0, 10)

	return nil
}

func (s *Starlark) SampleConfig() string {
	return sampleConfig
}

func (s *Starlark) Description() string {
	return description
}

func (s *Starlark) Apply(metrics ...telegraf.Metric) []telegraf.Metric {
	s.results = s.results[:]
	for _, m := range metrics {
		s.args[0].(*Metric).metric = m
		rv, err := starlark.Call(s.thread, s.applyFunc, s.args, nil)
		if err != nil {
			// FIXME What do? toss metric/keep metric
			// s.Log.Errorf("Error calling apply function: %v", err)
			err := err.(*starlark.EvalError)
			for _, line := range strings.Split(err.Backtrace(), "\n") {
				s.Log.Error(line)
			}
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
					s.results = append(s.results, v.metric)
				default:
					fmt.Printf("error, rv: %T\n", v)
				}
			}
		case *Metric:
			s.results = append(s.results, rv.metric)
		default:
			fmt.Printf("error, rv: %T\n", rv)
		}
	}

	return s.results
}

func init() {
	processors.Add("starlark", func() telegraf.Processor {
		return &Starlark{}
	})
}
