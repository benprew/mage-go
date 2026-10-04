// Package ron loads the RON card vocabulary into the mage DSL.
package ron

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Kind separates literals, lists, and constructor expressions.
type Kind uint8

const (
	Symbol Kind = iota
	String
	IntegerNode
	List
	Tuple
	Struct
)

// Position identifies an expression in its source file.
type Position struct {
	File   string
	Line   int
	Column int
}

func (p Position) String() string { return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column) }

// Node preserves the distinction between positional and named arguments.
type Node struct {
	Kind     Kind
	Name     string
	String   string
	Integer  int
	Args     []Node
	Fields   map[string]Node
	Position Position
}

// Parse reads a stream of card expressions, including multiline records.
// The parser accepts the RON subset used by the card files. It rejects other syntax.
func Parse(r io.Reader, filename string) ([]Node, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}
	p := parser{data: string(data), file: filename, line: 1, column: 1}
	var nodes []Node
	for {
		if err := p.space(); err != nil {
			return nil, err
		}
		if p.offset == len(p.data) {
			return nodes, nil
		}
		n, err := p.value(0)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
}

type parser struct {
	data, file           string
	offset, line, column int
}

func (p *parser) position() Position { return Position{p.file, p.line, p.column} }
func (p *parser) fail(format string, args ...any) error {
	return fmt.Errorf("%s: %s", p.position(), fmt.Sprintf(format, args...))
}
func (p *parser) starts(s string) bool { return strings.HasPrefix(p.data[p.offset:], s) }
func (p *parser) take() byte {
	b := p.data[p.offset]
	p.offset++
	if b == '\n' {
		p.line++
		p.column = 1
	} else {
		p.column++
	}
	return b
}
func (p *parser) space() error {
	for p.offset < len(p.data) {
		switch {
		case strings.ContainsRune(" \t\r\n\v\f", rune(p.data[p.offset])):
			p.take()
		case p.starts("//"):
			for p.offset < len(p.data) && p.take() != '\n' {
			}
		case p.starts("/*"):
			p.take()
			p.take()
			depth := 1
			for depth > 0 {
				if p.offset == len(p.data) {
					return p.fail("unterminated block comment")
				}
				switch {
				case p.starts("/*"):
					p.take()
					p.take()
					depth++
				case p.starts("*/"):
					p.take()
					p.take()
					depth--
				default:
					p.take()
				}
			}
		default:
			return nil
		}
	}
	return nil
}

