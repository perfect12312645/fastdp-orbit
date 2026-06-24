package errs

import "net/http"

// BizError 业务错误，携带业务码和用户可见提示
type BizError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *BizError) Error() string { return e.Message }

// HTTPStatus 根据业务码映射到 HTTP 状态码
func (e *BizError) HTTPStatus() int {
	switch {
	// 通用参数/校验错误 → 400
	case e.Code == CodeParamInvalid || e.Code == CodeIDFormat || e.Code == CodeValidateFailed:
		return http.StatusBadRequest
	// 资源不存在 → 404
	case e.Code == CodeNotFound || e.Code == CodeStageTemplateNotFound ||
		e.Code == CodeStageTemplateVersionMiss ||
		e.Code == CodeWorkflowNotFound ||
		e.Code == CodeMachineGroupNotFound ||
		e.Code == CodeExecutionNotFound:
		return http.StatusNotFound
	// 名称重复 → 409 Conflict
	case e.Code == CodeNameDuplicate || e.Code == CodeStageTemplateNameDuplicate ||
		e.Code == CodeMachineGroupNameDuplicate:
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}

// ==================== 通用错误码 1xxxx ====================

const (
	CodeParamInvalid   = 10000 // 参数校验失败
	CodeIDFormat       = 10001 // ID 格式错误
	CodeNotFound       = 10002 // 资源不存在
	CodeNameDuplicate  = 10003 // 名称重复
	CodeValidateFailed = 10004 // 业务校验失败
)

// ==================== 阶段模板 2xxxx ====================

const (
	CodeStageTemplateNotFound      = 20001
	CodeStageTemplateNameDuplicate = 20002
	CodeStageTemplateValidate      = 20003
	CodeStageTemplateVersionMiss   = 20004
)

// ==================== 工作流 3xxxx ====================

const (
	CodeWorkflowNotFound = 30001
	CodeWorkflowValidate = 30002
)

// ==================== 机器分组 4xxxx ====================

const (
	CodeMachineGroupNotFound      = 40001
	CodeMachineGroupNameDuplicate = 40002
)

// ==================== 执行 5xxxx ====================

const (
	CodeExecutionNotFound = 50001
)

// ==================== 工厂函数 ====================

func NewBadRequest(code int, msg string) *BizError {
	return &BizError{Code: code, Message: msg}
}

func NewNotFound(code int, msg string) *BizError {
	return &BizError{Code: code, Message: msg}
}

func NewConflict(code int, msg string) *BizError {
	return &BizError{Code: code, Message: msg}
}
