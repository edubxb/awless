package ast

import (
	"reflect"
	"testing"
)

func TestCompositeValues(t *testing.T) {
	tcases := []struct {
		val          CompositeValue
		holesFillers map[string]interface{}
		expHoles     []string
		expRefs      []string
		expValue     interface{}
	}{
		{val: &interfaceValue{val: "test"}, expValue: "test"},
		{val: &interfaceValue{val: 10}, expValue: 10},
		{val: &holeValue{hole: "myhole"}, expHoles: []string{"myhole"}},
		{val: &referenceValue{ref: "myref"}, expRefs: []string{"myref"}},
		{
			val: NewCompositeValue(
				&interfaceValue{val: "test"},
				&interfaceValue{val: 10},
				&holeValue{hole: "myhole"},
				&referenceValue{ref: "myref"},
			),
			expRefs:  []string{"myref"},
			expHoles: []string{"myhole"},
			expValue: []interface{}{"test", 10},
		},
		{val: &holeValue{hole: "myhole"}, holesFillers: map[string]interface{}{"myhole": "my-value"}, expValue: "my-value"},
		{
			val: NewCompositeValue(
				&interfaceValue{val: "test"},
				&interfaceValue{val: 10},
				&holeValue{hole: "myhole"},
				&referenceValue{ref: "myref"},
			),
			holesFillers: map[string]interface{}{"myhole": "my-value"},
			expRefs:      []string{"myref"},
			expValue:     []interface{}{"test", 10, "my-value"},
		},
	}

	for i, tcase := range tcases {
		tcase.val.ProcessHoles(tcase.holesFillers)
		if got, want := tcase.val.GetHoles(), tcase.expHoles; !reflect.DeepEqual(got, want) {
			t.Fatalf("%d: holes: got %#v, want %#v", i+1, got, want)
		}
		if got, want := tcase.val.GetRefs(), tcase.expRefs; !reflect.DeepEqual(got, want) {
			t.Fatalf("%d: refs: got %#v, want %#v", i+1, got, want)
		}
		if got, want := tcase.val.Value(), tcase.expValue; !reflect.DeepEqual(got, want) {
			t.Fatalf("%d: value: got %#v, want %#v", i+1, got, want)
		}
	}
}
