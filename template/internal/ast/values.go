package ast

type CompositeValue interface {
	WithHoles
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
func (l *referenceValue) ProcessHoles(map[string]interface{}) map[string]interface{} {
	return make(map[string]interface{})
}
