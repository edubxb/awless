package ast

import (
	"fmt"
	"net"
	"strconv"
)

func (a *AST) addAction(text string) {
	if IsInvalidAction(text) {
		panic(fmt.Errorf("unknown action '%s'", text))
	}

	cmd := &CommandNode{Action: text}
	action := &ActionNode{Action: text}

	decl := a.currentDeclaration()
	if decl != nil {
		decl.Expr = cmd
	} else {
		node := a.currentCommand()
		if node == nil {
			a.addStatement(cmd)
		} else {
			node.Action = text
		}

		actionNode := a.currentAction()
		if actionNode == nil {
			a.addActionStatement(action)
		} else {
			actionNode.Action = text
		}
	}
}

func (a *AST) addEntity(text string) {
	if IsInvalidEntity(text) {
		panic(fmt.Errorf("unknown entity '%s'", text))
	}
	node := a.currentCommand()
	node.Entity = text
	action := a.currentAction()
	if action != nil {
		action.Entity = text
	}
}

func (a *AST) addValue() {
	val := &ValueNode{}

	decl := a.currentDeclaration()
	if decl != nil {
		decl.Expr = val
	}
}

func (a *AST) addDeclarationIdentifier(text string) {
	a.addStatement(&DeclarationNode{Ident: text})
}

func (a *AST) LineDone() {
	if currentAction := a.currentAction(); a.currentActionStatement != nil && a.currentActionStatement.Node != nil && currentAction != nil && a.actionNodeContainsList(currentAction) {
		a.Statements = append(a.Statements, a.currentActionStatement)
	} else if a.currentStatement != nil && a.currentStatement.Node != nil {
		a.Statements = append(a.Statements, a.currentStatement)
	}
	a.currentStatement = nil
	a.currentKey = ""
	a.currentActionStatement = nil
	a.currentListBuilder = nil
}

func (a *AST) addParam(i interface{}) {
	if node := a.currentCommand(); node != nil {
		node.Params[a.currentKey] = i
	} else {
		varDecl := a.currentDeclarationValue()
		varDecl.Value = i
	}
	if action := a.currentAction(); action != nil {
		action.Params[a.currentKey] = &interfaceValue{val: i}
	}
}

func (a *AST) addParamKey(text string) {
	node := a.currentCommand()
	if node.Params == nil {
		node.Refs = make(map[string]string)
		node.Params = make(map[string]interface{})
		node.Holes = make(map[string]string)
	}
	action := a.currentAction()
	if action != nil && action.Params == nil {
		action.Params = make(map[string]CompositeValue)
	}
	a.currentKey = text
}

func (a *AST) addAliasParam(text string) {
	a.addParam("@" + text)
}

func (a *AST) addParamValue(text string) {
	var val interface{}
	i, err := strconv.Atoi(text)
	if err == nil {
		val = i
	} else {
		f, err := strconv.ParseFloat(text, 64)
		if err == nil {
			val = f
		} else {
			val = text
		}
	}
	if a.currentListBuilder != nil {
		a.currentListBuilder.add(&interfaceValue{val: val})
	} else {
		a.addParam(val)
	}
}

func (a *AST) addFirstValueInList() {
	a.currentListBuilder = &listValueBuilder{}
}
func (a *AST) lastValueInList() {
	if action := a.currentAction(); action != nil {
		if a.currentListBuilder != nil {
			action.Params[a.currentKey] = a.currentListBuilder.build()
		}
	}
	a.currentListBuilder = nil
}

func (a *AST) addStringValue(text string) {
	if a.currentListBuilder != nil {
		a.currentListBuilder.add(&interfaceValue{val: text})
	} else {
		a.addParam(text)
	}
}

func (a *AST) addParamFloatValue(text string) {
	num, err := strconv.ParseFloat(text, 64)
	if err != nil {
		panic(fmt.Sprintf("cannot convert '%s' to float", text))
	}
	if a.currentListBuilder != nil {
		a.currentListBuilder.add(&interfaceValue{val: num})
	} else {
		a.addParam(num)
	}
}

