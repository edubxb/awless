package ast

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const endSymbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleScript
	ruleStatement
	ruleAction
	ruleEntity
	ruleDeclaration
	ruleValueExpr
	ruleCmdExpr
	ruleParams
	ruleParam
	ruleIdentifier
	ruleCompositeValue
	ruleListValue
	ruleNoRefValue
	ruleValue
	ruleCustomTypedValue
	ruleOtherParamValue
	ruleDoubleQuotedValue
	ruleSingleQuotedValue
	ruleCidrValue
	ruleIpValue
	ruleIntRangeValue
	ruleRefValue
	ruleAliasValue
	ruleHoleValue
	ruleComment
	ruleSingleQuote
	ruleDoubleQuote
	ruleWhiteSpacing
	ruleMustWhiteSpacing
	ruleEqual
	ruleBlankLine
	ruleWhitespace
	ruleEndOfLine
	ruleEndOfFile
	rulePegText
	ruleAction0
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6
	ruleAction7
	ruleAction8
	ruleAction9
	ruleAction10
	ruleAction11
	ruleAction12
	ruleAction13
	ruleAction14
	ruleAction15
	ruleAction16
	ruleAction17
	ruleAction18
	ruleAction19
)

var rul3s = [...]string{
	"Unknown",
	"Script",
	"Statement",
	"Action",
	"Entity",
	"Declaration",
	"ValueExpr",
	"CmdExpr",
	"Params",
	"Param",
	"Identifier",
	"CompositeValue",
	"ListValue",
	"NoRefValue",
	"Value",
	"CustomTypedValue",
	"OtherParamValue",
	"DoubleQuotedValue",
	"SingleQuotedValue",
	"CidrValue",
	"IpValue",
	"IntRangeValue",
	"RefValue",
	"AliasValue",
	"HoleValue",
	"Comment",
	"SingleQuote",
	"DoubleQuote",
	"WhiteSpacing",
	"MustWhiteSpacing",
	"Equal",
	"BlankLine",
	"Whitespace",
	"EndOfLine",
	"EndOfFile",
	"PegText",
	"Action0",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",
	"Action6",
	"Action7",
	"Action8",
	"Action9",
	"Action10",
	"Action11",
	"Action12",
	"Action13",
	"Action14",
	"Action15",
	"Action16",
	"Action17",
	"Action18",
	"Action19",
}

type token32 struct {
	pegRule
	begin, end uint32
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v", rul3s[t.pegRule], t.begin, t.end)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(pretty bool, buffer string) {
	var print func(node *node32, depth int)
	print = func(node *node32, depth int) {
		for node != nil {
			for c := 0; c < depth; c++ {
				fmt.Printf(" ")
			}
			rule := rul3s[node.pegRule]
			quote := strconv.Quote(string(([]rune(buffer)[node.begin:node.end])))
			if !pretty {
				fmt.Printf("%v %v\n", rule, quote)
			} else {
				fmt.Printf("\x1B[34m%v\x1B[m %v\n", rule, quote)
			}
			if node.up != nil {
				print(node.up, depth+1)
			}
			node = node.next
		}
	}
	print(node, 0)
}

func (node *node32) Print(buffer string) {
	node.print(false, buffer)
}

func (node *node32) PrettyPrint(buffer string) {
	node.print(true, buffer)
}

type tokens32 struct {
	tree []token32
}

func (t *tokens32) Trim(length uint32) {
	t.tree = t.tree[:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) AST() *node32 {
	type element struct {
		node *node32
		down *element
	}
	tokens := t.Tokens()
	var stack *element
	for _, token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	if stack != nil {
		return stack.node
	}
	return nil
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	t.AST().Print(buffer)
}

func (t *tokens32) PrettyPrintSyntaxTree(buffer string) {
	t.AST().PrettyPrint(buffer)
}

func (t *tokens32) Add(rule pegRule, begin, end, index uint32) {
	if tree := t.tree; int(index) >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	t.tree[index] = token32{
		pegRule: rule,
		begin:   begin,
		end:     end,
	}
}

func (t *tokens32) Tokens() []token32 {
	return t.tree
}

type Peg struct {
	*AST

	Buffer string
	buffer []rune
	rules  [56]func() bool
	parse  func(rule ...int) error
	reset  func()
	Pretty bool
	tokens32
}

func (p *Peg) Parse(rule ...int) error {
	return p.parse(rule...)
}

func (p *Peg) Reset() {
	p.reset()
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer []rune, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p   *Peg
	max token32
}

func (e *parseError) Error() string {
	tokens, error := []token32{e.max}, "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.buffer, positions)
	format := "parse error near %v (line %v symbol %v - line %v symbol %v):\n%v\n"
	if e.p.Pretty {
		format = "parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n"
	}
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf(format,
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			strconv.Quote(string(e.p.buffer[begin:end])))
	}

	return error
}

func (p *Peg) PrintSyntaxTree() {
	if p.Pretty {
		p.tokens32.PrettyPrintSyntaxTree(p.Buffer)
	} else {
		p.tokens32.PrintSyntaxTree(p.Buffer)
	}
}

