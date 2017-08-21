package ast

import (
	"reflect"
	"testing"
)

func TestCompositeValues(t *testing.T) {
	tcases := []struct {
		val          CompositeValue
		holesFillers map[string]interface{}
		refsFillers  map[string]interface{}
		expHoles     []string
		expRefs      []string
		expAliases   []string
		expValue     interface{}
	}{
		{val: &interfaceValue{val: "test"}, expValue: "test"},
		{val: &interfaceValue{val: 10}, expValue: 10},
		{val: &holeValue{hole: "myhole"}, expHoles: []string{"myhole"}},
		{val: &referenceValue{ref: "myref"}, expRefs: []string{"myref"}},
		{val: &aliasValue{alias: "myalias"}, expAliases: []string{"myalias"}},
		{
			val: NewCompositeValue(
				&interfaceValue{val: "test"},
				&interfaceValue{val: 10},
				&holeValue{hole: "myhole"},
				&referenceValue{ref: "myref"},
				&aliasValue{alias: "myalias"},
			),
			expRefs:    []string{"myref"},
			expHoles:   []string{"myhole"},
			expValue:   []interface{}{"test", 10},
			expAliases: []string{"myalias"},
		},
		{val: &holeValue{hole: "myhole"}, holesFillers: map[string]interface{}{"myhole": "my-value"}, expValue: "my-value"},
		{
			val: NewCompositeValue(
				&interfaceValue{val: "test"},
				&interfaceValue{val: 10},
				&holeValue{hole: "myhole"},
				&referenceValue{ref: "myref"},
			),
			refsFillers:  map[string]interface{}{"myref": "refvalue"},
			holesFillers: map[string]interface{}{"myhole": "my-value"},
			expValue:     []interface{}{"test", 10, "my-value", "refvalue"},
		},
	}

	for i, tcase := range tcases {
		if withHoles, ok := tcase.val.(WithHoles); ok {
			withHoles.ProcessHoles(tcase.holesFillers)
		}
		if withRefs, ok := tcase.val.(WithRefs); ok {
			withRefs.ProcessRefs(tcase.refsFillers)
		}
		if len(tcase.expHoles) > 0 {
			withHoles, ok := tcase.val.(WithHoles)
			if !ok {
				t.Fatalf("%d: holes: expect value to implement `WithHoles`", i+1)
			}
			if got, want := withHoles.GetHoles(), tcase.expHoles; !reflect.DeepEqual(got, want) {
				t.Fatalf("%d: holes: got %#v, want %#v", i+1, got, want)
			}
		}
		if len(tcase.expRefs) > 0 {
			withRefs, ok := tcase.val.(WithRefs)
			if !ok {
				t.Fatalf("%d: refs: expect value to implement `WithRefs`", i+1)
			}
			if got, want := withRefs.GetRefs(), tcase.expRefs; !reflect.DeepEqual(got, want) {
				t.Fatalf("%d: refs: got %#v, want %#v", i+1, got, want)
			}
		}
		if len(tcase.expAliases) > 0 {
			aliasVal, ok := tcase.val.(WithAlias)
			if !ok {
				t.Fatalf("%d: aliases: expect value to implement `WithAlias`", i+1)
			}
			if got, want := aliasVal.GetAliases(), tcase.expAliases; !reflect.DeepEqual(got, want) {
				t.Fatalf("%d: aliases: got %#v, want %#v", i+1, got, want)
			}
		}
		if got, want := tcase.val.Value(), tcase.expValue; !reflect.DeepEqual(got, want) {
			t.Fatalf("%d: value: got %#v, want %#v", i+1, got, want)
		}
	}
}

func TestCompositeValuesStringer(t *testing.T) {
	tcases := []struct {
		val    CompositeValue
		expect string
	}{
		{val: &interfaceValue{val: "test"}, expect: "test"},
		{val: &interfaceValue{val: "te\"st"}, expect: "'te\"st'"},
		{val: &interfaceValue{val: "te'st"}, expect: "\"te'st\""},
		{val: &interfaceValue{val: 10}, expect: "10"},
		{val: &interfaceValue{val: "10"}, expect: "'10'"},
		{val: &holeValue{hole: "myhole"}, expect: "{myhole}"},
		{val: &referenceValue{ref: "myref"}, expect: "$myref"},
		{val: &aliasValue{alias: "myalias"}, expect: "@myalias"},
		{
			val: NewCompositeValue(
				&interfaceValue{val: "test"},
				&interfaceValue{val: 10},
				&holeValue{hole: "myhole"},
				&referenceValue{ref: "myref"},
				&aliasValue{alias: "myalias"},
			),
			expect: "[test,10,{myhole},$myref,@myalias]",
		},
	}

	for i, tcase := range tcases {
		if got, want := tcase.val.String(), tcase.expect; got != want {
			t.Fatalf("%d: got %s, want %s", i+1, got, want)
		}
	}
}
