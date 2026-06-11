package user

import (
	"GopherAI/common/code"
	myemail "GopherAI/common/email"
	"GopherAI/common/logger"
	myredis "GopherAI/common/redis"
	"GopherAI/dao/user"
	"GopherAI/model"
	"GopherAI/utils"
	"GopherAI/utils/myjwt"
	"context"
)

func Login(ctx context.Context, username, password string) (string, code.Code) {
	l := logger.With("userName", username)
	var userInformation *model.User
	var ok bool
	//1:判断用户是否存在
	if ok, userInformation = user.IsExistUser(ctx, username); !ok {
		l.Warn("user not exist")
		return "", code.CodeUserNotExist
	}
	//2:判断用户是否密码账号正确
	if userInformation.Password != utils.MD5(password) {
		l.Warn("invalid password")
		return "", code.CodeInvalidPassword
	}
	//3:返回一个Token
	token, err := myjwt.GenerateToken(userInformation.ID, userInformation.Username)

	if err != nil {
		l.Error("generate token failed", "error", err)
		return "", code.CodeServerBusy
	}
	l.Info("user logged in")
	return token, code.CodeSuccess
}

func Register(ctx context.Context, email, password, captcha string) (string, code.Code) {
	l := logger.With("email", email)
	var ok bool
	var userInformation *model.User

	//1:先判断用户是否已经存在了
	if ok, _ := user.IsExistUser(ctx, email); ok {
		l.Warn("user already exists")
		return "", code.CodeUserExist
	}

	//2:从redis中验证验证码是否有效
	if ok, _ := myredis.CheckCaptchaForEmail(ctx, email, captcha); !ok {
		l.Warn("invalid captcha")
		return "", code.CodeInvalidCaptcha
	}

	//3：生成11位的账号
	username := utils.GetRandomNumbers(11)
	l = l.With("userName", username)

	//4：注册到数据库中
	if userInformation, ok = user.Register(ctx, username, email, password); !ok {
		l.Error("register to database failed")
		return "", code.CodeServerBusy
	}

	//5：将账号一并发送到对应邮箱上去，后续需要账号登录
	if err := myemail.SendCaptcha(ctx, email, username, user.UserNameMsg); err != nil {
		l.Error("send username email failed", "error", err)
		return "", code.CodeServerBusy
	}

	// 6:生成Token
	token, err := myjwt.GenerateToken(userInformation.ID, userInformation.Username)

	if err != nil {
		l.Error("generate token failed", "error", err)
		return "", code.CodeServerBusy
	}

	l.Info("user registered")
	return token, code.CodeSuccess
}

// 往指定邮箱发送验证码
// 分为以下任务：
// 1：先存放redis
// 2：再进行远程发送
func SendCaptcha(ctx context.Context, email_ string) code.Code {
	l := logger.With("email", email_)
	send_code := utils.GetRandomNumbers(6)
	//1:先存放到redis
	if err := myredis.SetCaptchaForEmail(ctx, email_, send_code); err != nil {
		l.Error("set captcha to redis failed", "error", err)
		return code.CodeServerBusy
	}

	//2:再进行远程发送
	if err := myemail.SendCaptcha(ctx, email_, send_code, myemail.CodeMsg); err != nil {
		l.Error("send captcha email failed", "error", err)
		return code.CodeServerBusy
	}

	l.Info("captcha sent")
	return code.CodeSuccess
}