func identStart(b byte) bool { return b == '_' || b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' }
func identPart(b byte) bool  { return identStart(b) || b >= '0' && b <= '9' }

func (p *parser) value(depth int) (Node, error) {
	if depth > 256 {
		return Node{}, p.fail("expression exceeds maximum depth")
	}
	if err := p.space(); err != nil {
		return Node{}, err
	}
	n := Node{Position: p.position()}
	if p.offset == len(p.data) {
		return n, p.fail("expected expression")
	}
	b := p.data[p.offset]
	switch {
	case b == '"' || p.starts("r\"") || p.starts("r#"):
		n.Kind = String
		value, err := p.readString()
		n.String = value
		return n, err
	case b == '[':
		n.Kind = List
		return p.group(n, ']', depth)
	case b == '(':
		n.Kind = Tuple
		return p.group(n, ')', depth)
	case b == '+' || b == '-' || b >= '0' && b <= '9':
		n.Kind = IntegerNode
		start := p.offset
		if b == '+' || b == '-' {
			p.take()
		}
		for p.offset < len(p.data) && (p.data[p.offset] >= '0' && p.data[p.offset] <= '9' || p.data[p.offset] == '_') {
			p.take()
		}
		value, err := strconv.Atoi(strings.ReplaceAll(p.data[start:p.offset], "_", ""))
		if err != nil {
			return n, p.fail("invalid integer")
		}
		n.Integer = value
		return n, nil
	case identStart(b):
		start := p.offset
		for p.offset < len(p.data) && identPart(p.data[p.offset]) {
			p.take()
		}
		n.Name = p.data[start:p.offset]
		n.Kind = Symbol
		if err := p.space(); err != nil {
			return n, err
		}
		if p.starts("(") {
			n.Kind = Tuple
			return p.group(n, ')', depth)
		}
		return n, nil
	default:
		return n, p.fail("unexpected character %q", b)
	}
}

func (p *parser) group(n Node, terminator byte, depth int) (Node, error) {
	p.take()
	for {
		if err := p.space(); err != nil {
			return n, err
		}
		if p.offset == len(p.data) {
			return n, p.fail("expected %q", terminator)
		}
		if p.data[p.offset] == terminator {
			p.take()
			return n, nil
		}
		arg, err := p.value(depth + 1)
		if err != nil {
			return n, err
		}
		if err := p.space(); err != nil {
			return n, err
		}
		if p.starts(":") {
			if terminator != ')' || arg.Kind != Symbol || len(n.Args) != 0 {
				return n, p.fail("invalid named field")
			}
			p.take()
			if n.Fields == nil {
				n.Fields = make(map[string]Node)
				n.Kind = Struct
			}
			if _, ok := n.Fields[arg.Name]; ok {
				return n, p.fail("duplicate field %s", arg.Name)
			}
			value, err := p.value(depth + 1)
			if err != nil {
				return n, err
			}
			n.Fields[arg.Name] = value
		} else {
			if n.Fields != nil {
				return n, p.fail("cannot mix named and positional arguments")
			}
			n.Args = append(n.Args, arg)
		}
		if err := p.space(); err != nil {
			return n, err
		}
		if p.offset == len(p.data) {
			return n, p.fail("expected %q", terminator)
		}
		if p.data[p.offset] == terminator {
			p.take()
			return n, nil
		}
		if !p.starts(",") {
			return n, p.fail("expected comma or %q", terminator)
		}
		p.take()
	}
}

func (p *parser) readString() (string, error) {
	if p.starts("r") {
		p.take()
		hashes := 0
		for p.starts("#") {
			p.take()
			hashes++
		}
		if !p.starts("\"") {
			return "", p.fail("expected raw string")
		}
		p.take()
		start := p.offset
		end := "\"" + strings.Repeat("#", hashes)
		for p.offset < len(p.data) {
			if p.starts(end) {
				value := p.data[start:p.offset]
				for range len(end) {
					p.take()
				}
				return value, nil
			}
			p.take()
		}
		return "", p.fail("unterminated raw string")
	}
	p.take()
	var out strings.Builder
	for p.offset < len(p.data) {
		b := p.take()
		if b == '"' {
			return out.String(), nil
		}
		if b != '\\' {
			out.WriteByte(b)
			continue
		}
		if p.offset == len(p.data) {
			return "", p.fail("unterminated string escape")
		}
		escape := p.take()
		switch escape {
		case '"', '\\', '\'':
			out.WriteByte(escape)
		case 'n':
			out.WriteByte('\n')
		case 'r':
			out.WriteByte('\r')
		case 't':
			out.WriteByte('\t')
		case '0':
			out.WriteByte(0)
		case 'u', 'x':
			count := 2
			braced := escape == 'u' && p.starts("{")
			if escape == 'u' {
				count = 4
			}
			if braced {
				p.take()
				count = 0
			}
			start := p.offset
			if braced {
				for p.offset < len(p.data) && !p.starts("}") {
					p.take()
				}
			} else {
				for range count {
					if p.offset < len(p.data) {
						p.take()
					}
				}
			}
			value, err := strconv.ParseUint(p.data[start:p.offset], 16, 32)
			if err != nil || !utf8.ValidRune(rune(value)) {
				return "", p.fail("invalid character escape")
			}
			if braced {
				if !p.starts("}") {
					return "", p.fail("unterminated character escape")
				}
				p.take()
			}
			out.WriteRune(rune(value))
		default:
			return "", p.fail("unsupported string escape %q", escape)
		}
	}
	return "", p.fail("unterminated string")
}
