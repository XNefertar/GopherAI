package user

import (
	"GopherAI/common/mysql"
	"GopherAI/model"
	"GopherAI/utils"
	"context"

	"gorm.io/gorm"
)

const (
	CodeMsg     = "GopherAI验证码如下(验证码仅限于2分钟有效): "
	UserNameMsg = "GopherAI的账号如下，请保留好，后续可以用账号进行登录 "
)

var ctx = context.Background()

// 这边只能通过账号进行登录
func IsExistUser(ctx context.Context, username string) (bool, *model.User) {

	user, err := mysql.GetUserByUsername(ctx, username)

	if err == gorm.ErrRecordNotFound || user == nil {
		return false, nil
	}

	return true, user
}

func Register(ctx context.Context, username, email, password string) (*model.User, bool) {
	if user, err := mysql.InsertUser(ctx, &model.User{
		Email:    email,
		Name:     username,
		Username: username,
		Password: utils.MD5(password),
	}); err != nil {
		return nil, false
	} else {
		return user, true
	}
}
