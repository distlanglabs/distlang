package ir

import "testing"

func TestBuildIRConsoleLog(t *testing.T) {
	src := `console.log("Hello from Goja!");`
	ir, err := Build("console.js", src)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	if len(ir.Body) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(ir.Body))
	}

	stmt := ir.Body[0]
	if stmt.Type != "ExpressionStatement" {
		t.Fatalf("unexpected stmt type: %s", stmt.Type)
	}

	if stmt.Expression == nil || stmt.Expression.Type != "CallExpression" {
		t.Fatalf("expected call expression, got %#v", stmt.Expression)
	}

	callee := stmt.Expression.Callee
	if callee == nil || callee.Type != "MemberExpression" {
		t.Fatalf("expected member expression callee, got %#v", callee)
	}

	if callee.Property != "log" {
		t.Fatalf("expected property log, got %s", callee.Property)
	}

	if len(stmt.Expression.Arguments) != 1 {
		t.Fatalf("expected 1 argument, got %d", len(stmt.Expression.Arguments))
	}

	arg := stmt.Expression.Arguments[0]
	if arg.Type != "Literal" || arg.Literal == nil || arg.Literal.Kind != "string" {
		t.Fatalf("expected string literal arg, got %#v", arg)
	}
}
