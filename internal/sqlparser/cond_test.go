package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// --------------------------------------------------------------------------
// condBlock state machine tests
// --------------------------------------------------------------------------

func TestCondBlock_BasicWhen_True(t *testing.T) {
	t.Parallel()
	c := &condBlock{envFunc: staticEnv("PROFILE", "cluster")}
	require.True(t, c.isEmitting(), "before any block, lines should pass through")

	require.NoError(t, c.handleWhen(`${PROFILE}=="cluster"`))
	require.True(t, c.isEmitting(), "branch matches, should emit")

	require.NoError(t, c.handleEndWhen())
	require.True(t, c.isEmitting(), "after END WHEN, back to normal")
}

func TestCondBlock_BasicWhen_False(t *testing.T) {
	t.Parallel()
	c := &condBlock{envFunc: staticEnv("PROFILE", "standalone")}

	require.NoError(t, c.handleWhen(`${PROFILE}=="cluster"`))
	require.False(t, c.isEmitting(), "branch does not match, should not emit")

	require.NoError(t, c.handleEndWhen())
	require.True(t, c.isEmitting())
}

func TestCondBlock_WhenElse_FallsThrough(t *testing.T) {
	t.Parallel()
	c := &condBlock{envFunc: staticEnv("PROFILE", "standalone")}

	require.NoError(t, c.handleWhen(`${PROFILE}=="cluster"`))
	require.False(t, c.isEmitting())

	require.NoError(t, c.handleElse())
	require.True(t, c.isEmitting(), "ELSE should emit when no branch matched")

	require.NoError(t, c.handleEndWhen())
	require.True(t, c.isEmitting())
}

func TestCondBlock_WhenElse_FirstBranchMatched(t *testing.T) {
	t.Parallel()
	c := &condBlock{envFunc: staticEnv("PROFILE", "cluster")}

	require.NoError(t, c.handleWhen(`${PROFILE}=="cluster"`))
	require.True(t, c.isEmitting())

	require.NoError(t, c.handleElse())
	require.False(t, c.isEmitting(), "ELSE should NOT emit when a branch already matched")

	require.NoError(t, c.handleEndWhen())
}

func TestCondBlock_ElseIf_SecondBranchMatches(t *testing.T) {
	t.Parallel()
	c := &condBlock{envFunc: staticEnv("PROFILE", "standalone")}

	require.NoError(t, c.handleWhen(`${PROFILE}=="cluster"`))
	require.False(t, c.isEmitting())

	require.NoError(t, c.handleWhen(`${PROFILE}=="standalone"`))
	require.True(t, c.isEmitting(), "second WHEN (ELSEIF) should match")

	require.NoError(t, c.handleElse())
	require.False(t, c.isEmitting(), "ELSE should be skipped since second branch matched")

	require.NoError(t, c.handleEndWhen())
}

func TestCondBlock_NoBranchMatched_NoElse(t *testing.T) {
	t.Parallel()
	c := &condBlock{envFunc: staticEnv("PROFILE", "test")}

	require.NoError(t, c.handleWhen(`${PROFILE}=="cluster"`))
	require.False(t, c.isEmitting())

	require.NoError(t, c.handleWhen(`${PROFILE}=="standalone"`))
	require.False(t, c.isEmitting())

	require.NoError(t, c.handleEndWhen())
	require.True(t, c.isEmitting(), "after END WHEN, lines pass through")
}

func TestCondBlock_MultipleSequentialBlocks(t *testing.T) {
	t.Parallel()
	c := &condBlock{envFunc: staticEnv("PROFILE", "standalone")}

	// First block - no match
	require.NoError(t, c.handleWhen(`${PROFILE}=="cluster"`))
	require.False(t, c.isEmitting())
	require.NoError(t, c.handleEndWhen())

	// Second block - match
	require.NoError(t, c.handleWhen(`${PROFILE}=="standalone"`))
	require.True(t, c.isEmitting())
	require.NoError(t, c.handleEndWhen())
}

