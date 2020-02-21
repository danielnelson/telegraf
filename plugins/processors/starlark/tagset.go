package starlark

import (
	"errors"
	"fmt"

	"github.com/influxdata/telegraf"
	"go.starlark.net/starlark"
)

type TagSet struct {
	metric telegraf.Metric
}

func (m *TagSet) String() string {
	return "tags"
}

func (m *TagSet) Type() string {
	return "TagSet"
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

// type IterableMapping
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

func (m *TagSet) Attr(name string) (starlark.Value, error) {
	switch name {
	case "items":
		impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			return dict_items(b, args, kwargs)
		}
		return starlark.NewBuiltin("items", impl).BindReceiver(m), nil
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

func (m *TagSet) Iterate() starlark.Iterator {
	return &TagIterator{metric: m.metric}
}

func (m *TagSet) Delete(k starlark.Value) (v starlark.Value, found bool, err error) {
	panic("not implemented")
}

func (m *TagSet) Clear() error {
	panic("not implemented")
}

func (m *TagSet) SetKey(k, v starlark.Value) error {
	panic("not implemented")
}

func (d *Dict) Len() int {
	panic("not implemented")
}

var _ starlark.IterableMapping = (*TagSet)(nil)

// ---- methods of built-in types ---

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·get
func dict_get(b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key, dflt starlark.Value
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &key, &dflt); err != nil {
		return nil, err
	}
	if v, ok, err := b.Receiver().(*TagSet).Get(key); err != nil {
		return nil, nameErr(b, err)
	} else if ok {
		return v, nil
	} else if dflt != nil {
		return dflt, nil
	}
	return starlark.None, nil
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·clear
func dict_clear(b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 0); err != nil {
		return nil, err
	}
	return starlark.None, b.Receiver().(*TagSet).Clear()
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·items
func dict_items(b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 0); err != nil {
		return nil, err
	}
	items := b.Receiver().(*TagSet).Items()
	res := make([]starlark.Value, len(items))
	for i, item := range items {
		res[i] = item // convert [2]starlark.Value to starlark.Value
	}
	return starlark.NewList(res), nil
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·keys
func dict_keys(b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 0); err != nil {
		return nil, err
	}
	return starlark.NewList(b.Receiver().(*TagSet).Keys()), nil
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·pop
func dict_pop(b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var k, d starlark.Value
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &k, &d); err != nil {
		return nil, err
	}
	if v, found, err := b.Receiver().(*TagSet).Delete(k); err != nil {
		return nil, nameErr(b, err) // dict is frozen or key is unhashable
	} else if found {
		return v, nil
	} else if d != nil {
		return d, nil
	}
	return nil, nameErr(b, "missing key")
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·popitem
func dict_popitem(b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 0); err != nil {
		return nil, err
	}
	recv := b.Receiver().(*TagSet)
	k, ok := recv.ht.first()
	if !ok {
		return nil, nameErr(b, "empty dict")
	}
	v, _, err := recv.Delete(k)
	if err != nil {
		return nil, nameErr(b, err) // dict is frozen
	}
	return starlark.Tuple{k, v}, nil
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·setdefault
func dict_setdefault(b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key, dflt starlark.Value = nil, starlark.None
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &key, &dflt); err != nil {
		return nil, err
	}
	tags := b.Receiver().(*TagSet)
	if v, ok, err := tags.Get(key); err != nil {
		return nil, nameErr(b, err)
	} else if ok {
		return v, nil
	} else if err := tags.SetKey(key, dflt); err != nil {
		return nil, nameErr(b, err)
	} else {
		return dflt, nil
	}
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·update
func dict_update(b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) > 1 {
		return nil, fmt.Errorf("update: got %d arguments, want at most 1", len(args))
	}
	if err := updateDict(b.Receiver().(*TagSet), args, kwargs); err != nil {
		return nil, fmt.Errorf("update: %v", err)
	}
	return starlark.None, nil
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·update
func dict_values(b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 0); err != nil {
		return nil, err
	}
	items := b.Receiver().(*TagSet).Items()
	res := make([]starlark.Value, len(items))
	for i, item := range items {
		res[i] = item[1]
	}
	return starlark.NewList(res), nil
}

// Common implementation of builtin dict function and dict.update method.
// Precondition: len(updates) == 0 or 1.
func updateDict(dict *TagSet, updates starlark.Tuple, kwargs []starlark.Tuple) error {
	if len(updates) == 1 {
		switch updates := updates[0].(type) {
		case starlark.IterableMapping:
			// Iterate over dict's key/value pairs, not just keys.
			for _, item := range updates.Items() {
				if err := dict.SetKey(item[0], item[1]); err != nil {
					return err // dict is frozen
				}
			}
		default:
			// all other sequences
			iter := starlark.Iterate(updates)
			if iter == nil {
				return fmt.Errorf("got %s, want iterable", updates.Type())
			}
			defer iter.Done()
			var pair starlark.Value
			for i := 0; iter.Next(&pair); i++ {
				iter2 := starlark.Iterate(pair)
				if iter2 == nil {
					return fmt.Errorf("dictionary update sequence element #%d is not iterable (%s)", i, pair.Type())

				}
				defer iter2.Done()
				len := starlark.Len(pair)
				if len < 0 {
					return fmt.Errorf("dictionary update sequence element #%d has unknown length (%s)", i, pair.Type())
				} else if len != 2 {
					return fmt.Errorf("dictionary update sequence element #%d has length %d, want 2", i, len)
				}
				var k, v starlark.Value
				iter2.Next(&k)
				iter2.Next(&v)
				if err := dict.SetKey(k, v); err != nil {
					return err
				}
			}
		}
	}

	// Then add the kwargs.
	before := dict.Len()
	for _, pair := range kwargs {
		if err := dict.SetKey(pair[0], pair[1]); err != nil {
			return err // dict is frozen
		}
	}
	// In the common case, each kwarg will add another dict entry.
	// If that's not so, check whether it is because there was a duplicate kwarg.
	if dict.Len() < before+len(kwargs) {
		keys := make(map[starlark.String]bool, len(kwargs))
		for _, kv := range kwargs {
			k := kv[0].(starlark.String)
			if keys[k] {
				return fmt.Errorf("duplicate keyword arg: %v", k)
			}
			keys[k] = true
		}
	}

	return nil
}

// nameErr returns an error message of the form "name: msg"
// where name is b.Name() and msg is a string or error.
func nameErr(b *starlark.Builtin, msg interface{}) error {
	return fmt.Errorf("%s: %v", b.Name(), msg)
}
