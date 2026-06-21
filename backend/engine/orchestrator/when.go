package orchestrator

import (
	"fmt"
	"html/template"
	"strings"
)

// 支持的运算符
const (
	opContains    = "contains"
	opNotContains = "!contains"
	opEqual       = "=="
	opNotEqual    = "!="
)

// evaluateWhen 渲染 when 模板表达式并判断条件是否满足
// 支持格式：
//   - "{{.machine.os_name}} == 'ubuntu'"
//   - "{{.machine.ip}} != '192.168.1.1'"
//   - "{{.machine.os_name}} contains 'ubuntu'"
//   - "{{.machine.hostname}} !contains 'test'"
func evaluateWhen(when string, vars map[string]interface{}) (bool, error) {
	rendered, err := renderWhenTemplate(when, vars)
	if err != nil {
		return false, fmt.Errorf("渲染 when 条件失败: %w", err)
	}
	return evaluateExpression(rendered)
}

// renderWhenTemplate 使用 Go template 渲染 when 表达式中的变量
func renderWhenTemplate(when string, vars map[string]interface{}) (string, error) {
	tpl, err := template.New("when").Parse(when)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	if err := tpl.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// evaluateExpression 解析并求值渲染后的表达式（支持 ==、!=、contains、!contains）
func evaluateExpression(expr string) (bool, error) {
	expr = strings.TrimSpace(expr)
	lowerExpr := strings.ToLower(expr)

	op, left, right, err := parseOperatorAndValues(lowerExpr)
	if err != nil {
		return false, err
	}

	switch op {
	case opEqual:
		return left == right, nil
	case opNotEqual:
		return left != right, nil
	case opContains:
		return strings.Contains(left, right), nil
	case opNotContains:
		return !strings.Contains(left, right), nil
	default:
		return false, fmt.Errorf("不支持的运算符: %s", op)
	}
}

// parseOperatorAndValues 解析表达式中的运算符、左值和右值
func parseOperatorAndValues(expr string) (op, left, right string, err error) {
	// 优先检测最长运算符 !contains，避免被 contains 误判
	if strings.Contains(expr, opNotContains) {
		op = opNotContains
		parts := strings.SplitN(expr, opNotContains, 2)
		if len(parts) != 2 {
			return "", "", "", fmt.Errorf("!contains 表达式格式错误: %s（正确格式: 'key !contains value'）", expr)
		}
		left = strings.TrimSpace(parts[0])
		right = trimQuotes(strings.TrimSpace(parts[1]))
		return op, left, right, nil
	} else if strings.Contains(expr, opContains) {
		op = opContains
		parts := strings.SplitN(expr, opContains, 2)
		if len(parts) != 2 {
			return "", "", "", fmt.Errorf("contains 表达式格式错误: %s（正确格式: 'key contains value'）", expr)
		}
		left = strings.TrimSpace(parts[0])
		right = trimQuotes(strings.TrimSpace(parts[1]))
		return op, left, right, nil
	} else if strings.Contains(expr, opEqual) {
		op = opEqual
		parts := strings.SplitN(expr, opEqual, 2)
		if len(parts) != 2 {
			return "", "", "", fmt.Errorf("== 表达式格式错误: %s（正确格式: 'key == value'）", expr)
		}
		left = strings.TrimSpace(parts[0])
		right = trimQuotes(strings.TrimSpace(parts[1]))
		return op, left, right, nil
	} else if strings.Contains(expr, opNotEqual) {
		op = opNotEqual
		parts := strings.SplitN(expr, opNotEqual, 2)
		if len(parts) != 2 {
			return "", "", "", fmt.Errorf("!= 表达式格式错误: %s（正确格式: 'key != value'）", expr)
		}
		left = strings.TrimSpace(parts[0])
		right = trimQuotes(strings.TrimSpace(parts[1]))
		return op, left, right, nil
	}

	return "", "", "", fmt.Errorf("未找到支持的运算符（支持 ==/!=/contains/!contains）: %s", expr)
}

// trimQuotes 去除字符串前后的单引号或双引号
func trimQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
