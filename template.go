package template

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Field struct {
	Type       string
	Value      interface{}       `json:"value,omitempty"`
	ArrayValue []*Field          `json:"array_value,omitempty"`
	MapValue   map[string]*Field `json:"map_value,omitempty"`
	err        string
}

func newNilField() *Field {
	return &Field{
		Type:  "nil",
		Value: nil,
	}
}

func newErrField(err string) *Field {
	return &Field{
		Type: "err",
		err:  err,
	}
}

func Convert(st interface{}) (*Field, error) {
	if st == nil {
		return newNilField(), nil
	}

	// 如果是 field,*field 类型, 直接返回
	if v, ok := st.(*Field); ok {
		return v, nil
	}
	if v, ok := st.(Field); ok {
		return &v, nil
	}

	typ := reflect.TypeOf(st)
	val := reflect.ValueOf(st)

	//kd := val.Kind() //获取到底层类型 指针类型要转换回真实类型
	if typ.Kind() == reflect.Ptr {
		val = reflect.ValueOf(st).Elem()
		typ = reflect.TypeOf(st).Elem()
	}

	field := newNilField()
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field = &Field{
			Type:  "int",
			Value: val.Int(),
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field = &Field{
			Type:  "uint",
			Value: val.Uint(),
		}
	case reflect.Float32, reflect.Float64:
		field = &Field{
			Type:  "float",
			Value: val.Float(),
		}
	case reflect.String:
		field = &Field{
			Type:  "string",
			Value: val.String(),
		}
	case reflect.Bool:
		field = &Field{
			Type:  "bool",
			Value: val.Bool(),
		}
	case reflect.Array, reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return &Field{
				Type:  "[]byte",
				Value: val.Bytes(),
			}, nil
		}

		num := val.Len()
		field = &Field{
			Type:       "array",
			ArrayValue: make([]*Field, num),
		}
		for i := 0; i < num; i++ {
			f, err := Convert(val.Index(i).Interface())
			if err != nil {
				return newNilField(), err
			}
			field.ArrayValue[i] = f
		}
	case reflect.Map:
		list := val.MapRange()
		field = &Field{
			Type:     "map",
			MapValue: make(map[string]*Field, 0),
		}

		for list.Next() {
			k := list.Key()
			v := list.Value()
			f, err := Convert(v.Interface())
			if err != nil {
				return newNilField(), err
			}
			field.MapValue[k.String()] = f
		}
	case reflect.Struct:
		field = &Field{
			Type:     "map",
			MapValue: make(map[string]*Field, 0),
		}

		for i := 0; i < val.NumField(); i++ {
			//判断是否为可导出字段
			if val.Field(i).CanInterface() {
				name := typ.Field(i).Name
				if tagVal := typ.Field(i).Tag.Get("json"); tagVal != "" {
					name = tagVal
				}
				f, err := Convert(val.Field(i).Interface())
				if err != nil {
					return newNilField(), err
				}
				field.MapValue[name] = f
			}
		}
	default:
		return newNilField(), errors.New(fmt.Sprintf("type:%T, value:%v  cannot support convert", st, st))
	}

	return field, nil
}

func (f *Field) Get(paths ...string) *Field {
	if err := f.Error(); err != nil {
		return newErrField(err.Error())
	}

	tmp := f
	num := len(paths)
	for i := 0; i < num; i++ {
		path := paths[i]
		switch tmp.Type {
		case "nil", "err":
			return newErrField("数据类型:" + tmp.Type + " 不支持递归")
		case "int", "uint", "float", "string", "bool":
			if num-i < 0 {
				return newErrField("基础数据类型,已达终态,无法继续递归")
			}
			return tmp
		case "map":
			if mf, ok := tmp.MapValue[path]; !ok {
				return newErrField("map路径" + strings.Join(paths[:i+1], "||") + "不存在")
			} else {
				tmp = mf
			}
		case "array":
			index, err := strconv.Atoi(path)
			if err != nil || index > len(tmp.ArrayValue)-1 {
				return newErrField("array路径 [" + strings.Join(paths[:i+1], ".") + "] 不存在")
			}

			tmp = tmp.ArrayValue[index]
		}
	}
	return tmp
}

func (f *Field) Error() error {
	if f.err != "" {
		return errors.New("has error, err: " + f.err)
	}

	if f.Type == "nil" {
		return errors.New("has error, type == nil: ")
	}

	return nil
}

func (f *Field) Int() (int64, error) {
	if err := f.Error(); err != nil {
		return 0, err
	}

	switch f.Type {
	case "int":
		if i, ok := f.Value.(int64); ok {
			return i, nil
		}
	case "uint":
		if i, ok := f.Value.(uint64); ok {
			return int64(i), nil
		}
	case "float":
		if i, ok := f.Value.(float64); ok {
			return int64(i), nil
		}
	case "bool":
		if i, ok := f.Value.(bool); ok {
			if i {
				return 1, nil
			}
			return 0, nil
		}
	case "string":
		if s, ok := f.Value.(string); ok {
			if i, err := strconv.Atoi(s); err == nil {
				return int64(i), nil
			}
		}
	}

	return 0, errors.New(fmt.Sprintf("%v, %s cannot convert int64", f.Value, f.Type))
}

