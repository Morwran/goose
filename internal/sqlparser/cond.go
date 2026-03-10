package sqlparser

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/expr-lang/expr"
)

// condBlock tracks the state of a -- +goose WHEN / ELSE / END WHEN conditional block.
// At most one block can be active at a time (nesting is not allowed).
// Zero value is ready to use with os.LookupEnv as the default env resolver.
type condBlock struct {
	active   bool
	matched  bool
	emitting bool
	seenElse bool

	// envFunc overrides the default os.LookupEnv.
	envFunc func(string) (string, bool)
}

func (c *condBlock) lookupEnv(key string) (string, bool) {
	if c.envFunc != nil {
		return c.envFunc(key)
	}
	return os.LookupEnv(key)
}

func (c *condBlock) isEmitting() bool {
	if !c.active {
		return true
	}
	return c.emitting
}

// when opens a new conditional block or acts as ELSEIF inside an existing one.
func (c *condBlock) handleWhen(expression string) error {
	if expression == "" {
		return fmt.Errorf("WHEN requires a boolean expression")
	}

	if c.active {
		if c.seenElse {
			return fmt.Errorf("WHEN after ELSE is not allowed inside the same block")
		}
		if c.matched {
			c.emitting = false
			return nil
		}
		result, err := evaluateCondition(expression, c.lookupEnv)
		if err != nil {
			return fmt.Errorf("failed to evaluate WHEN expression %q: %w", expression, err)
		}
		c.emitting = result
		if result {
			c.matched = true
		}
		return nil
	}

	c.active = true
	c.matched = false
	c.emitting = false
	c.seenElse = false

	result, err := evaluateCondition(expression, c.lookupEnv)
	if err != nil {
		c.reset()
		return fmt.Errorf("failed to evaluate WHEN expression %q: %w", expression, err)
	}
	c.emitting = result
	c.matched = result
	return nil
}

func (c *condBlock) handleElse() error {
	if !c.active {
		return fmt.Errorf("ELSE without a preceding WHEN")
	}
	if c.seenElse {
		return fmt.Errorf("duplicate ELSE inside the same WHEN block")
	}
	c.seenElse = true
	c.emitting = !c.matched
	if c.emitting {
		c.matched = true
	}
	return nil
}

func (c *condBlock) handleEndWhen() error {
	if !c.active {
		return fmt.Errorf("END WHEN without a preceding WHEN")
	}
	c.reset()
	return nil
}

func (c *condBlock) reset() {
	envFunc := c.envFunc
	*c = condBlock{envFunc: envFunc}
}

// --------------------------------------------------------------------------
// Expression evaluation
// --------------------------------------------------------------------------

var (
	envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

	// inFuncPattern rewrites in(a, b, c) => (a in [b, c]) because "in" is
	// a reserved operator in expr-lang.
	inFuncPattern = regexp.MustCompile(`\bin\s*\(([^)]+)\)`)
)

func evaluateCondition(expression string, envFunc func(string) (string, bool)) (bool, error) {
	resolved := rewriteInFunc(expression)

	resolved = envVarPattern.ReplaceAllStringFunc(resolved, func(match string) string {
		name := envVarPattern.FindStringSubmatch(match)[1]
		val, _ := envFunc(name)
		return quote(val)
	})

	env := buildExprEnv(envFunc)

	program, err := expr.Compile(resolved, expr.Env(env), expr.AsBool())
	if err != nil {
		return false, fmt.Errorf("compile error: %w", err)
	}
	out, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("runtime error: %w", err)
	}
	b, ok := out.(bool)
	if !ok {
		return false, fmt.Errorf("expression must evaluate to bool, got %T", out)
	}
	return b, nil
}

func buildExprEnv(envFunc func(string) (string, bool)) map[string]interface{} {
	env := make(map[string]interface{})
	for _, kv := range os.Environ() {
		if k, v, ok := strings.Cut(kv, "="); ok {
			env[k] = v
		}
	}
	env["defined"] = func(name string) bool {
		_, ok := envFunc(name)
		return ok
	}
	return env
}

func rewriteInFunc(s string) string {
	return inFuncPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := inFuncPattern.FindStringSubmatch(match)
		args := strings.Split(sub[1], ",")
		if len(args) < 2 {
			return match
		}
		val := strings.TrimSpace(args[0])
		candidates := make([]string, 0, len(args)-1)
		for _, a := range args[1:] {
			candidates = append(candidates, strings.TrimSpace(a))
		}
		return "(" + val + " in [" + strings.Join(candidates, ", ") + "])"
	})
}

func quote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

// extractWhenExpr extracts the boolean expression from a WHEN annotation line.
// Example: "-- +goose WHEN ${PROFILE}=='cluster'" => "${PROFILE}=='cluster'"
func extractWhenExpr(line string) string {
	cmd := strings.ReplaceAll(line, "--", "")
	cmd = strings.Replace(cmd, "+goose", "", 1)
	cmd = strings.TrimSpace(cmd)

	prefix := string(annotationWhen) + " "
	if len(cmd) >= len(prefix) && strings.EqualFold(cmd[:len(prefix)], prefix) {
		return strings.TrimSpace(cmd[len(prefix):])
	}
	return ""
}