func (p *Peg) Execute() {
	buffer, _buffer, text, begin, end := p.Buffer, p.buffer, "", 0, 0
	for _, token := range p.Tokens() {
		switch token.pegRule {

		case rulePegText:
			begin, end = int(token.begin), int(token.end)
			text = string(_buffer[begin:end])

		case ruleAction0:
			p.addDeclarationIdentifier(text)
		case ruleAction1:
			p.addValue()
		case ruleAction2:
			p.LineDone()
		case ruleAction3:
			p.addAction(text)
		case ruleAction4:
			p.addEntity(text)
		case ruleAction5:
			p.LineDone()
		case ruleAction6:
			p.addParamKey(text)
		case ruleAction7:
			p.addFirstValueInList()
		case ruleAction8:
			p.lastValueInList()
		case ruleAction9:
			p.addParamHoleValue(text)
		case ruleAction10:
			p.addAliasParam(text)
		case ruleAction11:
			p.addStringValue(text)
		case ruleAction12:
			p.addStringValue(text)
		case ruleAction13:
			p.addParamValue(text)
		case ruleAction14:
			p.addParamRefValue(text)
		case ruleAction15:
			p.addParamCidrValue(text)
		case ruleAction16:
			p.addParamIpValue(text)
		case ruleAction17:
			p.addParamValue(text)
		case ruleAction18:
			p.LineDone()
		case ruleAction19:
			p.LineDone()

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}

func (p *Peg) Init() {
	var (
		max                  token32
		position, tokenIndex uint32
		buffer               []rune
	)
	p.reset = func() {
		max = token32{}
		position, tokenIndex = 0, 0

		p.buffer = []rune(p.Buffer)
		if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != endSymbol {
			p.buffer = append(p.buffer, endSymbol)
		}
		buffer = p.buffer
	}
	p.reset()

	_rules := p.rules
	tree := tokens32{tree: make([]token32, math.MaxInt16)}
	p.parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokens32 = tree
		if matches {
			p.Trim(tokenIndex)
			return nil
		}
		return &parseError{p, max}
	}

	add := func(rule pegRule, begin uint32) {
		tree.Add(rule, begin, position, tokenIndex)
		tokenIndex++
		if begin != position && position > max.end {
			max = token32{rule, begin, position}
		}
	}

	matchDot := func() bool {
		if buffer[position] != endSymbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 Script <- <((BlankLine* Statement BlankLine*)+ WhiteSpacing EndOfFile)> */
		func() bool {
			position0, tokenIndex0 := position, tokenIndex
			{
				position1 := position
			l4:
				{
					position5, tokenIndex5 := position, tokenIndex
					if !_rules[ruleBlankLine]() {
						goto l5
					}
					goto l4
				l5:
					position, tokenIndex = position5, tokenIndex5
				}
				{
					position6 := position
					if !_rules[ruleWhiteSpacing]() {
						goto l0
					}
					{
						position7, tokenIndex7 := position, tokenIndex
						if !_rules[ruleCmdExpr]() {
							goto l8
						}
						goto l7
					l8:
						position, tokenIndex = position7, tokenIndex7
						{
							position10 := position
							{
								position11 := position
								if !_rules[ruleIdentifier]() {
									goto l9
								}
								add(rulePegText, position11)
							}
							{
								add(ruleAction0, position)
							}
							if !_rules[ruleEqual]() {
								goto l9
							}
							{
								position13, tokenIndex13 := position, tokenIndex
								if !_rules[ruleCmdExpr]() {
									goto l14
								}
								goto l13
							l14:
								position, tokenIndex = position13, tokenIndex13
								{
									position15 := position
									{
										add(ruleAction1, position)
									}
									if !_rules[ruleCompositeValue]() {
										goto l9
									}
									{
										add(ruleAction2, position)
									}
									add(ruleValueExpr, position15)
								}
							}
						l13:
							add(ruleDeclaration, position10)
						}
						goto l7
					l9:
						position, tokenIndex = position7, tokenIndex7
						{
							position18 := position
							{
								position19, tokenIndex19 := position, tokenIndex
								if buffer[position] != rune('#') {
									goto l20
								}
								position++
							l21:
								{
									position22, tokenIndex22 := position, tokenIndex
									{
										position23, tokenIndex23 := position, tokenIndex
										if !_rules[ruleEndOfLine]() {
											goto l23
										}
										goto l22
									l23:
										position, tokenIndex = position23, tokenIndex23
									}
									if !matchDot() {
										goto l22
									}
									goto l21
								l22:
									position, tokenIndex = position22, tokenIndex22
								}
								goto l19
							l20:
								position, tokenIndex = position19, tokenIndex19
								if buffer[position] != rune('/') {
									goto l0
								}
								position++
								if buffer[position] != rune('/') {
									goto l0
								}
								position++
							l24:
								{
									position25, tokenIndex25 := position, tokenIndex
									{
										position26, tokenIndex26 := position, tokenIndex
										if !_rules[ruleEndOfLine]() {
											goto l26
										}
										goto l25
									l26:
										position, tokenIndex = position26, tokenIndex26
									}
									if !matchDot() {
										goto l25
									}
									goto l24
								l25:
									position, tokenIndex = position25, tokenIndex25
								}
								{
									add(ruleAction18, position)
								}
							}
						l19:
							add(ruleComment, position18)
						}
					}
				l7:
					if !_rules[ruleWhiteSpacing]() {
						goto l0
					}
				l28:
					{
						position29, tokenIndex29 := position, tokenIndex
						if !_rules[ruleEndOfLine]() {
							goto l29
						}
						goto l28
					l29:
						position, tokenIndex = position29, tokenIndex29
					}
					add(ruleStatement, position6)
				}
			l30:
				{
					position31, tokenIndex31 := position, tokenIndex
					if !_rules[ruleBlankLine]() {
						goto l31
					}
					goto l30
				l31:
					position, tokenIndex = position31, tokenIndex31
				}
			l2:
				{
					position3, tokenIndex3 := position, tokenIndex
				l32:
					{
						position33, tokenIndex33 := position, tokenIndex
						if !_rules[ruleBlankLine]() {
							goto l33
						}
						goto l32
					l33:
						position, tokenIndex = position33, tokenIndex33
					}
					{
						position34 := position
						if !_rules[ruleWhiteSpacing]() {
							goto l3
						}
						{
							position35, tokenIndex35 := position, tokenIndex
							if !_rules[ruleCmdExpr]() {
								goto l36
							}
							goto l35
						l36:
							position, tokenIndex = position35, tokenIndex35
							{
								position38 := position
								{
									position39 := position
									if !_rules[ruleIdentifier]() {
										goto l37
									}
									add(rulePegText, position39)
								}
								{
									add(ruleAction0, position)
								}
								if !_rules[ruleEqual]() {
									goto l37
								}
								{
									position41, tokenIndex41 := position, tokenIndex
									if !_rules[ruleCmdExpr]() {
										goto l42
									}
									goto l41
								l42:
									position, tokenIndex = position41, tokenIndex41
									{
										position43 := position
										{
											add(ruleAction1, position)
										}
										if !_rules[ruleCompositeValue]() {
											goto l37
										}
										{
											add(ruleAction2, position)
										}
										add(ruleValueExpr, position43)
									}
								}
							l41:
								add(ruleDeclaration, position38)
							}
							goto l35
						l37:
							position, tokenIndex = position35, tokenIndex35
							{
								position46 := position
								{
									position47, tokenIndex47 := position, tokenIndex
									if buffer[position] != rune('#') {
										goto l48
									}
									position++
								l49:
									{
										position50, tokenIndex50 := position, tokenIndex
										{
											position51, tokenIndex51 := position, tokenIndex
											if !_rules[ruleEndOfLine]() {
												goto l51
											}
											goto l50
										l51:
											position, tokenIndex = position51, tokenIndex51
										}
										if !matchDot() {
											goto l50
										}
										goto l49
									l50:
										position, tokenIndex = position50, tokenIndex50
									}
									goto l47
								l48:
									position, tokenIndex = position47, tokenIndex47
									if buffer[position] != rune('/') {
										goto l3
									}
									position++
									if buffer[position] != rune('/') {
										goto l3
									}
									position++
								l52:
									{
										position53, tokenIndex53 := position, tokenIndex
										{
											position54, tokenIndex54 := position, tokenIndex
											if !_rules[ruleEndOfLine]() {
												goto l54
											}
											goto l53
										l54:
											position, tokenIndex = position54, tokenIndex54
										}
										if !matchDot() {
											goto l53
										}
										goto l52
									l53:
										position, tokenIndex = position53, tokenIndex53
									}
									{
										add(ruleAction18, position)
									}
								}
							l47:
								add(ruleComment, position46)
							}
						}
					l35:
						if !_rules[ruleWhiteSpacing]() {
							goto l3
						}
					l56:
						{
							position57, tokenIndex57 := position, tokenIndex
							if !_rules[ruleEndOfLine]() {
								goto l57
							}
							goto l56
						l57:
							position, tokenIndex = position57, tokenIndex57
						}
						add(ruleStatement, position34)
					}
				l58:
					{
						position59, tokenIndex59 := position, tokenIndex
						if !_rules[ruleBlankLine]() {
							goto l59
						}
						goto l58
					l59:
						position, tokenIndex = position59, tokenIndex59
					}
					goto l2
				l3:
					position, tokenIndex = position3, tokenIndex3
				}
				if !_rules[ruleWhiteSpacing]() {
					goto l0
				}
				{
					position60 := position
					{
						position61, tokenIndex61 := position, tokenIndex
						if !matchDot() {
							goto l61
						}
						goto l0
					l61:
						position, tokenIndex = position61, tokenIndex61
					}
					add(ruleEndOfFile, position60)
				}
				add(ruleScript, position1)
			}
			return true
		l0:
			position, tokenIndex = position0, tokenIndex0
			return false
		},
		/* 1 Statement <- <(WhiteSpacing (CmdExpr / Declaration / Comment) WhiteSpacing EndOfLine*)> */
		nil,
		/* 2 Action <- <[a-z]+> */
		nil,
		/* 3 Entity <- <([a-z] / [0-9])+> */
		nil,
		/* 4 Declaration <- <(<Identifier> Action0 Equal (CmdExpr / ValueExpr))> */
		nil,
		/* 5 ValueExpr <- <(Action1 CompositeValue Action2)> */
		nil,
		/* 6 CmdExpr <- <(<Action> Action3 MustWhiteSpacing <Entity> Action4 (MustWhiteSpacing Params)? Action5)> */
		func() bool {
			position67, tokenIndex67 := position, tokenIndex
			{
				position68 := position
				{
					position69 := position
					{
						position70 := position
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l67
						}
						position++
					l71:
						{
							position72, tokenIndex72 := position, tokenIndex
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l72
							}
							position++
							goto l71
						l72:
							position, tokenIndex = position72, tokenIndex72
						}
						add(ruleAction, position70)
					}
					add(rulePegText, position69)
				}
				{
					add(ruleAction3, position)
				}
				if !_rules[ruleMustWhiteSpacing]() {
					goto l67
				}
				{
					position74 := position
					{
						position75 := position
						{
							position78, tokenIndex78 := position, tokenIndex
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l79
							}
							position++
							goto l78
						l79:
							position, tokenIndex = position78, tokenIndex78
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l67
							}
							position++
						}
					l78:
					l76:
						{
							position77, tokenIndex77 := position, tokenIndex
							{
								position80, tokenIndex80 := position, tokenIndex
								if c := buffer[position]; c < rune('a') || c > rune('z') {
									goto l81
								}
								position++
								goto l80
							l81:
								position, tokenIndex = position80, tokenIndex80
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l77
								}
								position++
							}
						l80:
							goto l76
						l77:
							position, tokenIndex = position77, tokenIndex77
						}
						add(ruleEntity, position75)
					}
					add(rulePegText, position74)
				}
				{
					add(ruleAction4, position)
				}
				{
					position83, tokenIndex83 := position, tokenIndex
					if !_rules[ruleMustWhiteSpacing]() {
						goto l83
					}
					{
						position85 := position
						{
							position88 := position
							{
								position89 := position
								if !_rules[ruleIdentifier]() {
									goto l83
								}
								add(rulePegText, position89)
							}
							{
								add(ruleAction6, position)
							}
							if !_rules[ruleEqual]() {
								goto l83
							}
							if !_rules[ruleCompositeValue]() {
								goto l83
							}
							if !_rules[ruleWhiteSpacing]() {
								goto l83
							}
							add(ruleParam, position88)
						}
					l86:
						{
							position87, tokenIndex87 := position, tokenIndex
							{
								position91 := position
								{
									position92 := position
									if !_rules[ruleIdentifier]() {
										goto l87
									}
									add(rulePegText, position92)
								}
								{
									add(ruleAction6, position)
								}
								if !_rules[ruleEqual]() {
									goto l87
								}
								if !_rules[ruleCompositeValue]() {
									goto l87
								}
								if !_rules[ruleWhiteSpacing]() {
									goto l87
								}
								add(ruleParam, position91)
							}
							goto l86
						l87:
							position, tokenIndex = position87, tokenIndex87
						}
						add(ruleParams, position85)
					}
					goto l84
				l83:
					position, tokenIndex = position83, tokenIndex83
				}
			l84:
				{
					add(ruleAction5, position)
				}
				add(ruleCmdExpr, position68)
			}
			return true
		l67:
			position, tokenIndex = position67, tokenIndex67
			return false
		},
		/* 7 Params <- <Param+> */
		nil,
		/* 8 Param <- <(<Identifier> Action6 Equal CompositeValue WhiteSpacing)> */
		nil,
		/* 9 Identifier <- <((&('.') '.') | (&('_') '_') | (&('-') '-') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+> */
		func() bool {
			position97, tokenIndex97 := position, tokenIndex
			{
				position98 := position
				{
					switch buffer[position] {
					case '.':
						if buffer[position] != rune('.') {
							goto l97
						}
						position++
						break
					case '_':
						if buffer[position] != rune('_') {
							goto l97
						}
						position++
						break
					case '-':
						if buffer[position] != rune('-') {
							goto l97
						}
						position++
						break
					case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l97
						}
						position++
						break
					case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l97
						}
						position++
						break
					default:
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l97
						}
						position++
						break
					}
				}

			l99:
				{
					position100, tokenIndex100 := position, tokenIndex
					{
						switch buffer[position] {
						case '.':
							if buffer[position] != rune('.') {
								goto l100
							}
							position++
							break
						case '_':
							if buffer[position] != rune('_') {
								goto l100
							}
							position++
							break
						case '-':
							if buffer[position] != rune('-') {
								goto l100
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l100
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l100
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l100
							}
							position++
							break
						}
					}

					goto l99
				l100:
					position, tokenIndex = position100, tokenIndex100
				}
				add(ruleIdentifier, position98)
			}
			return true
		l97:
			position, tokenIndex = position97, tokenIndex97
			return false
		},
		/* 10 CompositeValue <- <(ListValue / Value)> */
		func() bool {
			position103, tokenIndex103 := position, tokenIndex
			{
				position104 := position
				{
					position105, tokenIndex105 := position, tokenIndex
					{
						position107 := position
						{
							add(ruleAction7, position)
						}
						if buffer[position] != rune('[') {
							goto l106
						}
						position++
						if !_rules[ruleWhiteSpacing]() {
							goto l106
						}
						if !_rules[ruleValue]() {
							goto l106
						}
						if !_rules[ruleWhiteSpacing]() {
							goto l106
						}
					l109:
						{
							position110, tokenIndex110 := position, tokenIndex
							if buffer[position] != rune(',') {
								goto l110
							}
							position++
							if !_rules[ruleWhiteSpacing]() {
								goto l110
							}
							if !_rules[ruleValue]() {
								goto l110
							}
							if !_rules[ruleWhiteSpacing]() {
								goto l110
							}
							goto l109
						l110:
							position, tokenIndex = position110, tokenIndex110
						}
						if buffer[position] != rune(']') {
							goto l106
						}
						position++
						{
							add(ruleAction8, position)
						}
						add(ruleListValue, position107)
					}
					goto l105
				l106:
					position, tokenIndex = position105, tokenIndex105
					if !_rules[ruleValue]() {
						goto l103
					}
				}
			l105:
				add(ruleCompositeValue, position104)
			}
			return true
		l103:
			position, tokenIndex = position103, tokenIndex103
			return false
		},
		/* 11 ListValue <- <(Action7 '[' WhiteSpacing Value WhiteSpacing (',' WhiteSpacing Value WhiteSpacing)* ']' Action8)> */
		nil,
		/* 12 NoRefValue <- <((AliasValue Action10) / (DoubleQuote CustomTypedValue DoubleQuote) / (SingleQuote CustomTypedValue SingleQuote) / CustomTypedValue / ((&('\'') (SingleQuote <SingleQuotedValue> Action12 SingleQuote)) | (&('"') (DoubleQuote <DoubleQuotedValue> Action11 DoubleQuote)) | (&('{') (HoleValue Action9)) | (&('*' | '+' | '-' | '.' | '/' | '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | ':' | ';' | '<' | '>' | '@' | 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z' | '_' | 'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z' | '~') (<OtherParamValue> Action13))))> */
		nil,
		/* 13 Value <- <((RefValue Action14) / NoRefValue)> */
		func() bool {
			position114, tokenIndex114 := position, tokenIndex
			{
				position115 := position
				{
					position116, tokenIndex116 := position, tokenIndex
					{
						position118 := position
						if buffer[position] != rune('$') {
							goto l117
						}
						position++
						{
							position119 := position
							if !_rules[ruleIdentifier]() {
								goto l117
							}
							add(rulePegText, position119)
						}
						add(ruleRefValue, position118)
					}
					{
						add(ruleAction14, position)
					}
					goto l116
				l117:
					position, tokenIndex = position116, tokenIndex116
					{
						position121 := position
						{
							position122, tokenIndex122 := position, tokenIndex
							{
								position124 := position
								{
									position125, tokenIndex125 := position, tokenIndex
									if buffer[position] != rune('@') {
										goto l126
									}
									position++
									{
										position127 := position
										if !_rules[ruleOtherParamValue]() {
											goto l126
										}
										add(rulePegText, position127)
									}
									goto l125
								l126:
									position, tokenIndex = position125, tokenIndex125
									if buffer[position] != rune('@') {
										goto l128
									}
									position++
									if !_rules[ruleDoubleQuote]() {
										goto l128
									}
									{
										position129 := position
										if !_rules[ruleDoubleQuotedValue]() {
											goto l128
										}
										add(rulePegText, position129)
									}
									if !_rules[ruleDoubleQuote]() {
										goto l128
									}
									goto l125
								l128:
									position, tokenIndex = position125, tokenIndex125
									if buffer[position] != rune('@') {
										goto l123
									}
									position++
									if !_rules[ruleSingleQuote]() {
										goto l123
									}
									{
										position130 := position
										if !_rules[ruleSingleQuotedValue]() {
											goto l123
										}
										add(rulePegText, position130)
									}
									if !_rules[ruleSingleQuote]() {
										goto l123
									}
								}
							l125:
								add(ruleAliasValue, position124)
							}
							{
								add(ruleAction10, position)
							}
							goto l122
						l123:
							position, tokenIndex = position122, tokenIndex122
							if !_rules[ruleDoubleQuote]() {
								goto l132
							}
							if !_rules[ruleCustomTypedValue]() {
								goto l132
							}
							if !_rules[ruleDoubleQuote]() {
								goto l132
							}
							goto l122
						l132:
							position, tokenIndex = position122, tokenIndex122
							if !_rules[ruleSingleQuote]() {
								goto l133
							}
							if !_rules[ruleCustomTypedValue]() {
								goto l133
							}
							if !_rules[ruleSingleQuote]() {
								goto l133
							}
							goto l122
						l133:
							position, tokenIndex = position122, tokenIndex122
							if !_rules[ruleCustomTypedValue]() {
								goto l134
							}
							goto l122
						l134:
							position, tokenIndex = position122, tokenIndex122
							{
								switch buffer[position] {
								case '\'':
									if !_rules[ruleSingleQuote]() {
										goto l114
									}
									{
										position136 := position
										if !_rules[ruleSingleQuotedValue]() {
											goto l114
										}
										add(rulePegText, position136)
									}
									{
										add(ruleAction12, position)
									}
									if !_rules[ruleSingleQuote]() {
										goto l114
									}
									break
								case '"':
									if !_rules[ruleDoubleQuote]() {
										goto l114
									}
									{
										position138 := position
										if !_rules[ruleDoubleQuotedValue]() {
											goto l114
										}
										add(rulePegText, position138)
									}
									{
										add(ruleAction11, position)
									}
									if !_rules[ruleDoubleQuote]() {
										goto l114
									}
									break
								case '{':
									{
										position140 := position
										if buffer[position] != rune('{') {
											goto l114
										}
										position++
										if !_rules[ruleWhiteSpacing]() {
											goto l114
										}
										{
											position141 := position
											if !_rules[ruleIdentifier]() {
												goto l114
											}
											add(rulePegText, position141)
										}
										if !_rules[ruleWhiteSpacing]() {
											goto l114
										}
										if buffer[position] != rune('}') {
											goto l114
										}
										position++
										add(ruleHoleValue, position140)
									}
									{
										add(ruleAction9, position)
									}
									break
								default:
									{
										position143 := position
										if !_rules[ruleOtherParamValue]() {
											goto l114
										}
										add(rulePegText, position143)
									}
									{
										add(ruleAction13, position)
									}
									break
								}
							}

						}
					l122:
						add(ruleNoRefValue, position121)
					}
				}
			l116:
				add(ruleValue, position115)
			}
			return true
		l114:
			position, tokenIndex = position114, tokenIndex114
			return false
		},
		/* 14 CustomTypedValue <- <((<CidrValue> Action15) / (<IpValue> Action16) / (<IntRangeValue> Action17))> */
		func() bool {
			position145, tokenIndex145 := position, tokenIndex
			{
				position146 := position
				{
					position147, tokenIndex147 := position, tokenIndex
					{
						position149 := position
						{
							position150 := position
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l148
							}
							position++
						l151:
							{
								position152, tokenIndex152 := position, tokenIndex
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l152
								}
								position++
								goto l151
							l152:
								position, tokenIndex = position152, tokenIndex152
							}
							if buffer[position] != rune('.') {
								goto l148
							}
							position++
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l148
							}
							position++
						l153:
							{
								position154, tokenIndex154 := position, tokenIndex
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l154
								}
								position++
								goto l153
							l154:
								position, tokenIndex = position154, tokenIndex154
							}
							if buffer[position] != rune('.') {
								goto l148
							}
							position++
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l148
							}
							position++
						l155:
							{
								position156, tokenIndex156 := position, tokenIndex
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l156
								}
								position++
								goto l155
							l156:
								position, tokenIndex = position156, tokenIndex156
							}
							if buffer[position] != rune('.') {
								goto l148
							}
							position++
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l148
							}
							position++
						l157:
							{
								position158, tokenIndex158 := position, tokenIndex
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l158
								}
								position++
								goto l157
							l158:
								position, tokenIndex = position158, tokenIndex158
							}
							if buffer[position] != rune('/') {
								goto l148
							}
							position++
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l148
							}
							position++
						l159:
							{
								position160, tokenIndex160 := position, tokenIndex
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l160
								}
								position++
								goto l159
							l160:
								position, tokenIndex = position160, tokenIndex160
							}
							add(ruleCidrValue, position150)
						}
						add(rulePegText, position149)
					}
					{
						add(ruleAction15, position)
					}
					goto l147
				l148:
					position, tokenIndex = position147, tokenIndex147
					{
						position163 := position
						{
							position164 := position
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l162
							}
							position++
						l165:
							{
								position166, tokenIndex166 := position, tokenIndex
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l166
								}
								position++
								goto l165
							l166:
								position, tokenIndex = position166, tokenIndex166
							}
							if buffer[position] != rune('.') {
								goto l162
							}
							position++
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l162
							}
							position++
						l167:
							{
								position168, tokenIndex168 := position, tokenIndex
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l168
								}
								position++
								goto l167
							l168:
								position, tokenIndex = position168, tokenIndex168
							}
							if buffer[position] != rune('.') {
								goto l162
							}
							position++
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l162
							}
							position++
						l169:
							{
								position170, tokenIndex170 := position, tokenIndex
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l170
								}
								position++
								goto l169
							l170:
								position, tokenIndex = position170, tokenIndex170
							}
							if buffer[position] != rune('.') {
								goto l162
							}
							position++
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l162
							}
							position++
						l171:
							{
								position172, tokenIndex172 := position, tokenIndex
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l172
								}
								position++
								goto l171
							l172:
								position, tokenIndex = position172, tokenIndex172
							}
							add(ruleIpValue, position164)
						}
						add(rulePegText, position163)
					}
					{
						add(ruleAction16, position)
					}
					goto l147
				l162:
					position, tokenIndex = position147, tokenIndex147
					{
						position174 := position
						{
							position175 := position
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l145
							}
							position++
						l176:
							{
								position177, tokenIndex177 := position, tokenIndex
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l177
								}
								position++
								goto l176
							l177:
								position, tokenIndex = position177, tokenIndex177
							}
							if buffer[position] != rune('-') {
								goto l145
							}
							position++
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l145
							}
							position++
						l178:
							{
								position179, tokenIndex179 := position, tokenIndex
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l179
								}
								position++
								goto l178
							l179:
								position, tokenIndex = position179, tokenIndex179
							}
							add(ruleIntRangeValue, position175)
						}
						add(rulePegText, position174)
					}
					{
						add(ruleAction17, position)
					}
				}
			l147:
				add(ruleCustomTypedValue, position146)
			}
			return true
		l145:
			position, tokenIndex = position145, tokenIndex145
			return false
		},
		/* 15 OtherParamValue <- <((&('*') '*') | (&('>') '>') | (&('<') '<') | (&('@') '@') | (&('~') '~') | (&(';') ';') | (&('+') '+') | (&('/') '/') | (&(':') ':') | (&('_') '_') | (&('.') '.') | (&('-') '-') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+> */
		func() bool {
			position181, tokenIndex181 := position, tokenIndex
			{
				position182 := position
				{
					switch buffer[position] {
					case '*':
						if buffer[position] != rune('*') {
							goto l181
						}
						position++
						break
					case '>':
						if buffer[position] != rune('>') {
							goto l181
						}
						position++
						break
					case '<':
						if buffer[position] != rune('<') {
							goto l181
						}
						position++
						break
					case '@':
						if buffer[position] != rune('@') {
							goto l181
						}
						position++
						break
					case '~':
						if buffer[position] != rune('~') {
							goto l181
						}
						position++
						break
					case ';':
						if buffer[position] != rune(';') {
							goto l181
						}
						position++
						break
					case '+':
						if buffer[position] != rune('+') {
							goto l181
						}
						position++
						break
					case '/':
						if buffer[position] != rune('/') {
							goto l181
						}
						position++
						break
					case ':':
						if buffer[position] != rune(':') {
							goto l181
						}
						position++
						break
					case '_':
						if buffer[position] != rune('_') {
							goto l181
						}
						position++
						break
					case '.':
						if buffer[position] != rune('.') {
							goto l181
						}
						position++
						break
					case '-':
						if buffer[position] != rune('-') {
							goto l181
						}
						position++
						break
					case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l181
						}
						position++
						break
					case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l181
						}
						position++
						break
					default:
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l181
						}
						position++
						break
					}
				}

			l183:
				{
					position184, tokenIndex184 := position, tokenIndex
					{
						switch buffer[position] {
						case '*':
							if buffer[position] != rune('*') {
								goto l184
							}
							position++
							break
						case '>':
							if buffer[position] != rune('>') {
								goto l184
							}
							position++
							break
						case '<':
							if buffer[position] != rune('<') {
								goto l184
							}
							position++
							break
						case '@':
							if buffer[position] != rune('@') {
								goto l184
							}
							position++
							break
						case '~':
							if buffer[position] != rune('~') {
								goto l184
							}
							position++
							break
						case ';':
							if buffer[position] != rune(';') {
								goto l184
							}
							position++
							break
						case '+':
							if buffer[position] != rune('+') {
								goto l184
							}
							position++
							break
						case '/':
							if buffer[position] != rune('/') {
								goto l184
							}
							position++
							break
						case ':':
							if buffer[position] != rune(':') {
								goto l184
							}
							position++
							break
						case '_':
							if buffer[position] != rune('_') {
								goto l184
							}
							position++
							break
						case '.':
							if buffer[position] != rune('.') {
								goto l184
							}
							position++
							break
						case '-':
							if buffer[position] != rune('-') {
								goto l184
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l184
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l184
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l184
							}
							position++
							break
						}
					}

					goto l183
				l184:
					position, tokenIndex = position184, tokenIndex184
				}
				add(ruleOtherParamValue, position182)
			}
			return true
		l181:
			position, tokenIndex = position181, tokenIndex181
			return false
		},
		/* 16 DoubleQuotedValue <- <(!'"' .)*> */
		func() bool {
			{
				position188 := position
			l189:
				{
					position190, tokenIndex190 := position, tokenIndex
					{
						position191, tokenIndex191 := position, tokenIndex
						if buffer[position] != rune('"') {
							goto l191
						}
						position++
						goto l190
					l191:
						position, tokenIndex = position191, tokenIndex191
					}
					if !matchDot() {
						goto l190
					}
					goto l189
				l190:
					position, tokenIndex = position190, tokenIndex190
				}
				add(ruleDoubleQuotedValue, position188)
			}
			return true
		},
		/* 17 SingleQuotedValue <- <(!'\'' .)*> */
		func() bool {
			{
				position193 := position
			l194:
				{
					position195, tokenIndex195 := position, tokenIndex
					{
						position196, tokenIndex196 := position, tokenIndex
						if buffer[position] != rune('\'') {
							goto l196
						}
						position++
						goto l195
					l196:
						position, tokenIndex = position196, tokenIndex196
					}
					if !matchDot() {
						goto l195
					}
					goto l194
				l195:
					position, tokenIndex = position195, tokenIndex195
				}
				add(ruleSingleQuotedValue, position193)
			}
			return true
		},
		/* 18 CidrValue <- <([0-9]+ '.' [0-9]+ '.' [0-9]+ '.' [0-9]+ '/' [0-9]+)> */
		nil,
		/* 19 IpValue <- <([0-9]+ '.' [0-9]+ '.' [0-9]+ '.' [0-9]+)> */
		nil,
		/* 20 IntRangeValue <- <([0-9]+ '-' [0-9]+)> */
		nil,
		/* 21 RefValue <- <('$' <Identifier>)> */
		nil,
		/* 22 AliasValue <- <(('@' <OtherParamValue>) / ('@' DoubleQuote <DoubleQuotedValue> DoubleQuote) / ('@' SingleQuote <SingleQuotedValue> SingleQuote))> */
		nil,
		/* 23 HoleValue <- <('{' WhiteSpacing <Identifier> WhiteSpacing '}')> */
		nil,
		/* 24 Comment <- <(('#' (!EndOfLine .)*) / ('/' '/' (!EndOfLine .)* Action18))> */
		nil,
		/* 25 SingleQuote <- <'\''> */
		func() bool {
			position204, tokenIndex204 := position, tokenIndex
			{
				position205 := position
				if buffer[position] != rune('\'') {
					goto l204
				}
				position++
				add(ruleSingleQuote, position205)
			}
			return true
		l204:
			position, tokenIndex = position204, tokenIndex204
			return false
		},
		/* 26 DoubleQuote <- <'"'> */
		func() bool {
			position206, tokenIndex206 := position, tokenIndex
			{
				position207 := position
				if buffer[position] != rune('"') {
					goto l206
				}
				position++
				add(ruleDoubleQuote, position207)
			}
			return true
		l206:
			position, tokenIndex = position206, tokenIndex206
			return false
		},
		/* 27 WhiteSpacing <- <Whitespace*> */
		func() bool {
			{
				position209 := position
			l210:
				{
					position211, tokenIndex211 := position, tokenIndex
					if !_rules[ruleWhitespace]() {
						goto l211
					}
					goto l210
				l211:
					position, tokenIndex = position211, tokenIndex211
				}
				add(ruleWhiteSpacing, position209)
			}
			return true
		},
		/* 28 MustWhiteSpacing <- <Whitespace+> */
		func() bool {
			position212, tokenIndex212 := position, tokenIndex
			{
				position213 := position
				if !_rules[ruleWhitespace]() {
					goto l212
				}
			l214:
				{
					position215, tokenIndex215 := position, tokenIndex
					if !_rules[ruleWhitespace]() {
						goto l215
					}
					goto l214
				l215:
					position, tokenIndex = position215, tokenIndex215
				}
				add(ruleMustWhiteSpacing, position213)
			}
			return true
		l212:
			position, tokenIndex = position212, tokenIndex212
			return false
		},
		/* 29 Equal <- <(WhiteSpacing '=' WhiteSpacing)> */
		func() bool {
			position216, tokenIndex216 := position, tokenIndex
			{
				position217 := position
				if !_rules[ruleWhiteSpacing]() {
					goto l216
				}
				if buffer[position] != rune('=') {
					goto l216
				}
				position++
				if !_rules[ruleWhiteSpacing]() {
					goto l216
				}
				add(ruleEqual, position217)
			}
			return true
		l216:
			position, tokenIndex = position216, tokenIndex216
			return false
		},
		/* 30 BlankLine <- <(WhiteSpacing EndOfLine Action19)> */
		func() bool {
			position218, tokenIndex218 := position, tokenIndex
			{
				position219 := position
				if !_rules[ruleWhiteSpacing]() {
					goto l218
				}
				if !_rules[ruleEndOfLine]() {
					goto l218
				}
				{
					add(ruleAction19, position)
				}
				add(ruleBlankLine, position219)
			}
			return true
		l218:
			position, tokenIndex = position218, tokenIndex218
			return false
		},
		/* 31 Whitespace <- <(' ' / '\t')> */
		func() bool {
			position221, tokenIndex221 := position, tokenIndex
			{
				position222 := position
				{
					position223, tokenIndex223 := position, tokenIndex
					if buffer[position] != rune(' ') {
						goto l224
					}
					position++
					goto l223
				l224:
					position, tokenIndex = position223, tokenIndex223
					if buffer[position] != rune('\t') {
						goto l221
					}
					position++
				}
			l223:
				add(ruleWhitespace, position222)
			}
			return true
		l221:
			position, tokenIndex = position221, tokenIndex221
			return false
		},
		/* 32 EndOfLine <- <(('\r' '\n') / '\n' / '\r')> */
		func() bool {
			position225, tokenIndex225 := position, tokenIndex
			{
				position226 := position
				{
					position227, tokenIndex227 := position, tokenIndex
					if buffer[position] != rune('\r') {
						goto l228
					}
					position++
					if buffer[position] != rune('\n') {
						goto l228
					}
					position++
					goto l227
				l228:
					position, tokenIndex = position227, tokenIndex227
					if buffer[position] != rune('\n') {
						goto l229
					}
					position++
					goto l227
				l229:
					position, tokenIndex = position227, tokenIndex227
					if buffer[position] != rune('\r') {
						goto l225
					}
					position++
				}
			l227:
				add(ruleEndOfLine, position226)
			}
			return true
		l225:
			position, tokenIndex = position225, tokenIndex225
			return false
		},
		/* 33 EndOfFile <- <!.> */
		nil,
		nil,
		/* 36 Action0 <- <{ p.addDeclarationIdentifier(text) }> */
		nil,
		/* 37 Action1 <- <{ p.addValue() }> */
		nil,
		/* 38 Action2 <- <{ p.LineDone() }> */
		nil,
		/* 39 Action3 <- <{ p.addAction(text) }> */
		nil,
		/* 40 Action4 <- <{ p.addEntity(text) }> */
		nil,
		/* 41 Action5 <- <{ p.LineDone() }> */
		nil,
		/* 42 Action6 <- <{ p.addParamKey(text) }> */
		nil,
		/* 43 Action7 <- <{  p.addFirstValueInList() }> */
		nil,
		/* 44 Action8 <- <{  p.lastValueInList() }> */
		nil,
		/* 45 Action9 <- <{  p.addParamHoleValue(text) }> */
		nil,
		/* 46 Action10 <- <{  p.addAliasParam(text) }> */
		nil,
		/* 47 Action11 <- <{ p.addStringValue(text) }> */
		nil,
		/* 48 Action12 <- <{ p.addStringValue(text) }> */
		nil,
		/* 49 Action13 <- <{ p.addParamValue(text) }> */
		nil,
		/* 50 Action14 <- <{  p.addParamRefValue(text) }> */
		nil,
		/* 51 Action15 <- <{ p.addParamCidrValue(text) }> */
		nil,
		/* 52 Action16 <- <{ p.addParamIpValue(text) }> */
		nil,
		/* 53 Action17 <- <{ p.addParamValue(text) }> */
		nil,
		/* 54 Action18 <- <{ p.LineDone() }> */
		nil,
		/* 55 Action19 <- <{ p.LineDone() }> */
		nil,
	}
	p.rules = _rules
}
