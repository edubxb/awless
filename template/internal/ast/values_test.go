package ast

import (
	"reflect"
	"testing"
)

func TestCompositeValues(t *testing.T) {
	tcases := []struct {
		val      CompositeValue
		expHoles []string
		expRefs  []string
		expValue interface{}
	}{
		{val: &interfaceValue{val: "test"}, expValue: "test"},
		{val: &interfaceValue{val: 10}, expValue: 10},
		{val: &holeValue{hole: "myhole"}, expHoles: []string{"myhole"}},
		{val: &referenceValue{ref: "myref"}, expRefs: []string{"myref"}},
		{val: NewCompositeValue(
			&interfaceValue{val: "test"},
			&interfaceValue{val: 10},
			&holeValue{hole: "myhole"},
			&referenceValue{ref: "myref"},
		),
			expRefs:  []string{"myref"},
			expHoles: []string{"myhole"},
			expValue: []interface{}{"test", 10},
		},
	}

	for i, tcase := range tcases {
		if got, want := tcase.val.Holes(), tcase.expHoles; !reflect.DeepEqual(got, want) {
			t.Fatalf("%d: got %#v, want %#v", i+1, got, want)
		}
		if got, want := tcase.val.Refs(), tcase.expRefs; !reflect.DeepEqual(got, want) {
			t.Fatalf("%d: got %#v, want %#v", i+1, got, want)
		}
		if got, want := tcase.val.Value(), tcase.expValue; !reflect.DeepEqual(got, want) {
			t.Fatalf("%d: got %#v, want %#v", i+1, got, want)
		}
	}
}
