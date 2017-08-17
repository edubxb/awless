package ast

type CompositeValue interface {
	Holes() []string
	Refs() []string
	Value() interface{}
}

func NewCompositeValue(values ...CompositeValue) CompositeValue {
	return &listValue{vals: values}
}

type listValue struct {
	vals []CompositeValue
}

func (l *listValue) Holes() (res []string) {
	for _, val := range l.vals {
		res = append(res, val.Holes()...)
	}
	return
}

func (l *listValue) Refs() (res []string) {
	for _, val := range l.vals {
		res = append(res, val.Refs()...)
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

type interfaceValue struct {
	val interface{}
}

func (i *interfaceValue) Holes() (res []string) {
	return
}

func (i *interfaceValue) Refs() (res []string) {
	return
}

func (i *interfaceValue) Value() interface{} {
	return i.val
}

type holeValue struct {
	hole string
	val  interface{}
}

func (h *holeValue) Holes() []string {
	return []string{h.hole}
}

func (h *holeValue) Refs() (res []string) {
	return
}

func (h *holeValue) Value() interface{} {
	return h.val
}

type referenceValue struct {
	ref string
	val interface{}
}

func (r *referenceValue) Holes() (res []string) {
	return
}

func (r *referenceValue) Refs() []string {
	return []string{r.ref}
}

func (r *referenceValue) Value() interface{} {
	return r.val
}
