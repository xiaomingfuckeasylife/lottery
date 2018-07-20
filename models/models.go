package models

import (
	"github.com/astaxie/beego/orm"
	"lottery/db"
	"time"
	"sync"
)

var (
	SUCCESS              = RetMsg{"", 0, "success"}
	FAIL_INVALID_PARAM   = RetMsg{"", -1, "invalid param"}
	FAIL_INTERNAL_ERROR  = RetMsg{"", -2, "internal error"}
	UpdateTimestamp      = time.Now().Unix()
	ValidateCodeInterval = 60
	UpdatetTimestampLock sync.RWMutex
)

type RetMsg struct {
	Result interface{}
	Error  int
	Desc   string
}

type Elastos_info struct {
	Info_id       int    `orm:"auto";orm:"column(Id)"`
	SecretCode    string `orm:"column(secretCode)"`
	VldCode       string `orm:"column(VldCode)"`
	Remark        string
	Name          string
	RenewInteval  int    `orm:"column(RenewInteval)"`
	Status        int
}

func init() {
	orm.RegisterModel(new(Elastos_info))
	db.Orm = orm.NewOrm()
}

func GetVldCode(secret string) (RetMsg, error) {
	qS := db.Orm.QueryTable("Elastos_info")
	var info Elastos_info
	err := qS.Filter("SecretCode", secret).One(&info, "VldCode")
	if err != nil {
		return RetMsg{}, err
	}
	ret := SUCCESS
	m := make(map[string]interface{})
	m["code"] = info.VldCode
	UpdatetTimestampLock.RLock()
	defer UpdatetTimestampLock.RUnlock()
	m["timeLeft"] = 60 - (time.Now().Unix() - UpdateTimestamp)
	ret.Result = m
	return ret, nil
}
