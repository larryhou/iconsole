package ns

import (
	"fmt"
	"reflect"
	"time"

	"howett.net/plist"
)

const NSNull = "$null"

type ArchiverRoot struct {
	Root plist.UID `plist:"root"`
}

type KeyedArchiver struct {
	Archiver string        `plist:"$archiver"`
	Objects  []interface{} `plist:"$objects"`
	Top      ArchiverRoot  `plist:"$top"`
	Version  int           `plist:"$version"`
}

func (this KeyedArchiver) UID() plist.UID {
	return plist.UID(len(this.Objects))
}

func NewKeyedArchiver() *KeyedArchiver {
	return &KeyedArchiver{
		Archiver: "NSKeyedArchiver",
		Version:  100000,
	}
}

type ArchiverClasses struct {
	Classes   []string `plist:"$classes"`
	ClassName string   `plist:"$classname"`
}

var (
	NSMutableDictionaryClass = &ArchiverClasses{
		Classes:   []string{"NSMutableDictionary", "NSDictionary", "NSObject"},
		ClassName: "NSMutableDictionary",
	}
	NSDictionaryClass = &ArchiverClasses{
		Classes:   []string{"NSDictionary", "NSObject"},
		ClassName: "NSDictionary",
	}
	NSMutableArrayClass = &ArchiverClasses{
		Classes:   []string{"NSMutableArray", "NSArray", "NSObject"},
		ClassName: "NSMutableArray",
	}
	NSArrayClass = &ArchiverClasses{
		Classes:   []string{"NSArray", "NSObject"},
		ClassName: "NSArray",
	}
	NSMutableDataClass = &ArchiverClasses{
		Classes:   []string{"NSMutableArray", "NSArray", "NSObject"},
		ClassName: "NSMutableArray",
	}
	NSDataClass = &ArchiverClasses{
		Classes:   []string{"NSData", "NSObject"},
		ClassName: "NSData",
	}
	NSDateClass = &ArchiverClasses{
		Classes:   []string{"NSDate", "NSObject"},
		ClassName: "NSDate",
	}
	NSErrorClass = &ArchiverClasses{
		Classes:   []string{"NSError", "NSObject"},
		ClassName: "NSError",
	}
	NSSetClass = &ArchiverClasses{
		Classes:   []string{`NSSet`, `NSObject`},
		ClassName: `NSSet`,
	}
	NSMutableSetClass = &ArchiverClasses{
		Classes:   []string{`NSMutableSet`, `NSSet`, `NSObject`},
		ClassName: `NSMutableSet`,
	}
	DTTapMessageClass = &ArchiverClasses{
		Classes:   []string{`DTTapMessage`, `NSObject`},
		ClassName: `DTTapMessage`,
	}
	DTSysmonTapMessageClass = &ArchiverClasses{
		Classes:   []string{`DTSysmonTapMessage`, `DTTapMessage`, `NSObject`},
		ClassName: `DTSysmonTapMessage`,
	}
)

type NSObject struct {
	Class plist.UID `plist:"$class"`
}

type NSArray struct {
	NSObject
	Values []plist.UID `plist:"NS.objects"`
}

type NSDictionary struct {
	NSArray
	Keys []plist.UID `plist:"NS.keys"`
}

type NSData struct {
	NSObject
	Data []byte `plist:"NS.data"`
}

type GoNSError struct {
	NSCode     int
	NSDomain   string
	NSUserInfo interface{}
}

type DTTapMessage struct {
	Message map[string]any
}

func (x GoNSError) Error() string {
	return fmt.Sprintf(`%d/%s %s`, x.NSCode, x.NSDomain, x.NSUserInfo)
}

type NSKeyedArchiver struct {
	objRefVal []interface{}
	objRef    map[interface{}]plist.UID
}

func NewNSKeyedArchiver() *NSKeyedArchiver {
	a := &NSKeyedArchiver{
		objRef: make(map[interface{}]plist.UID),
	}

	return a
}

func (this *NSKeyedArchiver) id(v interface{}) plist.UID {
	var ref plist.UID
	if id, ok := this.objRef[v]; !ok {
		ref = plist.UID(len(this.objRef))
		this.objRefVal = append(this.objRefVal, v)
		this.objRef[v] = ref
	} else {
		ref = id
	}
	return ref
}

func (this *NSKeyedArchiver) flushToStruct(root *KeyedArchiver) {
	for i := 0; i < len(this.objRefVal); i++ {
		val := this.objRefVal[i]
		vt := reflect.ValueOf(val)
		if vt.Kind() == reflect.Ptr {
			val = vt.Elem().Interface()
		}
		root.Objects = append(root.Objects, val)
	}
}

