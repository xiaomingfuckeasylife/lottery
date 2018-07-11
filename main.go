package main

import (
	_ "lottery/routers"

	"github.com/astaxie/beego"
	"lottery/db"
	"net/http"
	"html/template"
)


func wechatLoginErr(rw http.ResponseWriter,r *http.Request)  {
	t,_:= template.ParseFiles(beego.BConfig.WebConfig.ViewsPath +"/wechatLoginErr.html")
	//data :=make(map[string]interface{})
	//data["content"] = "page not found"
	data := map[string]interface{}{
		"Title":        "Login Error",
		"Content":      template.HTML("<br> Invalid QR code </br>"),
	}
	t.Execute(rw, data)
}

func main() {
	beego.ErrorHandler("wechatLoginErr",wechatLoginErr)
	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
		//beego.SetStaticPath("/lottery/wechat", "wechat")
	}
	defer db.Dia.Close()
	beego.Run()
}

func init() {
	beego.SetLogger("file", `{"filename":"lottery.log"}`)
}

