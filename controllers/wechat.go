package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/httplib"
	"encoding/json"
	"lottery/db"
)

// Operations about Users
type WechatController struct {
	beego.Controller
}

const WECHAT_LOGIN_REDIRECT = "wechatLoginErr"

// @router /:vldCode [get]
func (u *WechatController) Get() {
	vldCode := u.GetString(":vldCode")
	redirectUrl := ""
	if vldCode != "" {
		beego.Debug("vldCode is ", vldCode)
		retlist , err :=db.Dia.Query(" select * from elastos_info where vldCode = '" + vldCode+"'")
		if err != nil || retlist.Len() == 0{
			beego.Error("vldCode len " , retlist.Len(),err)
			u.Abort(WECHAT_LOGIN_REDIRECT)
		}
		redirectUrl =retlist.Front().Value.(map[string]string)["redirectUrl"]
	}else{
		beego.Error("validation code is blank")
		u.Abort(WECHAT_LOGIN_REDIRECT)
	}

	code := u.GetString("code")
	if code == "" {
		beego.Error("wechat code is blank")
		u.Abort(WECHAT_LOGIN_REDIRECT)
	}
	appId := beego.AppConfig.String("AppId")
	secret := beego.AppConfig.String("AppSecret")

	accessToken_url := "https://api.weixin.qq.com/sns/oauth2/access_token?appid="+appId+"&secret="+secret+"&code="+code+"&grant_type=authorization_code"
	req := httplib.Get(accessToken_url)
	retBytes , _ := req.Bytes()
	retMap := make(map[string]interface{})
	err := json.Unmarshal(retBytes,&retMap)
	if err != nil || retMap == nil{
		beego.Error(err)
		u.Abort(WECHAT_LOGIN_REDIRECT)
	}
	var accessToken string
	if token , ok := retMap["access_token"];!ok {
		beego.Error("acccessToken is blank")
		u.Abort(WECHAT_LOGIN_REDIRECT)
	}else{
		accessToken = token.(string)
	}
	openid := retMap["openid"].(string)
	beego.Debug("accessToken is ", accessToken,", openid is ",openid)

	weL , err := db.Dia.Query("select * from elastos_members where openid = '" + openid +"'")
	if err != nil {
		beego.Error(err)
		u.Abort(WECHAT_LOGIN_REDIRECT)
	}

	if weL.Len() > 0 {
		we := weL.Front().Value.(map[string]string)
		u.Redirect(redirectUrl+"?headimgurl="+we["wxImg"]+"&nickname="+we["wxNickName"]+"&openid="+we["wxOpenid"]+"&code="+vldCode,301)
		return
	}

	userInfo_url := "https://api.weixin.qq.com/sns/userinfo?access_token="+accessToken+"&openid="+openid+"&lang=zh_CN"
	req = httplib.Get(userInfo_url)
	retBytes , _ = req.Bytes()
	err = json.Unmarshal(retBytes,&retMap)
	if err != nil || retMap == nil{
		beego.Error(err)
		u.Abort(WECHAT_LOGIN_REDIRECT)
	}

	headimgurl := retMap["headimgurl"].(string)
	openid = retMap["openid"].(string)
	nickname := retMap["nickname"].(string)

	beego.Debug("headimgurl " ,headimgurl, " nickname ",nickname)
	// save user info to db
	_ , err = db.Dia.Exec("insert into elastos_members(nickName,openid,headimgurl) values('"+nickname+"','"+openid+"','"+headimgurl+"')")
	if err != nil{
		beego.Error(err)
		u.Abort(WECHAT_LOGIN_REDIRECT)
	}

	u.Redirect(redirectUrl+"?headimgurl="+headimgurl+"&nickname="+nickname+"&openid="+openid+"&code="+vldCode,301)
}