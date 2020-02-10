package starlark

import (
	"errors"
	"fmt"
	"time"

	"github.com/influxdata/telegraf"
	"go.starlark.net/starlark"
)

type Metric struct {
	metric telegraf.Metric
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

func (m *Metric) AttrNames() []string {
	return []string{"name", "tags", "fields", "time"}
}

// Attr implements the HasAttrs interface
func (m *Metric) Attr(name string) (starlark.Value, error) {
	switch name {
	case "name":
		return starlark.String(m.metric.Name()), nil
	case "tags":
		return m.TagsToDict(), nil
	case "fields":
		return m.FieldsToDict(), nil
	case "time":
		return starlark.MakeInt64(m.metric.Time().UnixNano()), nil
	default:
		// "no such field or method"
		return nil, nil
	}
}

func (m *Metric) TagsToDict() *starlark.Dict {
	dict := starlark.NewDict(len(m.metric.TagList()))
	for _, tag := range m.metric.TagList() {
		dict.SetKey(
			starlark.String(tag.Key),
			starlark.String(tag.Value),
		)
	}
	return dict
}

func (m *Metric) FieldsToDict() *starlark.Dict {
	dict := starlark.NewDict(len(m.metric.FieldList()))
	for _, field := range m.metric.FieldList() {
		var sk = starlark.String(field.Key)
		var sv starlark.Value
		switch fv := field.Value.(type) {
		case float64:
			sv = starlark.Float(fv)
		case int64:
			sv = starlark.MakeInt64(fv)
		case uint64:
			sv = starlark.MakeUint64(fv)
		case string:
			sv = starlark.String(fv)
		case bool:
			sv = starlark.Bool(fv)
		default:
			// todo error
		}

		dict.SetKey(sk, sv)
	}
	return dict
}

type TagSet struct {
	metric telegraf.Metric
}

func (m *TagSet) String() string {
	return "tags"
}

func (m *TagSet) Type() string {
	return "tags"
}

func (m *TagSet) Freeze() {
}

func (m *TagSet) Truth() starlark.Bool {
	return true
}

func (m *TagSet) Hash() (uint32, error) {
	return 0, errors.New("not hashable")
}

func (m *TagSet) Get(key starlark.Value) (v starlark.Value, found bool, err error) {
	switch key := key.(type) {
	case starlark.String:
		t, ok := m.metric.GetTag(key.GoString())
		return starlark.String(t), ok, nil
	default:
		return starlark.String(""), false, errors.New("type error")
	}
}

func (m *TagSet) Keys() []starlark.Value {
	items := make([]starlark.Value, 0, len(m.metric.TagList()))
	for _, tags := range m.metric.TagList() {
		item := starlark.String(tags.Key)
		items = append(items, item)
	}
	return items
}

func (m *TagSet) Items() []starlark.Tuple {
	items := make([]starlark.Tuple, 0, len(m.metric.TagList()))
	for _, tags := range m.metric.TagList() {
		pair := starlark.Tuple{
			starlark.String(tags.Key),
			starlark.String(tags.Value),
		}
		items = append(items, pair)
	}
	return items
}

func (m *TagSet) AttrNames() []string {
	return []string{"items", "keys"}
}

func dict_keys(b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 0); err != nil {
		return nil, err
	}
	return starlark.NewList(b.Receiver().(*TagSet).Keys()), nil
}

func (m *TagSet) Attr(name string) (starlark.Value, error) {
	switch name {
	case "items":
	case "keys":
		impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			return dict_keys(b, args, kwargs)
		}
		return starlark.NewBuiltin("keys", impl).BindReceiver(m), nil
	default:
		// "no such field or method"
		return nil, nil
	}
	return nil, nil
}

var _ starlark.IterableMapping = (*TagSet)(nil)

type TagIterator struct {
	metric telegraf.Metric
	index  int
}

var _ starlark.Iterator = (*TagIterator)(nil)

func (i *TagIterator) Next(p *starlark.Value) bool {
	if i.index >= len(i.metric.TagList()) {
		return false
	}
	s := starlark.String(i.metric.TagList()[i.index].Key)
	*p = s
	i.index++
	return true
}

func (i *TagIterator) Done() {
}

func (m *TagSet) Iterate() starlark.Iterator {
	return &TagIterator{metric: m.metric}
}

func (m *Metric) Tags() starlark.Value {
	return &TagSet{metric: m.metric}
}

func (m *Metric) Fields() starlark.Value {
	list := &starlark.List{}
	for _, fields := range m.metric.FieldList() {
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

func (m *Metric) SetField(name string, value starlark.Value) error {
	switch name {
	case "name":
		m.SetName(name, value)
		return nil
	case "time":
		m.SetTime(name, value)
		return nil
	default:
		return starlark.NoSuchAttrError(
			fmt.Sprintf("cannot assign to field '%s'", name))
	}
}

func (m *Metric) SetName(name string, value starlark.Value) error {
	switch v := value.(type) {
	case starlark.String:
		m.metric.SetName(v.GoString())
		return nil
	default:
		return errors.New("type error")
	}
}

func (m *Metric) SetTime(name string, value starlark.Value) error {
	switch v := value.(type) {
	case starlark.Int:
		ns, ok := v.Int64()
		if !ok {
			return errors.New("type error: unrepresentable time")
		}
		tm := time.Unix(0, ns)
		m.metric.SetTime(tm)
		return nil
	default:
		return errors.New("type error")
	}
}
