package template

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wallix/awless/logger"
	"github.com/wallix/awless/template/driver"
	"github.com/wallix/awless/template/internal/ast"
)

type Env struct {
	Driver driver.Driver

	ResolvedReferences map[string]interface{}

	Fillers          map[string]interface{}
	DefLookupFunc    DefinitionLookupFunc
	AliasFunc        func(entity, key, alias string) string
	MissingHolesFunc func(string) interface{}
	Log              *logger.Logger

	processedFillers map[string]interface{}
}

func NewEnv() *Env {
	return &Env{
		AliasFunc:          nil,
		MissingHolesFunc:   nil,
		Log:                logger.DiscardLogger,
		ResolvedReferences: make(map[string]interface{}),
		processedFillers:   make(map[string]interface{}),
	}
}

func (e *Env) AddFillers(fills ...map[string]interface{}) {
	if e.Fillers == nil {
		e.Fillers = make(map[string]interface{})
	}

	for _, f := range fills {
		for k, v := range f {
			e.Fillers[k] = v
		}
	}
}

func (e *Env) addToProcessedFillers(fills ...map[string]interface{}) {
	if e.processedFillers == nil {
		e.processedFillers = make(map[string]interface{})
	}

	for _, f := range fills {
		for k, v := range f {
			e.processedFillers[k] = v
		}
	}
}

func (e *Env) GetProcessedFillers() (copy map[string]interface{}) {
	copy = make(map[string]interface{}, 0)
	for k, v := range e.processedFillers {
		copy[k] = v
	}
	return
}

type Mode []compileFunc

var (
	LenientCompileMode = []compileFunc{
		resolveAgainstDefinitions,
		checkInvalidReferenceDeclarations,
		resolveHolesPass,
		resolveMissingHolesPass,
		replaceVariableValuePass,
		removeValueStatementsPass,
		resolveAliasPass,
	}

	NormalCompileMode = append(
		LenientCompileMode,
		failOnUnresolvedHoles,
		failOnUnresolvedAlias,
	)
)

func Compile(tpl *Template, env *Env, mode ...Mode) (*Template, *Env, error) {
	var pass *multiPass

	if len(mode) > 0 {
		pass = newMultiPass(mode[0]...)
	} else {
		pass = newMultiPass(NormalCompileMode...)
	}

	return pass.compile(tpl, env)
}

type compileFunc func(*Template, *Env) (*Template, *Env, error)

// Leeloo Dallas
type multiPass struct {
	passes []compileFunc
}

func newMultiPass(passes ...compileFunc) *multiPass {
	return &multiPass{passes: passes}
}

func (p *multiPass) compile(tpl *Template, env *Env) (newTpl *Template, newEnv *Env, err error) {
	newTpl, newEnv = tpl, env
	for _, pass := range p.passes {
		newTpl, newEnv, err = pass(newTpl, newEnv)
		if err != nil {
			return
		}
	}

	return
}