func (f *Field) UInt() (uint64, error) {
	if err := f.Error(); err != nil {
		return 0, err
	}

	switch f.Type {
	case "int":
		if i, ok := f.Value.(int64); ok {
			return uint64(i), nil
		}
	case "uint":
		if i, ok := f.Value.(uint64); ok {
			return i, nil
		}
	case "float":
		if i, ok := f.Value.(float64); ok {
			return uint64(i), nil
		}
	case "bool":
		if i, ok := f.Value.(bool); ok {
			if i {
				return 1, nil
			}
			return 0, nil
		}
	case "string":
		if s, ok := f.Value.(string); ok {
			if i, err := strconv.Atoi(s); err == nil {
				return uint64(i), nil
			}
		}
	}

	return 0, errors.New(fmt.Sprintf("%v, %s cannot convert uint64", f.Value, f.Type))
}

func (f *Field) Float() (float64, error) {
	if err := f.Error(); err != nil {
		return 0, err
	}

	switch f.Type {
	case "int":
		if i, ok := f.Value.(int64); ok {
			return float64(i), nil
		}
	case "uint":
		if i, ok := f.Value.(uint64); ok {
			return float64(i), nil
		}
	case "float":
		if i, ok := f.Value.(float64); ok {
			return i, nil
		}
	case "bool":
		if i, ok := f.Value.(bool); ok {
			if i {
				return 1, nil
			}
			return 0, nil
		}
	case "string":
		if s, ok := f.Value.(string); ok {
			if float, err := strconv.ParseFloat(s, 64); err == nil {
				return float, nil
			}
		}
	}

	return 0, errors.New(fmt.Sprintf("%v, %s cannot convert float64", f.Value, f.Type))
}

func (f *Field) String() (string, error) {
	if err := f.Error(); err != nil {
		return "", err
	}

	switch f.Type {
	case "int":
		if i, ok := f.Value.(int64); ok {
			return strconv.FormatInt(i, 10), nil
		}
	case "uint":
		if i, ok := f.Value.(uint64); ok {
			return strconv.FormatUint(i, 10), nil
		}
	case "float":
		if i, ok := f.Value.(float64); ok {
			return strconv.FormatFloat(i, 'f', -1, 64), nil
		}
	case "bool":
		if i, ok := f.Value.(bool); ok {
			if i {
				return "true", nil
			}
			return "false", nil
		}
	case "string":
		if s, ok := f.Value.(string); ok {
			return s, nil
		}
	}

	return "", errors.New(fmt.Sprintf("%v, %s cannot convert string", f.Value, f.Type))
}

func (f *Field) Bool() (bool, error) {
	if err := f.Error(); err != nil {
		return false, err
	}

	switch f.Type {
	case "int":
		if i, ok := f.Value.(int64); ok {
			if i == 0 {
				return false, nil
			}
			return true, nil
		}
	case "uint":
		if i, ok := f.Value.(uint64); ok {
			if i == 0 {
				return false, nil
			}
			return true, nil
		}
	case "float":
		if i, ok := f.Value.(float64); ok {
			if i == 0 {
				return false, nil
			}
			return true, nil
		}
	case "bool":
		if i, ok := f.Value.(bool); ok {
			return i, nil
		}
	case "string":
		if s, ok := f.Value.(string); ok {
			if s == "" {
				return false, nil
			}
			return true, nil
		}
	}

	return false, errors.New(fmt.Sprintf("%v, %s cannot convert bool", f.Value, f.Type))
}

func (f *Field) Bytes() ([]byte, error) {
	if err := f.Error(); err != nil {
		return nil, err
	}

	switch f.Type {
	case "[]byte":
		if b, ok := f.Value.([]byte); ok {
			return b, nil
		}
	case "string":
		if s, ok := f.Value.(string); ok {
			return []byte(s), nil
		}
	}

	return nil, errors.New(fmt.Sprintf("%v, %s cannot convert []byte", f.Value, f.Type))
}

func (f *Field) Interface() interface{} {
	if err := f.Error(); err != nil {
		return nil
	}

	switch f.Type {
	case "map":
		return f.MapValue
	case "array":
		return f.ArrayValue
	}

	return f.Value
}

func (f *Field) JsonUnmarshal(i interface{}) error {
	if err := f.Error(); err != nil {
		return err
	}

	if f.Type != "[]byte" {
		return errors.New("only []byte support json.Unmarshal")
	}

	if b, ok := f.Value.([]byte); ok {
		return json.Unmarshal(b, i)
	}

	return errors.New(fmt.Sprintf("%v, %s cannot convert []byte", f.Value, f.Type))
}

func (f *Field) ByteToField() *Field {
	if err := f.Error(); err != nil {
		return newErrField(err.Error())
	}

	if f.Type != "[]byte" {
		return newErrField("only []byte support json.Unmarshal")
	}

	if b, ok := f.Value.([]byte); ok {
		var i interface{}
		if err := json.Unmarshal(b, &i); err != nil {
			return newErrField(err.Error())
		}
		field, err := Convert(i)
		if err != nil {
			return newErrField(err.Error())
		}

		return field
	}

	return newErrField(fmt.Sprintf("%v, %s cannot convert []byte", f.Value, f.Type))
}

func (f *Field) ToJson() (string, error) {
	if err := f.Error(); err != nil {
		return "", err
	}

	r, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	return string(r), err
}

func (f *Field) MapRange(fn func(string, *Field) error) error {
	if err := f.Error(); err != nil {
		return err
	}

	if f.Type != "map" {
		return errors.New("only map support range")
	}

	for k, item := range f.MapValue {
		if err := fn(k, item); err != nil {
			return err
		}
	}

	return nil
}

func (f *Field) ArrayRange(fn func(int, *Field) error) error {
	if err := f.Error(); err != nil {
		return err
	}

	if f.Type != "array" {
		return errors.New("only map support range")
	}

	for k, item := range f.ArrayValue {
		if err := fn(k, item); err != nil {
			return err
		}
	}

	return nil
}
