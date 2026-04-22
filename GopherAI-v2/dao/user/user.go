package user

import (
	"GopherAI/common/mysql"
	myredis "GopherAI/common/redis"
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
func IsExistUser(username string) (bool, *model.User) {
	user, err := GetUserByUsername(username)

	if err == gorm.ErrRecordNotFound || user == nil {
		return false, nil
	}

	return true, user
}

func GetUserByUsername(username string) (*model.User, error) {
	if user, hit, err := myredis.GetUserByUsername(ctx, username); err == nil && hit {
		return user, nil
	}

	user, err := mysql.GetUserByUsername(username)
	if err != nil {
		return user, err
	}

	_ = myredis.CacheUserByUsername(ctx, user)
	return user, nil
}

func Register(username, email, password string) (*model.User, bool) {
	if user, err := mysql.InsertUser(&model.User{
		Email:    email,
		Name:     username,
		Username: username,
		Password: utils.MD5(password),
	}); err != nil {
		return nil, false
	} else {
		_ = myredis.DeleteUserCacheByUsername(ctx, username)
		return user, true
	}
}
