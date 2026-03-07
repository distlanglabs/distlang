package ir

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// IR represents a minimal, normalized intermediate form for debugging.
type IR struct {
	Body []Statement `json:"body"`
}

type Statement struct {
	Type       string      `json:"type"`
	Expression *Expression `json:"expression,omitempty"`
}

type Expression struct {
	Type       string       `json:"type"`
	Identifier string       `json:"identifier,omitempty"`
	Literal    *Literal     `json:"literal,omitempty"`
	Callee     *Expression  `json:"callee,omitempty"`
	Arguments  []Expression `json:"arguments,omitempty"`
	Object     *Expression  `json:"object,omitempty"`
	Property   string       `json:"property,omitempty"`
	Computed   bool         `json:"computed,omitempty"`
}

type Literal struct {
	Kind  string      `json:"kind"`
	Value interface{} `json:"value"`
}

// Build parses JavaScript source and returns a simplified IR for debugging.
func Build(filename, source string) (*IR, error) {
	ir := &IR{}
	for _, raw := range splitStatements(source) {
		s, err := convertStmt(raw)
		if err != nil {
			return nil, err
		}
		ir.Body = append(ir.Body, s)
	}

	if len(ir.Body) == 0 {
		return nil, fmt.Errorf("parse: no supported statements found in %s", filename)
	}

	return ir, nil
}

// MarshalIndented returns a pretty JSON encoding of the IR.
func (ir *IR) MarshalIndented() (string, error) {
	data, err := json.MarshalIndent(ir, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func splitStatements(source string) []string {
	parts := strings.Split(source, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func convertStmt(source string) (Statement, error) {
	expr, err := convertExpr(strings.TrimSpace(source))
	if err != nil {
		return Statement{}, err
	}
	return Statement{Type: "ExpressionStatement", Expression: &expr}, nil

}

func convertExpr(source string) (Expression, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return Expression{}, fmt.Errorf("unsupported expression: empty")
	}

	if idx := strings.Index(source, "("); idx > 0 && strings.HasSuffix(source, ")") {
		callee, err := convertExpr(source[:idx])
		if err != nil {
			return Expression{}, err
		}
		argSource := strings.TrimSpace(source[idx+1 : len(source)-1])
		args, err := convertArguments(argSource)
		if err != nil {
			return Expression{}, err
		}
		return Expression{Type: "CallExpression", Callee: &callee, Arguments: args}, nil
	}

	if i := strings.LastIndex(source, "."); i > 0 {
		object, err := convertExpr(source[:i])
		if err != nil {
			return Expression{}, err
		}
		return Expression{Type: "MemberExpression", Object: &object, Property: strings.TrimSpace(source[i+1:]), Computed: false}, nil
	}

	if lit, ok, err := convertLiteral(source); ok || err != nil {
		return lit, err
	}

	return Expression{Type: "Identifier", Identifier: source}, nil
}

func convertArguments(source string) ([]Expression, error) {
	if strings.TrimSpace(source) == "" {
		return nil, nil
	}

	parts := splitArguments(source)
	out := make([]Expression, 0, len(parts))
	for _, part := range parts {
		expr, err := convertExpr(part)
		if err != nil {
			return nil, err
		}
		out = append(out, expr)
	}
	return out, nil
}

func splitArguments(source string) []string {
	var (
		parts   []string
		current strings.Builder
		depth   int
		quote   rune
	)

	for _, r := range source {
		switch {
		case quote != 0:
			current.WriteRune(r)
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
			current.WriteRune(r)
		case r == '(':
			depth++
			current.WriteRune(r)
		case r == ')':
			depth--
			current.WriteRune(r)
		case r == ',' && depth == 0:
			parts = append(parts, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}

	if tail := strings.TrimSpace(current.String()); tail != "" {
		parts = append(parts, tail)
	}

	return parts
}

func convertLiteral(source string) (Expression, bool, error) {
	if unquoted, err := strconv.Unquote(source); err == nil {
		return Expression{Type: "Literal", Literal: &Literal{Kind: "string", Value: unquoted}}, true, nil
	}

	if source == "true" || source == "false" {
		return Expression{Type: "Literal", Literal: &Literal{Kind: "boolean", Value: source == "true"}}, true, nil
	}

	if source == "null" {
		return Expression{Type: "Literal", Literal: &Literal{Kind: "null", Value: nil}}, true, nil
	}

	if n, err := strconv.ParseFloat(source, 64); err == nil {
		return Expression{Type: "Literal", Literal: &Literal{Kind: "number", Value: n}}, true, nil
	}

	return Expression{}, false, nil
}
