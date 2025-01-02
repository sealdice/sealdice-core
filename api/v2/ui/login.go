package ui

import (
	"encoding/binary"
	"encoding/hex"
	"time"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
	"sealdice-core/utils/web/response"
)

// LoginApi 存放登录接口，事实上这个本来想和base合并，但这样的话前端就会有一些要调整记录的工作量
// 想法是等对接完毕并废弃V1接口后，再考虑合并。
type LoginApi struct {
	dice *dice.Dice
}

// DoSignIn
// @Tags      Base
// @Summary   用户登录
// @Accept    application/json
// @Produce   application/json
// @Param     password  body  string  true  "用户密码"
// @Success   200  {object}  response.Result{data=map[string]string,msg=string}  "返回登录成功的token"
// @Failure   400  {object}  response.Result{msg=string}  "返回参数绑定错误或登录失败信息"
// @Router    /login/signin [post]
func (b *LoginApi) DoSignIn(c echo.Context) error {
	v := struct {
		Password string `json:"password"`
	}{}

	err := c.Bind(&v)
	if err != nil {
		return response.FailWithMessage("参数绑定错误", c)
	}

	// 如果UIPasswordHash为空，证明没有密码，自己生成一个
	if b.dice.Parent.UIPasswordHash == "" {
		return response.OkWithData(map[string]string{
			"token": b.generateToken(),
		}, c)
	}
	// 如果UIPasswordHash不为空，证明有密码，验证密码是否正确
	if b.dice.Parent.UIPasswordHash == v.Password {
		return response.OkWithData(map[string]string{
			"token": b.generateToken(),
		}, c)
	}
	return response.FailWithMessage("登录失败，请检查密码是否正确", c)
}

// DoSignInGetSalt
// @Tags      Base
// @Summary   获取密码盐
// @Security  ApiKeyAuth
// @Accept    application/json
// @Produce   application/json
// @Success   200  {object}  response.Result{data=map[string]string,msg=string}  "返回密码盐"
// @Router    /login/salt [get]
func (b *LoginApi) DoSignInGetSalt(c echo.Context) error {
	return response.OkWithData(map[string]string{
		"salt": b.dice.Parent.UIPasswordSalt,
	}, c)
}

// 工具函数放在最下面
func (b *LoginApi) generateToken() string {
	now := time.Now().Unix()
	head := hex.EncodeToString(binary.BigEndian.AppendUint64(nil, uint64(now)))
	token := dice.RandStringBytesMaskImprSrcSB2(64) + ":" + head
	b.dice.Parent.AccessTokens[token] = true
	b.dice.LastUpdatedTime = time.Now().Unix()
	b.dice.Parent.Save()
	return token
}
