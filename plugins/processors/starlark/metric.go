package starlark

import (
	"errors"

	"github.com/influxdata/telegraf"
	"go.starlark.net/starlark"
)

type Metric struct {
	telegraf.Metric
}

func (m *Metric) String() string {
	return "metric"
}

func (m *Metric) Type() string {
	return "metric"
}

func (m *Metric) Freeze() {
}

func (m *Metric) Truth() starlark.Bool {
	return true
}

func (m *Metric) Hash() (uint32, error) {
	return 0, errors.New("not hashable")
}

func (m *Metric) Attr(name string) (starlark.Value, error) {
	switch name {
	case "name":
		return starlark.String(m.Name()), nil
	case "tags":
		return m.Tags(), nil
	case "fields":
		return m.Fields(), nil
	case "time":
		return starlark.MakeInt64(m.Time().UnixNano()), nil
	default:
		return nil, nil
	}
}

func (m *Metric) AttrNames() []string {
	return []string{"name", "tags", "fields", "time"}
}

func (m *Metric) Tags() starlark.Value {
	list := &starlark.List{}
	for _, tags := range m.TagList() {
		t := starlark.Tuple{
			starlark.String(tags.Key),
			starlark.String(tags.Value),
		}
		list.Append(t)
	}
	return list
}

func (m *Metric) Fields() starlark.Value {
	list := &starlark.List{}
	for _, fields := range m.FieldList() {
		switch fv := fields.Value.(type) {
		case int64:
			f := starlark.Tuple{
				starlark.String(fields.Key),
				starlark.MakeInt64(fv),
			}
			list.Append(f)
		case float64:
			f := starlark.Tuple{
				starlark.String(fields.Key),
				starlark.Float(fv),
			}
			list.Append(f)
		}
	}
	return list
}