func (a *AST) addParamIntValue(text string) {
	num, err := strconv.Atoi(text)
	if err != nil {
		panic(fmt.Sprintf("cannot convert '%s' to int", text))
	}
	if a.currentListBuilder != nil {
		a.currentListBuilder.add(&interfaceValue{val: num})
	} else {
		a.addParam(num)
	}
}

func (a *AST) addParamCidrValue(text string) {
	_, ipnet, err := net.ParseCIDR(text)
	if err != nil {
		panic(fmt.Sprintf("cannot convert '%s' to net cidr", text))
	}
	if a.currentListBuilder != nil {
		a.currentListBuilder.add(&interfaceValue{val: ipnet.String()})
	} else {
		a.addParam(ipnet.String())
	}
}

func (a *AST) addParamIpValue(text string) {
	ip := net.ParseIP(text)
	if ip == nil {
		panic(fmt.Sprintf("cannot convert '%s' to net ip", text))
	}
	if a.currentListBuilder != nil {
		a.currentListBuilder.add(&interfaceValue{val: ip.String()})
	} else {
		a.addParam(ip.String())
	}
}

func (a *AST) addParamRefValue(text string) {
	if a.currentListBuilder != nil {
		a.currentListBuilder.add(&referenceValue{ref: text})
	} else {
		if node := a.currentCommand(); node != nil {
			node.Refs[a.currentKey] = text
		}
	}
}

func (a *AST) addParamHoleValue(text string) {
	if a.currentListBuilder != nil {
		a.currentListBuilder.add(&holeValue{hole: text})
	} else {
		if node := a.currentCommand(); node != nil {
			node.Holes[a.currentKey] = text
		} else {
			varDecl := a.currentDeclarationValue()
			varDecl.Hole = text
		}
	}
}

func (a *AST) currentDeclaration() *DeclarationNode {
	st := a.currentStatement
	if st == nil {
		return nil
	}

	switch st.Node.(type) {
	case *DeclarationNode:
		return st.Node.(*DeclarationNode)
	}

	return nil
}

func (a *AST) currentCommand() *CommandNode {
	st := a.currentStatement
	if st == nil {
		return nil
	}

	switch st.Node.(type) {
	case *CommandNode:
		return st.Node.(*CommandNode)
	case *DeclarationNode:
		expr := st.Node.(*DeclarationNode).Expr
		switch expr.(type) {
		case *CommandNode:
			return expr.(*CommandNode)
		}
		return nil
	default:
		return nil
	}
}

func (a *AST) currentAction() *ActionNode {
	st := a.currentActionStatement
	if st == nil {
		return nil
	}

	switch st.Node.(type) {
	case *ActionNode:
		return st.Node.(*ActionNode)
	case *DeclarationNode:
		expr := st.Node.(*DeclarationNode).Expr
		switch expr.(type) {
		case *ActionNode:
			return expr.(*ActionNode)
		}
		return nil
	default:
		return nil
	}
}

func (a *AST) actionNodeContainsList(action *ActionNode) bool {
	for _, p := range action.Params {
		if _, ok := p.(*listValue); ok {
			return true
		}
	}
	return false
}

func (a *AST) currentDeclarationValue() *ValueNode {
	st := a.currentStatement
	if st == nil {
		return nil
	}

	switch st.Node.(type) {
	case *DeclarationNode:
		expr := st.Node.(*DeclarationNode).Expr
		switch expr.(type) {
		case *ValueNode:
			return expr.(*ValueNode)
		}
		return nil
	default:
		return nil
	}
}

func (a *AST) addStatement(n Node) {
	stat := &Statement{Node: n}
	a.currentStatement = stat
}

func (a *AST) addActionStatement(n Node) {
	stat := &Statement{Node: n}
	a.currentActionStatement = stat
}

type listValueBuilder struct {
	vals []CompositeValue
}

func (c *listValueBuilder) add(v CompositeValue) *listValueBuilder {
	c.vals = append(c.vals, v)
	return c
}

func (c *listValueBuilder) build() CompositeValue {
	return &listValue{c.vals}
}
