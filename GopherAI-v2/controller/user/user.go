package user

import (
	"GopherAI/common/code"
	"GopherAI/controller"
	"GopherAI/service/user"
	"net/http"

	"github.com/gin-gonic/gin"
)

type (
	//这里的Username只能是账号登录
	LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	// omitempty当字段为空的时候，不返回这个东西
	LoginResponse struct {
		controller.Response
		Token string `json:"token,omitempty"`
	}
	//验证码由后端生成，存放到redis中，固然需要先发送一次请求CaptchaRequest,然后用返回的验证码
	//邮箱以及密码进行注册，后续再将账号进行返回
	RegisterRequest struct {
		Email    string `json:"email" binding:"required"`
		Captcha  string `json:"captcha"`
		Password string `json:"password"`
	}
	//注册成功之后，直接让其进行登录状态
	RegisterResponse struct {
		controller.Response
		Token string `json:"token,omitempty"`
	}

	CaptchaRequest struct {
		Email string `json:"email" binding:"required"`
	}

	CaptchaResponse struct {
		controller.Response
	}
)

func Login(c *gin.Context) {

	req := new(LoginRequest)
	res := new(LoginResponse)
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	token, statusCode := user.Login(c.Request.Context(), req.Username, req.Password)
	if statusCode != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(statusCode))
		return
	}

	res.Success()
	res.Token = token
	c.JSON(http.StatusOK, res)

}

func Register(c *gin.Context) {

	req := new(RegisterRequest)
	res := new(RegisterResponse)
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	token, statusCode := user.Register(c.Request.Context(), req.Email, req.Password, req.Captcha)
	if statusCode != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(statusCode))
		return
	}

	res.Success()
	res.Token = token
	c.JSON(http.StatusOK, res)
}

func HandleCaptcha(c *gin.Context) {
	req := new(CaptchaRequest)
	res := new(CaptchaResponse)
	//解析参数
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	//给service层进行处理
	statusCode := user.SendCaptcha(c.Request.Context(), req.Email)
	if statusCode != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(statusCode))
		return
	}
	//匿名字段，其实本身res.Success()调用就是res.Response.Success()
	//res.Response.Success()
	res.Success()
	c.JSON(http.StatusOK, res)
}
