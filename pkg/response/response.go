package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response is the standard API envelope: {code, msg, data}.
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// Error codes
const (
	CodeSuccess = 0

	// 1001–1999: client errors
	CodeBadRequest   = 1001
	CodeNotFound     = 1002
	CodeInvalidParam = 1003
	CodeFileTypeErr  = 1004

	// 2001–2999: server errors
	CodeInternalError = 2001
	CodeDBError       = 2002
	CodeFileIOError   = 2003
)

// Default error messages
var errMsg = map[int]string{
	CodeBadRequest:    "请求参数错误",
	CodeNotFound:      "资源不存在",
	CodeInvalidParam:  "无效的请求参数",
	CodeFileTypeErr:   "仅支持TXT文件",
	CodeInternalError: "服务器内部错误",
	CodeDBError:       "数据库操作失败",
	CodeFileIOError:   "文件操作失败",
}

// Msg returns the default message for an error code.
func Msg(code int) string {
	if m, ok := errMsg[code]; ok {
		return m
	}
	return "未知错误"
}

// Success writes a success response with data.
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: CodeSuccess, Msg: "ok", Data: data})
}

// SuccessMsg writes a success response with a custom message and no data.
func SuccessMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Response{Code: CodeSuccess, Msg: msg, Data: nil})
}

// Error writes an error response using the default message for the code.
func Error(c *gin.Context, code int) {
	c.JSON(http.StatusOK, Response{Code: code, Msg: Msg(code), Data: nil})
}

// ErrorMsg writes an error response with a custom detail appended to the default message.
func ErrorMsg(c *gin.Context, code int, detail string) {
	c.JSON(http.StatusOK, Response{Code: code, Msg: Msg(code) + ": " + detail, Data: nil})
}
