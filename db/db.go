package db

import "job/db"

var Dia *db.Dialect

func init()  {
	Dia = &db.Dialect{}
	Dia.Create("mysql","root:@tcp(127.0.0.1:3306)/lottery")
}
