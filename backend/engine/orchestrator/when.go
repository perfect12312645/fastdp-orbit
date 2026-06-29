package orchestrator

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"os"
	"strings"
)

// 支持的运算符
const (
	opContains    = "contains"
	opNotContains = "!contains"
	opEqual       = "=="
	opNotEqual    = "!="
)

// 自定义模板函数（供模板渲染和 when 条件使用）
var customFuncMap = template.FuncMap{
	// lookup：读取文件内容
	"lookup": func(path string) (string, error) {
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("读取文件 %s 失败: %w", path, err)
		}
		return string(content), nil
	},
	// b64encode：Base64编码
	"b64encode": func(input string) string {
		return base64.StdEncoding.EncodeToString([]byte(input))
	},
	// lower：转小写
	"lower": func(s string) string {
		return strings.ToLower(s)
	},
}

// evaluateWhen 渲染 when 模板表达式并判断条件是否满足
// 支持格式：
//   - "{{.machine.os_name}} == 'ubuntu'"
//   - "{{.machine.ip}} != '192.168.1.1'"
//   - "{{.machine.os_name}} contains 'ubuntu'"
//   - "{{.machine.hostname}} !contains 'test'"
func evaluateWhen(when string, vars map[string]interface{}) (bool, error) {
	rendered, err := RenderTemplate(when, vars)
	if err != nil {
		return false, fmt.Errorf("渲染 when 条件失败: %w", err)
	}
	return evaluateExpression(rendered)
}

// RenderTemplate 使用 Go template 渲染字符串（支持自定义函数）
func RenderTemplate(tplStr string, vars map[string]interface{}) (string, error) {
	tpl, err := template.New("tpl").Funcs(customFuncMap).Parse(tplStr)
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
