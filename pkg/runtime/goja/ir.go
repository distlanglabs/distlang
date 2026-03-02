package gojaengine

import (
	"encoding/json"
	"fmt"

	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/parser"
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

// BuildIR parses JavaScript source and returns a simplified IR for debugging.
func BuildIR(filename, source string) (*IR, error) {
	prog, err := parser.ParseFile(nil, filename, source, 0)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	ir := &IR{}
	for _, stmt := range prog.Body {
		s, err := convertStmt(stmt)
		if err != nil {
			return nil, err
		}
		ir.Body = append(ir.Body, s)
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

func convertStmt(n ast.Statement) (Statement, error) {
	switch stmt := n.(type) {
	case *ast.ExpressionStatement:
		expr, err := convertExpr(stmt.Expression)
		if err != nil {
			return Statement{}, err
		}
		return Statement{Type: "ExpressionStatement", Expression: &expr}, nil
	default:
		return Statement{}, fmt.Errorf("unsupported statement: %T", n)
	}
}

func convertExpr(n ast.Expression) (Expression, error) {
	switch expr := n.(type) {
	case *ast.Identifier:
		return Expression{Type: "Identifier", Identifier: expr.Name.String()}, nil
	case *ast.StringLiteral:
		return Expression{Type: "Literal", Literal: &Literal{Kind: "string", Value: expr.Value}}, nil
	case *ast.NumberLiteral:
		return Expression{Type: "Literal", Literal: &Literal{Kind: "number", Value: expr.Value}}, nil
	case *ast.BooleanLiteral:
		return Expression{Type: "Literal", Literal: &Literal{Kind: "boolean", Value: expr.Value}}, nil
	case *ast.NullLiteral:
		return Expression{Type: "Literal", Literal: &Literal{Kind: "null", Value: nil}}, nil
	case *ast.CallExpression:
		callee, err := convertExpr(expr.Callee)
		if err != nil {
			return Expression{}, err
		}
		args := make([]Expression, len(expr.ArgumentList))
		for i, a := range expr.ArgumentList {
			converted, err := convertExpr(a)
			if err != nil {
				return Expression{}, err
			}
			args[i] = converted
		}
		return Expression{Type: "CallExpression", Callee: &callee, Arguments: args}, nil
	case *ast.DotExpression:
		object, err := convertExpr(expr.Left)
		if err != nil {
			return Expression{}, err
		}
		return Expression{Type: "MemberExpression", Object: &object, Property: expr.Identifier.Name.String(), Computed: false}, nil
	default:
		return Expression{}, fmt.Errorf("unsupported expression: %T", n)
	}
}
