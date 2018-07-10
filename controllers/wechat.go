package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/httplib"
	"lottery/models"
	"encoding/json"
)

// Operations about Users
type WechatController struct {
	beego.Controller
}

// @Title Get
// @Description get user by vldCode
// @Param	uid		path 	string	true		"The key for staticblock"
// @Success 200 {object} models.Wechat
// @Failure 403 :vldCode is empty
// @router /:vldCode [get]
func (u *WechatController) Get() {
	vldCode := u.GetString(":vldCode")
	if vldCode != "" {
		//TODO validate validation code
		beego.Debug("vldCode is ", vldCode)
	}

	rm := models.RetMsg{}

	code := u.GetString("code")
	if code == "" {
		rm.Error = 0
		rm.Desc = "SUCCESS"
		rm.Result = "code can not be blank"
		u.Data["json"] = rm
		u.ServeJSON()
		return
	}
	appId := beego.AppConfig.String("AppId")
	secret := beego.AppConfig.String("AppSecret")
	accessToken_url := "https://api.weixin.qq.com/sns/oauth2/access_token?appid="+appId+"&secret="+secret+"&code="+code+"&grant_type=authorization_code"
	req := httplib.Get(accessToken_url)
	retBytes , err := req.Bytes()
	if err != nil {
		beego.Error("accessToken_url error " ,err)
		rm.Error = 0
		rm.Desc = "SUCCESS"
		rm.Result = err
		u.Data["json"] = rm
		u.ServeJSON()
		return
	}
	retMap := make(map[string]interface{})
	json.Unmarshal(retBytes,&retMap)
	accessToken := retMap["access_token"].(string)
	openid := retMap["openid"].(string)
	beego.Debug("accessToken is %s , openid is %s",accessToken,openid)

	userInfo_url := "https://api.weixin.qq.com/sns/userinfo?access_token="+accessToken+"&openid="+openid+"&lang=zh_CN"
	req = httplib.Get(userInfo_url)
	retBytes , err = req.Bytes()
	if err != nil {
		beego.Error("userInfo_url error " ,err)
		rm.Error = 0
		rm.Desc = "SUCCESS"
		rm.Result = err
		u.Data["json"] = rm
		u.ServeJSON()
		return
	}

	json.Unmarshal(retBytes,&retMap)
	headimgurl := retMap["headimgurl"].(string)
	openid = retMap["openid"].(string)
	nickname := retMap["nickname"].(string)
	beego.Debug("headimgurl %s , openid %s,nickname %s",headimgurl,openid,nickname)

	resultMap := make(map[string]interface{})
	resultMap["headimgurl"] = headimgurl
	resultMap["nickname"] = nickname
	resultMap["openid"] = openid

	rm.Error = 0
	rm.Desc = "SUCCESS"
	rm.Result = resultMap

	u.Data["json"] = rm
	u.ServeJSON()
}


/*
String code = getPara("code");
String state  = getPara("state");
System.out.println("code is " + code + " state is " + state );
String appId = Constant.APP_ID;
String secret = Constant.APP_SECRET;
// get access_token
String accessToken_url = "https://api.weixin.qq.com/sns/oauth2/access_token?appid="+appId+"&secret="+secret+"&code="+code+"&grant_type=authorization_code";
String retJson = MyHttpKit.get(accessToken_url);
System.out.println("retJson :" + retJson);
Map map = (Map) JSON.parse(retJson);
String accessToken = (String) map.get("access_token");
String openid = (String) map.get("openid");
System.out.println("openid : " + openid + " accessToken :" + accessToken);
// get userInfo
String userInfo_url = "https://api.weixin.qq.com/sns/userinfo?access_token="+accessToken+"&openid="+openid+"&lang=zh_CN";
String retJson2 = MyHttpKit.get(userInfo_url);
System.out.println("userinfo :" + retJson2);
Map userInfoMap = (Map) JSON.parse(retJson2);
String headimgurl = (String) userInfoMap.get("headimgurl");
String openId = (String) userInfoMap.get("openid");
String name = (String)userInfoMap.get("nickname");
System.out.println("nickName is " + name);
 */