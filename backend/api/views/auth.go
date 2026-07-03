package views

import (
	"net/http"
	"regexp"
	"time"
	"unicode"

	"fastdp-orbit/backend/api/middleware"
	"fastdp-orbit/backend/database"
	"fastdp-orbit/backend/models/common"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// loginData 登录成功返回的数据
type loginData struct {
	Token string   `json:"token"`
	User  userData `json:"user"`
}

// userData 用户信息数据
type userData struct {
	ID            uint    `json:"id"`
	Username      string  `json:"username"`
	Nickname      string  `json:"nickname"`
	Avatar        string  `json:"avatar"`
	Role          string  `json:"role"`
	Email         string  `json:"email"`
	MustChangePwd bool    `json:"must_change_pwd"`
	LastLoginAt   *string `json:"last_login_at"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// successResp 返回统一成功响应
func successResp(data any) gin.H {
	return gin.H{"code": 0, "message": "success", "data": data}
}

// failResp 返回统一失败响应
func failResp(msg string) gin.H {
	return gin.H{"code": 1, "message": msg, "data": nil}
}

// validatePasswordStrength 校验密码强度：至少8位，包含大小写字母和数字
func validatePasswordStrength(password string) string {
	if len(password) < 8 {
		return "密码长度不能少于8位"
	}

	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}

	if !hasUpper {
		return "密码必须包含至少一个大写字母"
	}
	if !hasLower {
		return "密码必须包含至少一个小写字母"
	}
	if !hasDigit {
		return "密码必须包含至少一个数字"
	}

	// 检查是否包含危险字符（防止shell注入）
	if matched, _ := regexp.MatchString(`[;&$\\|]`, password); matched {
		return "密码包含不允许的特殊字符（; & $ \\ |）"
	}

	return ""
}

// Login 用户登录
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, failResp("请提供用户名和密码"))
		return
	}

	db := database.GetDB()

	var user common.User
	if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, failResp("用户名或密码错误"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, failResp("用户名或密码错误"))
		return
	}

	token, err := middleware.GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, failResp("生成令牌失败"))
		return
	}

	// 更新最后登录时间
	now := time.Now()
	user.LastLoginAt = &now
	db.Model(&user).Update("last_login_at", now)

	var lastLoginStr *string
	if user.LastLoginAt != nil {
		s := user.LastLoginAt.Format("2006-01-02 15:04:05")
		lastLoginStr = &s
	}

	c.JSON(http.StatusOK, successResp(loginData{
		Token: token,
		User: userData{
			ID:            user.ID,
			Username:      user.Username,
			Nickname:      user.Nickname,
			Avatar:        user.Avatar,
			Role:          user.Role,
			Email:         user.Email,
			MustChangePwd: user.MustChangePwd,
			LastLoginAt:   lastLoginStr,
		},
	}))
}

// GetUserInfo 获取当前用户信息
func GetUserInfo(c *gin.Context) {
	userID, _ := middleware.GetCurrentUser(c)

	db := database.GetDB()
	var user common.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, failResp("用户不存在"))
		return
	}

	var lastLoginStr *string
	if user.LastLoginAt != nil {
		s := user.LastLoginAt.Format("2006-01-02 15:04:05")
		lastLoginStr = &s
	}

	c.JSON(http.StatusOK, successResp(userData{
		ID:            user.ID,
		Username:      user.Username,
		Nickname:      user.Nickname,
		Avatar:        user.Avatar,
		Role:          user.Role,
		Email:         user.Email,
		MustChangePwd: user.MustChangePwd,
		LastLoginAt:   lastLoginStr,
	}))
}

// ChangePassword 修改密码
func ChangePassword(c *gin.Context) {
	userID, _ := middleware.GetCurrentUser(c)

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, failResp("请提供旧密码和新密码"))
		return
	}

	db := database.GetDB()
	var user common.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, failResp("用户不存在"))
		return
	}

	// 校验旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusBadRequest, failResp("旧密码错误"))
		return
	}

	// 新密码不能与旧密码相同
	if req.OldPassword == req.NewPassword {
		c.JSON(http.StatusBadRequest, failResp("新密码不能与旧密码相同"))
		return
	}

	// 校验密码强度
	if errMsg := validatePasswordStrength(req.NewPassword); errMsg != "" {
		c.JSON(http.StatusBadRequest, failResp(errMsg))
		return
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, failResp("密码加密失败"))
		return
	}

	// 更新密码并清除强制修改标记
	db.Model(&user).Updates(map[string]any{
		"password":        string(hashedPassword),
		"must_change_pwd": false,
	})

	c.JSON(http.StatusOK, successResp(gin.H{"message": "密码修改成功"}))
}

// UpdateProfileRequest 更新个人信息请求
type UpdateProfileRequest struct {
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
}

// UpdateProfile 更新个人信息（昵称、邮箱）
func UpdateProfile(c *gin.Context) {
	userID, _ := middleware.GetCurrentUser(c)

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, failResp("参数错误"))
		return
	}

	db := database.GetDB()
	if err := db.Model(&common.User{}).Where("id = ?", userID).Updates(map[string]any{
		"nickname": req.Nickname,
		"email":    req.Email,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, failResp("更新失败"))
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{"message": "更新成功"}))
}

// Logout 退出登录（JWT 无状态，仅返回成功）
func Logout(c *gin.Context) {
	c.JSON(http.StatusOK, successResp(gin.H{"message": "已退出登录"}))
}