// --------------------------------------------------------------------------
// condBlock error cases
// --------------------------------------------------------------------------

func TestCondBlock_Error_ElseWithoutWhen(t *testing.T) {
	t.Parallel()
	c := &condBlock{envFunc: staticEnv("X", "1")}
	err := c.handleElse()
	require.Error(t, err)
	require.Contains(t, err.Error(), "ELSE without a preceding WHEN")
}

func TestCondBlock_Error_EndWhenWithoutWhen(t *testing.T) {
	t.Parallel()
	c := &condBlock{envFunc: staticEnv("X", "1")}
	err := c.handleEndWhen()
	require.Error(t, err)
	require.Contains(t, err.Error(), "END WHEN without a preceding WHEN")
}

func TestCondBlock_Error_DuplicateElse(t *testing.T) {
	t.Parallel()
	c := &condBlock{envFunc: staticEnv("X", "1")}
	require.NoError(t, c.handleWhen(`${X}=="1"`))
	require.NoError(t, c.handleElse())
	err := c.handleElse()
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate ELSE")
}

func TestCondBlock_Error_WhenAfterElse(t *testing.T) {
	t.Parallel()
	c := &condBlock{envFunc: staticEnv("X", "1")}
	require.NoError(t, c.handleWhen(`${X}=="2"`))
	require.NoError(t, c.handleElse())
	err := c.handleWhen(`${X}=="1"`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "WHEN after ELSE is not allowed")
}

