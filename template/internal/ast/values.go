package ast

import (
	"bytes"
	"fmt"
)

type CompositeValue interface {
	WithHoles
	String() string
	GetRefs() []string
	Value() interface{}
}

func NewCompositeValue(values ...CompositeValue) CompositeValue {
	return &listValue{vals: values}
}

type listValue struct {
	vals []CompositeValue
}

func (l *listValue) GetHoles() (res []string) {
	for _, val := range l.vals {
		res = append(res, val.GetHoles()...)
	}
	return
}

func (l *listValue) GetRefs() (res []string) {
	for _, val := range l.vals {
		res = append(res, val.GetRefs()...)
	}
	return
}

func (l *listValue) Value() interface{} {
	var res []interface{}
	for _, val := range l.vals {
		if v := val.Value(); v != nil {
			res = append(res, v)
		}
	}
	return res
}

func (l *listValue) ProcessHoles(fills map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})
	for _, val := range l.vals {
		valProc := val.ProcessHoles(fills)
		for k, v := range valProc {
			processed[k] = v
		}
	}
	return processed
}

func (l *listValue) String() string {
	var buff bytes.Buffer
	buff.WriteRune('[')
	for i, val := range l.vals {
		buff.WriteString(val.String())
		if i < len(l.vals)-1 {
			buff.WriteString(",")
		}
	}
	buff.WriteRune(']')
	return buff.String()
}

func (l *listValue) GetAliases() (res []string) {
	for _, val := range l.vals {
		if alias, ok := val.(WithAlias); ok {
			res = append(res, alias.GetAliases()...)
		}
	}
	return
}

func (l *listValue) ResolveAlias(resolvFunc func(string) string) {
	for _, val := range l.vals {
		if alias, ok := val.(WithAlias); ok {
			alias.ResolveAlias(resolvFunc)
		}
	}

}

type interfaceValue struct {
	val interface{}
}

func (i *interfaceValue) GetHoles() (res []string) {
	return
}

func (i *interfaceValue) GetRefs() (res []string) {
	return
}

func (i *interfaceValue) Value() interface{} {
	return i.val
}

func (i *interfaceValue) ProcessHoles(map[string]interface{}) map[string]interface{} {
	return make(map[string]interface{})
}

func (i *interfaceValue) String() string {
	return printParamValue(i.val)
}

type holeValue struct {
	hole string
	val  interface{}
}

func (h *holeValue) GetHoles() (res []string) {
	if h.val == nil {
		res = append(res, h.hole)
	}
	return
}

func (h *holeValue) GetRefs() (res []string) {
	return
}

func (h *holeValue) Value() interface{} {
	return h.val
}

func (h *holeValue) ProcessHoles(fills map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})
	if fill, ok := fills[h.hole]; ok {
		h.val = fill
		processed[h.hole] = fill
	}
	return processed
}

func (h *holeValue) String() string {
	if h.val != nil {
		return printParamValue(h.val)
	} else {
		return fmt.Sprintf("{%s}", h.hole)
	}
}

type WithAlias interface {
	GetAliases() []string
	ResolveAlias(func(string) string)
}

type aliasValue struct {
	alias string
	val   interface{}
}

func (a *aliasValue) GetHoles() (res []string) {
	return
}

func (a *aliasValue) GetRefs() (res []string) {
	return
}

func (a *aliasValue) Value() interface{} {
	return a.val
}

func (a *aliasValue) ProcessHoles(fills map[string]interface{}) map[string]interface{} {
	return make(map[string]interface{})
}

func (a *aliasValue) String() string {
	if a.val != nil {
		return printParamValue(a.val)
	} else {
		return fmt.Sprintf("@%s", a.alias)
	}
}

func (a *aliasValue) GetAliases() []string {
	return []string{a.alias}
}

func (a *aliasValue) ResolveAlias(resolvFunc func(string) string) {
	a.val = resolvFunc(a.alias)
}

type referenceValue struct {
	ref string
	val interface{}
}

func (r *referenceValue) GetHoles() (res []string) {
	return
}

func (r *referenceValue) GetRefs() []string {
	return []string{r.ref}
}

func (r *referenceValue) Value() interface{} {
	return r.val
}

func (r *referenceValue) ProcessHoles(map[string]interface{}) map[string]interface{} {
	return make(map[string]interface{})
}

func (r *referenceValue) String() string {
	if r.val != nil {
		return printParamValue(r.val)
	} else {
		return fmt.Sprintf("$%s", r.ref)
	}
}