func (this *NSKeyedArchiver) clear() {
	this.objRef = make(map[interface{}]plist.UID)
	this.objRefVal = []interface{}{}
}

func (this *NSKeyedArchiver) Marshal(obj interface{}) ([]byte, error) {
	root := NewKeyedArchiver()

	this.id(NSNull)
	root.Top.Root = this.idAny(obj)

	this.flushToStruct(root)

	this.clear()

	return plist.Marshal(root, plist.BinaryFormat)
}

func (this *NSKeyedArchiver) idAny(obj any) plist.UID {
	v := reflect.ValueOf(obj)
	t := v.Type()

	var uid plist.UID
	switch t.Kind() {
	case reflect.Map:
		uid = this.idDictionary(v)
	case reflect.Slice, reflect.Array:
		uid = this.idArray(v)
	case reflect.String:
		uid = this.id(obj)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Bool,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		uid = this.id(obj)
	}

	return uid
}

func (this *NSKeyedArchiver) idArray(v reflect.Value) plist.UID {
	t := v.Type()
	if t.Elem().Kind() == reflect.Uint8 {
		d := &NSData{}
		d.Class = this.id(NSDataClass)
		var w []byte
		for i := 0; i < v.Len(); i++ {
			w = append(w, uint8(v.Index(i).Uint()))
		}
		d.Data = w
		return this.id(d)
	}

	a := &NSArray{}
	a.Class = this.id(NSArrayClass)
	for i := 0; i < v.Len(); i++ {
		a.Values = append(a.Values, this.idAny(v.Index(i).Interface()))
	}

	return this.id(a)
}

func (this *NSKeyedArchiver) idDictionary(v reflect.Value) plist.UID {
	m := &NSDictionary{}
	m.Class = this.id(NSDictionaryClass)
	keys := v.MapKeys()
	for _, k := range keys {
		m.Keys = append(m.Keys, this.id(k.Interface()))
		m.Values = append(m.Values, this.idAny(v.MapIndex(k).Interface()))
	}

	return this.id(m)
}

func (this *NSKeyedArchiver) convertValue(v interface{}) interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		className := this.objRefVal[m["$class"].(plist.UID)].(map[string]interface{})["$classname"]

		switch className {
		case NSMutableDictionaryClass.Classes[0], NSDictionaryClass.Classes[0]:
			ret := make(map[string]interface{})
			keys := m["NS.keys"].([]interface{})
			values := m["NS.objects"].([]interface{})
			for i := 0; i < len(keys); i++ {
				key := this.objRefVal[keys[i].(plist.UID)].(string)
				val := this.convertValue(this.objRefVal[values[i].(plist.UID)])
				ret[key] = val
			}
			return ret
		case NSMutableSetClass.Classes[0], NSSetClass.Classes[0]:
			ret := make(map[any]struct{})
			values := m[`NS.objects`].([]any)
			for i := range values {
				val := this.convertValue(this.objRefVal[values[i].(plist.UID)])
				ret[val] = struct{}{}
			}
			return ret
		case NSMutableArrayClass.Classes[0], NSArrayClass.Classes[0]:
			ret := make([]interface{}, 0)
			values := m["NS.objects"].([]interface{})
			for i := 0; i < len(values); i++ {
				ret = append(ret, this.convertValue(values[i]))
			}
			return ret
		case NSMutableDataClass.Classes[0], NSDataClass.Classes[0]:
			return m["NS.data"].([]byte)
		case NSDateClass.Classes[0]:
			return time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC).
				Add(time.Duration(m["NS.time"].(float64)) * time.Second)
		case NSErrorClass.Classes[0]:
			err := &GoNSError{}
			err.NSCode = int(m["NSCode"].(uint64))
			err.NSDomain = this.objRefVal[m["NSDomain"].(plist.UID)].(string)
			err.NSUserInfo = this.convertValue(this.objRefVal[m["NSUserInfo"].(plist.UID)])
			return err
		case DTTapMessageClass.Classes[0], DTSysmonTapMessageClass.Classes[0]:
			tap := &DTTapMessage{
				Message: this.convertValue(m[`DTTapMessagePlist`]).(map[string]any),
			}
			return tap
		}
	} else if uid, ok := v.(plist.UID); ok {
		return this.convertValue(this.objRefVal[uid])
	}
	return v
}

func (this *NSKeyedArchiver) Unmarshal(b []byte) (interface{}, error) {
	archiver := &KeyedArchiver{}

	_, err := plist.Unmarshal(b, archiver)
	if err != nil {
		return nil, err
	}

	for _, v := range archiver.Objects {
		this.objRefVal = append(this.objRefVal, v)
	}

	ret := this.convertValue(this.objRefVal[archiver.Top.Root])

	this.clear()

	return ret, nil
}
