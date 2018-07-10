package main

import (
	_ "lottery/routers"

	"github.com/astaxie/beego"
	"lottery/db"
)

func main() {
	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
		//beego.SetStaticPath("/lottery/wechat", "wechat")
	}
	defer db.Dia.Close()
	beego.Run()
}

func init() {
	beego.SetLogger("file", `{"filename":"logs/lottery.log"}`)
}