func TestCondBlock_Error_EmptyExpression(t *testing.T) {
	t.Parallel()
	c := &condBlock{envFunc: staticEnv("X", "1")}
	err := c.handleWhen("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "WHEN requires a boolean expression")
}

// --------------------------------------------------------------------------
// Expression evaluation tests
// --------------------------------------------------------------------------

func TestEvaluateCondition_Equality(t *testing.T) {
	t.Parallel()
	env := staticEnv("MODE", "debug")

	result, err := evaluateCondition(`${MODE}=="debug"`, env)
	require.NoError(t, err)
	require.True(t, result)

	result, err = evaluateCondition(`${MODE}=="release"`, env)
	require.NoError(t, err)
	require.False(t, result)
}

func TestEvaluateCondition_NotEqual(t *testing.T) {
	t.Parallel()
	env := staticEnv("MODE", "debug")

	result, err := evaluateCondition(`${MODE}!="release"`, env)
	require.NoError(t, err)
	require.True(t, result)

	result, err = evaluateCondition(`${MODE}!="debug"`, env)
	require.NoError(t, err)
	require.False(t, result)
}

func TestEvaluateCondition_Negation(t *testing.T) {
	t.Parallel()
	env := staticEnv("MODE", "debug")

	result, err := evaluateCondition(`!(${MODE}=="release")`, env)
	require.NoError(t, err)
	require.True(t, result)
}

func TestEvaluateCondition_And(t *testing.T) {
	t.Parallel()
	env := multiEnv(map[string]string{"A": "1", "B": "2"})

	result, err := evaluateCondition(`${A}=="1" && ${B}=="2"`, env)
	require.NoError(t, err)
	require.True(t, result)

	result, err = evaluateCondition(`${A}=="1" && ${B}=="3"`, env)
	require.NoError(t, err)
	require.False(t, result)
}

func TestEvaluateCondition_Or(t *testing.T) {
	t.Parallel()
	env := multiEnv(map[string]string{"A": "1", "B": "2"})

	result, err := evaluateCondition(`${A}=="x" || ${B}=="2"`, env)
	require.NoError(t, err)
	require.True(t, result)

	result, err = evaluateCondition(`${A}=="x" || ${B}=="y"`, env)
	require.NoError(t, err)
	require.False(t, result)
}

func TestEvaluateCondition_Parentheses(t *testing.T) {
	t.Parallel()
	env := multiEnv(map[string]string{"A": "1", "B": "2", "C": "3"})

	result, err := evaluateCondition(`(${A}=="1" || ${B}=="x") && ${C}=="3"`, env)
	require.NoError(t, err)
	require.True(t, result)

	result, err = evaluateCondition(`(${A}=="x" || ${B}=="x") && ${C}=="3"`, env)
	require.NoError(t, err)
	require.False(t, result)
}

func TestEvaluateCondition_Defined(t *testing.T) {
	t.Parallel()
	env := staticEnv("EXISTS", "value")

	result, err := evaluateCondition(`defined("EXISTS")`, env)
	require.NoError(t, err)
	require.True(t, result)

	result, err = evaluateCondition(`defined("MISSING")`, env)
	require.NoError(t, err)
	require.False(t, result)
}

func TestEvaluateCondition_DefinedWithAnd(t *testing.T) {
	t.Parallel()
	env := staticEnv("CH_CLUSTER", "my-cluster")

	result, err := evaluateCondition(`defined("CH_CLUSTER") && ${CH_CLUSTER}!=""`, env)
	require.NoError(t, err)
	require.True(t, result)
}

func TestEvaluateCondition_DefinedEmpty(t *testing.T) {
	t.Parallel()
	env := staticEnv("EMPTY_VAR", "")

	result, err := evaluateCondition(`defined("EMPTY_VAR") && ${EMPTY_VAR}==""`, env)
	require.NoError(t, err)
	require.True(t, result)
}

func TestEvaluateCondition_In(t *testing.T) {
	t.Parallel()
	env := staticEnv("REGION", "us-east")

	result, err := evaluateCondition(`in(${REGION}, "us-east", "us-west", "eu-central")`, env)
	require.NoError(t, err)
	require.True(t, result)

	result, err = evaluateCondition(`in(${REGION}, "ap-south", "eu-central")`, env)
	require.NoError(t, err)
	require.False(t, result)
}

func TestEvaluateCondition_ComplexExpression(t *testing.T) {
	t.Parallel()
	env := multiEnv(map[string]string{"ENV": "prod", "REGION": "us-east"})

	expr := `${ENV}=="prod" && in(${REGION}, "us-east", "eu-west")`
	result, err := evaluateCondition(expr, env)
	require.NoError(t, err)
	require.True(t, result)
}

func TestEvaluateCondition_UndefinedVarResolves(t *testing.T) {
	t.Parallel()
	env := func(string) (string, bool) { return "", false }

	// An undefined ${VAR} resolves to ""
	result, err := evaluateCondition(`${MISSING}==""`, env)
	require.NoError(t, err)
	require.True(t, result)
}

func TestEvaluateCondition_InvalidExpression(t *testing.T) {
	t.Parallel()
	env := staticEnv("X", "1")

	_, err := evaluateCondition(`${X}==`, env)
	require.Error(t, err)
}

// --------------------------------------------------------------------------
// extractWhenExpr tests
// --------------------------------------------------------------------------

func TestExtractWhenExpr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		line string
		want string
	}{
		{`-- +goose WHEN ${PROFILE}=='cluster'`, `${PROFILE}=='cluster'`},
		{`-- +goose WHEN defined("X") && ${X}!=""`, `defined("X") && ${X}!=""`},
		{`-- +goose WHEN   true  `, `true`},
		{`-- +goose WHEN`, ``},
	}
	for _, tc := range tests {
		got := extractWhenExpr(tc.line)
		require.Equal(t, tc.want, got, "line: %s", tc.line)
	}
}

// --------------------------------------------------------------------------
// helpers
// --------------------------------------------------------------------------

// staticEnv returns an env function that knows a single key-value pair.
func staticEnv(key, value string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		if k == key {
			return value, true
		}
		return "", false
	}
}

// multiEnv returns an env function backed by a map.
func multiEnv(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}
