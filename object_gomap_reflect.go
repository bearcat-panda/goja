package goja

import "reflect"

type objectGoMapReflect struct {
	objectGoReflect

	keyType, valueType reflect.Type
}

func (o *objectGoMapReflect) init() {
	o.objectGoReflect.init()
	o.keyType = o.value.Type().Key()
	o.valueType = o.value.Type().Elem()
}

func (o *objectGoMapReflect) toKey(n Value, throw bool) reflect.Value {
	if _, ok := n.(*valueSymbol); ok {
		o.val.runtime.typeErrorResult(throw, "Cannot set Symbol properties on Go maps")
		return reflect.Value{}
	}
	key, err := o.val.runtime.toReflectValue(n, o.keyType)
	if err != nil {
		o.val.runtime.typeErrorResult(throw, "map key conversion error: %v", err)
		return reflect.Value{}
	}
	return key
}

func (o *objectGoMapReflect) strToKey(name string, throw bool) reflect.Value {
	if o.keyType.Kind() == reflect.String {
		return reflect.ValueOf(name).Convert(o.keyType)
	}
	return o.toKey(newStringValue(name), throw)
}

func (o *objectGoMapReflect) _get(n Value) Value {
	key := o.toKey(n, false)
	if !key.IsValid() {
		return nil
	}
	if v := o.value.MapIndex(key); v.IsValid() {
		return o.val.runtime.ToValue(v.Interface())
	}

	return nil
}

func (o *objectGoMapReflect) _getStr(name string) Value {
	key := o.strToKey(name, false)
	if !key.IsValid() {
		return nil
	}
	if v := o.value.MapIndex(key); v.IsValid() {
		return o.val.runtime.ToValue(v.Interface())
	}

	return nil
}

func (o *objectGoMapReflect) get(n Value, receiver Value) Value {
	if v := o._get(n); v != nil {
		return v
	}
	return o.objectGoReflect.get(n, receiver)
}

func (o *objectGoMapReflect) getStr(name string, receiver Value) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.objectGoReflect.getStr(name, receiver)
}

func (o *objectGoMapReflect) getProp(n Value) Value {
	if v := o._get(n); v != nil {
		return v
	}
	return o.objectGoReflect.getProp(n)
}

func (o *objectGoMapReflect) getPropStr(name string) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.objectGoReflect.getPropStr(name)
}

func (o *objectGoMapReflect) getOwnPropStr(name string) Value {
	if v := o._getStr(name); v != nil {
		return &valueProperty{
			value:      v,
			writable:   true,
			enumerable: true,
		}
	}
	return o.objectGoReflect.getOwnPropStr(name)
}

func (o *objectGoMapReflect) getOwnProp(name Value) Value {
	if v := o._get(name); v != nil {
		return &valueProperty{
			value:      v,
			writable:   true,
			enumerable: true,
		}
	}
	return o.objectGoReflect.getOwnProp(name)
}

func (o *objectGoMapReflect) toValue(val Value, throw bool) (reflect.Value, bool) {
	v, err := o.val.runtime.toReflectValue(val, o.valueType)
	if err != nil {
		o.val.runtime.typeErrorResult(throw, "map value conversion error: %v", err)
		return reflect.Value{}, false
	}

	return v, true
}

func (o *objectGoMapReflect) put(key, val Value, throw bool) {
	k := o.toKey(key, throw)
	v, ok := o.toValue(val, throw)
	if !ok {
		return
	}
	o.value.SetMapIndex(k, v)
}

func (o *objectGoMapReflect) putStr(name string, val Value, throw bool) {
	k := o.strToKey(name, throw)
	if !k.IsValid() {
		return
	}
	v, ok := o.toValue(val, throw)
	if !ok {
		return
	}
	o.value.SetMapIndex(k, v)
}

func (o *objectGoMapReflect) _putProp(name string, value Value, writable, enumerable, configurable bool) Value {
	o.putStr(name, value, true)
	return value
}

func (o *objectGoMapReflect) defineOwnProperty(n Value, descr PropertyDescriptor, throw bool) bool {
	if !o.val.runtime.checkHostObjectPropertyDescr(n, descr, throw) {
		return false
	}

	o.put(n, descr.Value, throw)
	return true
}

func (o *objectGoMapReflect) hasOwnPropertyStr(name string) bool {
	key := o.strToKey(name, false)
	if !key.IsValid() {
		return false
	}
	return o.value.MapIndex(key).IsValid()
}

func (o *objectGoMapReflect) hasOwnProperty(n Value) bool {
	key := o.toKey(n, false)
	if !key.IsValid() {
		return false
	}

	return o.value.MapIndex(key).IsValid()
}

func (o *objectGoMapReflect) delete(n Value, throw bool) bool {
	key := o.toKey(n, throw)
	if !key.IsValid() {
		return false
	}
	o.value.SetMapIndex(key, reflect.Value{})
	return true
}

func (o *objectGoMapReflect) deleteStr(name string, throw bool) bool {
	key := o.strToKey(name, throw)
	if !key.IsValid() {
		return false
	}
	o.value.SetMapIndex(key, reflect.Value{})
	return true
}

type gomapReflectPropIter struct {
	o         *objectGoMapReflect
	keys      []reflect.Value
	idx       int
	recursive bool
}

func (i *gomapReflectPropIter) next() (propIterItem, iterNextFunc) {
	for i.idx < len(i.keys) {
		key := i.keys[i.idx]
		v := i.o.value.MapIndex(key)
		i.idx++
		if v.IsValid() {
			return propIterItem{name: key.String(), enumerable: _ENUM_TRUE}, i.next
		}
	}

	if i.recursive {
		return i.o.objectGoReflect._enumerate(true)()
	}

	return propIterItem{}, nil
}

func (o *objectGoMapReflect) _enumerate(recursive bool) iterNextFunc {
	r := &gomapReflectPropIter{
		o:         o,
		keys:      o.value.MapKeys(),
		recursive: recursive,
	}
	return r.next
}

func (o *objectGoMapReflect) enumerate(all, recursive bool) iterNextFunc {
	return (&propFilterIter{
		wrapped: o._enumerate(recursive),
		all:     all,
		seen:    make(map[string]bool),
	}).next
}

func (o *objectGoMapReflect) equal(other objectImpl) bool {
	if other, ok := other.(*objectGoMapReflect); ok {
		return o.value.Interface() == other.value.Interface()
	}
	return false
}