func resolveAgainstDefinitions(tpl *Template, env *Env) (*Template, *Env, error) {
	if env.DefLookupFunc == nil {
		return tpl, env, fmt.Errorf("definition lookup function is undefined")
	}
	each := func(cmd *ast.CommandNode) error {
		tplKey := fmt.Sprintf("%s%s", cmd.Action, cmd.Entity)
		def, ok := env.DefLookupFunc(tplKey)
		if !ok {
			return fmt.Errorf("cannot find template definition for '%s'", tplKey)
		}

		for _, key := range cmd.Keys() {
			var found bool

			for _, k := range def.Required() {
				if k == key {
					found = true
					break
				}
			}

			for _, k := range def.Extra() {
				if k == key {
					found = true
					break
				}
			}
			if !found {
				var extraParams, requiredParams string
				if len(def.Extra()) > 0 {
					extraParams = fmt.Sprintf("\n\t- extra params: %s", strings.Join(def.Extra(), ", "))
				}
				if len(def.Required()) > 0 {
					requiredParams = fmt.Sprintf("\n\t- required params: %s", strings.Join(def.Required(), ", "))
				}
				return fmt.Errorf("%s %s: unexpected param key '%s'%s%s\n", cmd.Action, cmd.Entity, key, requiredParams, extraParams)
			}
		}

		return nil
	}

	if err := tpl.visitCommandNodesE(each); err != nil {
		return tpl, env, err
	}

	tpl.visitCommandNodes(func(cmd *ast.CommandNode) {
		if cmd.Holes == nil {
			cmd.Holes = make(map[string]string)
		}
		key := fmt.Sprintf("%s%s", cmd.Action, cmd.Entity)
		def, _ := env.DefLookupFunc(key)
		for _, required := range def.Required() {
			var isInParams bool
			var isInRefs bool

			for k := range cmd.Params {
				if k == required {
					isInParams = true
				}
			}
			for k := range cmd.Refs {
				if k == required {
					isInRefs = true
				}
			}
			normalized := fmt.Sprintf("%s.%s", cmd.Entity, required)

			if isInParams || isInRefs {
				delete(cmd.Holes, normalized)
				continue
			} else {
				if _, ok := cmd.Holes[required]; !ok {
					cmd.Holes[required] = normalized
				}
			}
		}
	})

	return tpl, env, nil
}

func checkInvalidReferenceDeclarations(tpl *Template, env *Env) (*Template, *Env, error) {
	usedRefs := make(map[string]struct{})
	tpl.visitCommandNodes(func(cmd *ast.CommandNode) {
		for _, v := range cmd.Refs {
			usedRefs[v] = struct{}{}
		}
	})

	knownRefs := make(map[string]bool)

	var each = func(cmd *ast.CommandNode) error {
		for _, ref := range cmd.Refs {
			if _, ok := knownRefs[ref]; !ok {
				return fmt.Errorf("using reference '$%s' but '%s' is undefined in template\n", ref, ref)
			}
		}
		return nil
	}

	for _, st := range tpl.Statements {
		switch n := st.Node.(type) {
		case *ast.CommandNode:
			if err := each(n); err != nil {
				return tpl, env, err
			}
		case *ast.DeclarationNode:
			expr := st.Node.(*ast.DeclarationNode).Expr
			switch nn := expr.(type) {
			case *ast.CommandNode:
				if err := each(nn); err != nil {
					return tpl, env, err
				}
			}
		}
		if decl, isDecl := st.Node.(*ast.DeclarationNode); isDecl {
			ref := decl.Ident
			if _, ok := knownRefs[ref]; ok {
				return tpl, env, fmt.Errorf("using reference '$%s' but '%s' has already been assigned in template\n", ref, ref)
			}
			knownRefs[ref] = true
		}
	}

	return tpl, env, nil
}

func replaceVariableValuePass(tpl *Template, env *Env) (*Template, *Env, error) {
	tpl.visitDeclarationNodes(func(decl *ast.DeclarationNode) {
		if value, isValueNode := decl.Expr.(*ast.ValueNode); isValueNode && value.IsResolved() {
			env.ResolvedReferences[decl.Ident] = decl.Expr.Result()
		}
	})
	tpl.visitCommandNodes(func(n *ast.CommandNode) {
		n.ProcessRefs(env.ResolvedReferences)
	})

	env.Log.ExtraVerbosef("references resolved so far: %v", env.ResolvedReferences)

	return tpl, env, nil
}

func removeValueStatementsPass(tpl *Template, env *Env) (*Template, *Env, error) {
	newTpl := &Template{ID: tpl.ID, AST: tpl.AST.Clone()}
	newTpl.Statements = []*ast.Statement{}
	for _, stmt := range tpl.Statements {
		if dcl, isDeclaration := stmt.Node.(*ast.DeclarationNode); isDeclaration {
			if value, isValueNode := dcl.Expr.(*ast.ValueNode); isValueNode && value.IsResolved() {
				continue
			}
		}
		newTpl.Statements = append(newTpl.Statements, stmt)
	}

	return newTpl, env, nil
}

