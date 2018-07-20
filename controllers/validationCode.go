package controllers

import (
	"github.com/astaxie/beego"
	"lottery/models"
)

type ValidationCodeController struct {
	beego.Controller
}

// @Title Get
// @Description get validation code
// @Param	secret		path 	string	true		"the access token to get validation code"
// @Success 200
// @router /:secret [get]
func (u *ValidationCodeController) Get() {
	secret := u.GetString(":secret")
	if secret == "" {
		ret := models.FAIL_INVALID_PARAM
		ret.Result = "secret can not be blank"
		u.Data["json"] = ret
		u.ServeJSON()
		return
	}
	ret , err := models.GetVldCode(secret)
	if err != nil {
		ret = models.FAIL_INVALID_PARAM
		ret.Result = err.Error()
		u.Data["json"] = ret
		u.ServeJSON()
		return
	}
	u.Data["json"] = ret
	u.ServeJSON()
}