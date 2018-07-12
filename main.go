package main

import (
	"github.com/astaxie/beego"
	"html/template"
	"lottery/jobs"
	_ "lottery/routers"
	"net/http"
)

func wechatLoginErr(rw http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles(beego.BConfig.WebConfig.ViewsPath + "/wechatLoginErr.html")
	data := map[string]interface{}{
		"Title":   "Login Error",
		"Content": template.HTML("<br> Invalid QR code </br>"),
	}
	t.Execute(rw, data)
}

func main() {
	beego.ErrorHandler("wechatLoginErr", wechatLoginErr)
	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
		//beego.SetStaticPath("/lottery/wechat", "wechat")
	}
	beego.Run()
}

func init() {
	beego.SetLogger("file", `{"filename":"lottery.log"}`)
	go jobs.Process()
}
