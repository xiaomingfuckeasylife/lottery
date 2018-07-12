package models



var SUCCESS_RET_MSG = RetMsg{"",0,"SUCCESS"}

type RetMsg struct {
	Result interface{}
	Error int
	Desc string
}
