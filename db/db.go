package db

import (
	"github.com/xiaomingfuckeasylife/job/db"
	"github.com/astaxie/beego"
)

var Dia *db.Dialect

func init()  {
	Dia = &db.Dialect{}
	Dia.Create(beego.AppConfig.String("dbDriverName"),beego.AppConfig.String("dbDriverSource"))
}
