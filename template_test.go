package template

import (
	"encoding/json"
	"fmt"
	"testing"
)

type zzz struct {
	Boo   bool `json:"boo2"`
	Uints []interface{}
}

type xxx struct {
	Name     string                 `json:"name"`
	Age      int                    `json:"age"`
	Age64    float64                `json:"age64"`
	Boo      bool                   `json:"boo"`
	Uints    []uint8                `json:"uints"`
	Ints     []int8                 `json:"ints"`
	Arr      []interface{}          `json:"arr"`
	IntMaps  map[string]interface{} `json:"int_maps"`
	IntMaps2 map[string]int16       `json:"int_maps_2"`
	ZZZ      zzz                    `json:"zzz"`
}

var aaa = &xxx{
	Name:  "张三",
	Age:   123,
	Age64: 112.22,
	Boo:   true,
	Uints: []uint8{11, 22, 33},
	Ints:  []int8{-1, -2, 11, 6},
	Arr: []interface{}{
		"ddd", true,
		123,
		123.22,
		&zzz{Boo: true, Uints: []interface{}{11, 22, 33, "ddd", "ccc"}},
	},
	IntMaps: map[string]interface{}{
		"float": 112.11,
		"bool":  false,
		"int":   -1,
		"zzz":   &zzz{Boo: true, Uints: []interface{}{"hello", "world", 2020, 11.11}},
	},
	IntMaps2: map[string]int16{
		"nn": -1,
		"aa": 11,
		"zz": 22,
	},
	ZZZ: zzz{Boo: true, Uints: []interface{}{"hello", "world", 2020, 11.11}},
}

func TestField(t *testing.T) {
	oom, _ := json.Marshal(aaa)
	ccc := map[string]json.RawMessage{"data": oom}

	if ddd, err := ConvertStruct(ccc); err == nil {
		kkk := ddd.Get("data").ByteToField()
		err = kkk.Get("int_maps").MapRange(func(s string, field *Field) error {
			if s == "zzz" {
				err = field.Get("Uints").ArrayRange(func(i int, field1 *Field) error {
					fmt.Printf("array::: name:%d, type:%s, T: %T, value:%v\n", i, field1.Type, field1.Interface(), field1.Interface())
					return nil
				})
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
	}
}