func resolveHolesPass(tpl *Template, env *Env) (*Template, *Env, error) {
	tpl.visitHoles(func(h ast.WithHoles) {
		processed := h.ProcessHoles(env.Fillers)
		env.addToProcessedFillers(processed)
	})

	return tpl, env, nil
}

func resolveMissingHolesPass(tpl *Template, env *Env) (*Template, *Env, error) {
	uniqueHoles := make(map[string]struct{})
	tpl.visitHoles(func(h ast.WithHoles) {
		for _, v := range h.GetHoles() {
			uniqueHoles[v] = struct{}{}
		}
	})
	var sortedHoles []string
	for k := range uniqueHoles {
		sortedHoles = append(sortedHoles, k)
	}
	sort.Strings(sortedHoles)
	fillers := make(map[string]interface{})
	for _, k := range sortedHoles {
		if env.MissingHolesFunc != nil {
			actual := env.MissingHolesFunc(k)
			fillers[k] = actual
		}
	}

	tpl.visitHoles(func(h ast.WithHoles) {
		processed := h.ProcessHoles(fillers)
		env.addToProcessedFillers(processed)
	})

	return tpl, env, nil
}

func resolveAliasPass(tpl *Template, env *Env) (*Template, *Env, error) {
	var emptyResolv []string
	resolvAliasFunc := func(key, entity string, i interface{}) (string, bool) {
		if s, ok := i.(string); ok {
			if strings.HasPrefix(s, "@") {
				env.Log.ExtraVerbosef("alias: resolving %s for key %s", s, key)
				alias := strings.TrimPrefix(s, "@")
				if env.AliasFunc == nil {
					return "", false
				}
				actual := env.AliasFunc(entity, key, alias)
				if actual == "" {
					emptyResolv = append(emptyResolv, alias)
					return "", false
				} else {
					env.Log.ExtraVerbosef("alias: resolved '%s' to '%s' for key %s", alias, actual, key)
					return actual, true
				}
			}
		}
		return "", false
	}

	tpl.visitCommandNodes(func(cmd *ast.CommandNode) {
		for k, v := range cmd.Params {
			if resolved, ok := resolvAliasFunc(k, cmd.Entity, v); ok {
				cmd.Params[k] = resolved
				delete(cmd.Holes, k)
			}
		}
	})

	tpl.visitActionNodes(func(action *ast.ActionNode) {
		for k, v := range action.Params {
			if vv, ok := v.(ast.WithAlias); ok {
				resolvAliasFunc := func(alias string) string {
					actual := env.AliasFunc(action.Entity, k, alias)
					if actual == "" {
						emptyResolv = append(emptyResolv, alias)
						return ""
					} else {
						env.Log.ExtraVerbosef("alias: resolved '%s' to '%s' for key %s", alias, actual, k)
						return actual
					}
				}
				vv.ResolveAlias(resolvAliasFunc)
			}
		}
	})

	if len(emptyResolv) > 0 {
		return tpl, env, fmt.Errorf("cannot resolve aliases: %q. Maybe you need to update your local model with `awless sync` ?", emptyResolv)
	}

	return tpl, env, nil
}

func failOnUnresolvedHoles(tpl *Template, env *Env) (*Template, *Env, error) {
	var unresolved []string
	tpl.visitCommandNodes(func(cmd *ast.CommandNode) {
		for _, v := range cmd.Holes {
			unresolved = append(unresolved, v)
		}
	})

	if len(unresolved) > 0 {
		return tpl, env, fmt.Errorf("template contains unresolved holes: %v", unresolved)
	}

	return tpl, env, nil
}

func failOnUnresolvedAlias(tpl *Template, env *Env) (*Template, *Env, error) {
	var unresolved []string
	tpl.visitCommandNodes(func(cmd *ast.CommandNode) {
		for _, v := range cmd.Params {
			if s, ok := v.(string); ok && strings.HasPrefix(s, "@") {
				unresolved = append(unresolved, s)
			}
		}
	})

	if len(unresolved) > 0 {
		return tpl, env, fmt.Errorf("template contains unresolved alias: %v", unresolved)
	}

	return tpl, env, nil
}
