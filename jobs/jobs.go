package jobs

import (
	"math/rand"
	"github.com/xiaomingfuckeasylife/job/cron"
	"encoding/hex"
	"lottery/db"
	"crypto/sha256"
	"strings"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"strconv"
	"log"
	"fmt"
	"github.com/xiaomingfuckeasylife/job/conf"
	"sync"
	"errors"
	"lottery/models"
	"github.com/astaxie/beego"
	"time"
	"math"
	"os"
)

const (
	SEED        = "1234567890abcdefghijklmnopqrstuvwxyz"
	SELA        = 100000000
)

var (
	GET_BEST_HEIGHT_URL     = conf.Config.ChainApi.GetBestHeight
	GET_BLOCK_BY_HEIGHT_URL = conf.Config.ChainApi.GetBlockByHeight
	GET_BLOCK_BY_HASH       = conf.Config.ChainApi.GetBlockByHash
	GET_TX_URL              = conf.Config.ChainApi.GetTransactionByHash
	SEND_TRANSFER_URL       = conf.Config.ChainApi.SendTransfer
	GEN_ADDR 			    = conf.Config.ChainApi.GenAddr
	TX_PERIOD               = conf.Config.Job.TxPeriod
	FEE_PERIOD              = conf.Config.Job.FeePeriod
	FEE_NUM                 = conf.Config.Fee.FeeNum
	FEE_AMT                 = conf.Config.Fee.FeeAMT
	SENDER_PUBADDR          = conf.Config.Fee.SenderPubAddr
	SENDER_PRIVKEY          = conf.Config.Fee.SenderPrivKey
	InitialHeight           = conf.Config.InitialHeight
	syncHeight 				= 0
)

// job_info details of a job , period type 1:sec & 2:min 3:hour
type job_info struct {
	name           string
	renew_interval int
	exec           func()
	sync.Mutex
	periodType     int
}

func Process() {
	jobs, err := assembleJobInfo()
	if err == nil {
		for _, v := range jobs {
			switch v.periodType {
			case 1:
				go cron.AddScheduleBySec(int64(v.renew_interval), v.exec)
			case 2:
				go cron.AddScheduleByMin(int64(v.renew_interval), v.exec)
			case 3:
				go cron.AddScheduleByHours(int64(v.renew_interval), v.exec)
			}
		}
	}
}

func assembleJobInfo() ([]job_info, error) {
	qS := db.Orm.QueryTable("elastos_info")
	var infos []models.Elastos_info
	_, err := qS.All(&infos)
	if err != nil {
		beego.Error(err.Error())
		return nil, err
	}
	var jobs []job_info
	for _, v := range infos {
		j := job_info{}
		j.name = "vld_code"
		j.renew_interval = v.RenewInteval
		if v.RenewInteval == 0 {
			continue
		}
		j.exec = func() {
			j.Lock()
			defer j.Unlock()
			v.VldCode = random(10)
			models.UpdatetTimestampLock.Lock()
			db.Orm.Update(&v, "VldCode")
			models.UpdateTimestamp = time.Now().Unix()
			models.UpdatetTimestampLock.Unlock()
		}
		j.Mutex = sync.Mutex{}
		j.periodType = 1
		jobs = append(jobs, j)
		models.ValidateCodeInterval = v.RenewInteval
	}
	j1 := job_info{
		name:           "chinajoy_game_fee",
		renew_interval: int(FEE_PERIOD),
		Mutex:          sync.Mutex{},
	}
	j1.exec = func() {
		// TODO get addresses from tables
		j1.Lock()
		beego.Debug(j1.name)
		defer j1.Unlock()
		ri := make([]map[string]string, 1)
		m := make(map[string]string)
		m["address"] = "EKDb9T8hDgT5CwrvxRuoCeKN3WcAKCShB2"
		m["amount"] = fmt.Sprintf("%.8f", math.Round(FEE_AMT*FEE_NUM*SELA)/SELA)
		ri[0] = m
		b, err := json.Marshal(ri)
		if err != nil {
			beego.Error("json marshal error ", err)
			return
		}
		processChinaJoyGamingFee(string(b))
	}
	j1.periodType = 3
	j2 := job_info{
		name:           "sync_block",
		renew_interval: int(TX_PERIOD),
		Mutex:          sync.Mutex{},
	}
	j2.exec = func() {
		j2.Lock()
		beego.Debug(j2.name)
		defer j2.Unlock()
		_, err := processTx()
		if err != nil {
			beego.Error("process sync_block error ", err)
			return
		}
	}
	j2.periodType = 1

	j3 := job_info{
		name:           "sync_block",
		renew_interval: 30,
		Mutex:          sync.Mutex{},
	}
	j3.exec = func() {
		j3.Lock()
		beego.Debug(j3.name)
		defer j3.Unlock()
		err := addAddrFee()
		if err != nil {
			beego.Error(" addAddrFee for j3", err)
			return
		}
	}
	j3.periodType = 1
	jobs = append(jobs, j1)
	jobs = append(jobs, j2)
	jobs = append(jobs, j3)

	return jobs, nil

}

func random(n int) string {
	num := len(SEED)
	var buf []byte
	for i := 0; i < n; i++ {
		buf = append(buf, SEED[rand.Intn(num)])
	}
	sha256 := sha256.Sum256(buf)
	return hex.EncodeToString(sha256[:])
}

func processChinaJoyGamingFee(receivInfo string) (bool, error) {
	body := `{
			"Action":"transfer",
			"Version":"1.0.0",
			"Data":
				{"senderAddr":"` + SENDER_PUBADDR + `",
   				 "senderPrivateKey":"` + SENDER_PRIVKEY + `",
				 "memo":"chinajoy fee transfer",
				 "receiver":` + receivInfo + `
				}
			}`
	r := strings.NewReader(body)
	rsp, err := http.Post(SEND_TRANSFER_URL, "application/json", r)
	if err != nil {
		return false, err
	}
	bytes, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return false, err
	}
	beego.Debug("ret Msg : %s \n", string(bytes))
	var ret map[string]interface{}
	err = json.Unmarshal(bytes, &ret)
	if err != nil {
		return false, err
	}
	if ret["error"].(float64) != 0 {
		return false, errors.New(" transfer api error ")
	}
	// TODO update table status
	return true, nil
}

func InitAddr() error {
	l , _ := db.Dia.Query("select * from elastos_addresses")
	if num , _ := strconv.Atoi(conf.Config.InitialAddressNum) ; l.Len() >= num{
		beego.Info("no need to initial address")
		return nil
	}
	body := `{
			"Action":"genAddress",
			"Version":"1.0.0",
			"Data":
				{"num":` + conf.Config.InitialAddressNum + `}
			}`
	beego.Info(body)
	r := strings.NewReader(body)
	rsp, err := http.Post(GEN_ADDR, "application/json", r)
	if err != nil {
		return err
	}
	bytes, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return err
	}
	beego.Debug("ret Msg : %s \n", string(bytes))
	var ret map[string]interface{}
	err = json.Unmarshal(bytes, &ret)
	if err != nil {
		return err
	}
	if ret["error"].(float64) != 0 {
		os.Exit(-1)
	}
	resultList := ret["result"].([]interface{})

	var insertSql string
	var receivInfo string
	fee := conf.Config.InitialAddressFee
	for i , vA := range resultList {
		v :=vA.(map[string]interface{})
		if i == 0 && len(resultList) != 1{
			insertSql += "insert into elastos_addresses (privKey,publicKey,publicAddr,status) values " + "('" + v["privKey"].(string) +"','"+v["publicKey"].(string) +"','" +v["address"].(string) + "',1),"
			receivInfo += `[{"address":"`+v["address"].(string)+`","amount":"`+fee+`"},`
		}else if i == 0 && len(resultList)  == 1{
			insertSql += "insert into elastos_addresses (privKey,publicKey,publicAddr,status) values " + "('" + v["privKey"].(string) +"','"+v["publicKey"].(string) +"','" +v["address"].(string) + "',1)"
			receivInfo += `[{"address":"`+v["address"].(string)+`","amount":"`+fee+`"}]`
		}else if i != len(resultList) - 1  {
			insertSql += "('" + v["privKey"].(string) +"','"+v["publicKey"].(string) +"','" +v["address"].(string) + "',0),"
			receivInfo += `{"address":"`+v["address"].(string)+`","amount":"`+fee+`"},`
		}else {
			insertSql += "('" + v["privKey"].(string) +"','"+v["publicKey"].(string) +"','" +v["address"].(string) + "',0)"
			receivInfo += `{"address":"`+v["address"].(string)+`","amount":"`+fee+`"}]`
		}
	}
	_ , err = db.Dia.Exec(insertSql)
	if err != nil {
		beego.Error("initial Address error")
		os.Exit(-1)
	}
	body = `{
			"Action":"transfer",
			"Version":"1.0.0",
			"Data":
				{"senderAddr":"` + SENDER_PUBADDR + `",
   				 "senderPrivateKey":"` + SENDER_PRIVKEY + `",
				 "memo":"chinajoy fee transfer",
				 "receiver":` + receivInfo + `
				}
			}`
	r = strings.NewReader(body)
	rsp, err = http.Post(SEND_TRANSFER_URL, "application/json", r)
	if err != nil {
		return err
	}
	bytes, err = ioutil.ReadAll(rsp.Body)
	if err != nil {
		return err
	}
	beego.Debug("ret Msg : %s \n", string(bytes))
	err = json.Unmarshal(bytes, &ret)
	if err != nil {
		return err
	}
	if ret["error"].(float64) != 0 {
		os.Exit(-1)
	}
	beego.Debug("Done initialize address")
	return nil
}

func addAddrFee() error{
	v , err := strconv.ParseFloat(conf.Config.InitialAddressFee,64)
	if err != nil {
		beego.Error(err)
	}
	var ret map[string]interface{}
	var receivInfo string
	l , err := db.Dia.Query("select * from elastos_addresses where spendTime/" + fmt.Sprintf("%.0f",math.Round(v/0.000001))+">1/2")
	if l.Len() > 0 {
		len := l.Len()
		var i int
		var updateAddrs string
		for e := l.Front(); e != nil ; e = e.Next() {
			v := e.Value.(map[string]string)
			spendTime , _ := strconv.ParseFloat(v["spendTime"],64)
			fee := (spendTime + 1) * 0.000001
			if i == 0 && l.Len() == 1 {
				receivInfo += `[{"address":"`+v["publicAddr"]+`","amount":"`+fmt.Sprintf("%.8f",fee)+`"}]`
				updateAddrs += "('"+v["publicAddr"]+"')"
			}else if i == 0 && l.Len() > 1 {
				receivInfo += `[{"address":"`+v["publicAddr"]+`","amount":"`+fmt.Sprintf("%.8f",fee)+`"},`
				updateAddrs += "('"+v["publicAddr"]+"',"
			}else if i != len - 1  {
				receivInfo += `{"address":"`+v["publicAddr"]+`","amount":"`+fmt.Sprintf("%.8f",fee)+`"},`
				updateAddrs += "'"+v["publicAddr"]+"',"
			}else {
				receivInfo += `{"address":"`+v["publicAddr"]+`","amount":"`+fmt.Sprintf("%.8f",fee)+`"}]`
				updateAddrs += "'"+v["publicAddr"]+"')"
			}
			i++
		}
		body := `{
			"Action":"transfer",
			"Version":"1.0.0",
			"Data":
				{"senderAddr":"` + SENDER_PUBADDR + `",
   				 "senderPrivateKey":"` + SENDER_PRIVKEY + `",
				 "memo":"chinajoy fee transfer",
				 "receiver":` + receivInfo + `
				}
			}`
		beego.Info("request info " , body)
		r := strings.NewReader(body)
		rsp, err := http.Post(SEND_TRANSFER_URL, "application/json", r)
		if err != nil {
			return err
		}
		bytes, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			return err
		}
		beego.Debug("ret Msg : %s \n", string(bytes))
		err = json.Unmarshal(bytes, &ret)
		if err != nil {
			return err
		}
		if ret["error"].(float64) != 0 {
			beego.Error("send fee Error")
		}else{
			_ , err = db.Dia.Exec("update elastos_addresses set spendTime = 0 where publicAddr in " + updateAddrs)
			if err != nil {
				beego.Error("update elastos_addresses spendtime error " ,err)
			}
		}
	}else {
		beego.Info("no need to send fee")
	}
	return nil
}

func processTx() (bool, error) {
	var start int
	var end int

	resp, err := http.Get(GET_BEST_HEIGHT_URL)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	result := make(map[string]interface{})
	json.Unmarshal(body, &result)
	end = int(result["Result"].(float64))
	if syncHeight == 0 {
		list, err := db.Dia.Query(" select height from elastos_txblock order by height desc limit 1")
		if err != nil {
			return false, err
		}
		if list.Len() == 0 {
			start = int(InitialHeight)
		} else {
			m := list.Front().Value.(map[string]string)
			startStr := m["height"]
			start, err = strconv.Atoi(startStr)
			if err != nil {
				return false, err
			}
			if start < int(InitialHeight) {
				start = int(InitialHeight)
			} else {
				start = start + 1
			}
		}
	} else {
		start = syncHeight + 1
	}
	if start >= end+1 {
		log.Println("no block need to sync")
		return true, nil
	}
	for height := start; height < end+1; height++ {
		log.Printf("sync height : %d \n", height)
		resp, err := http.Get(GET_BLOCK_BY_HEIGHT_URL + strconv.Itoa(height))
		if err != nil {
			return false, err
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}
		var blockInfo map[string]interface{}
		json.Unmarshal(body, &blockInfo)
		rstMap := blockInfo["Result"].(map[string]interface{})
		blockHash := rstMap["hash"].(string)
		timestamp := fmt.Sprintf("%.0f", rstMap["time"])
		txArr := rstMap["tx"].([]interface{})
		blockHeight := int(rstMap["height"].(float64))
		for i := 0; i < len(txArr); i++ {
			txInfo := txArr[i].(map[string]interface{})
			txId := txInfo["txid"].(string)
			if int(txInfo["type"].(float64)) != 2 {
				continue;
			}
			attributes := txInfo["attributes"].([]interface{})
			memoByte, err := hex.DecodeString(attributes[0].(map[string]interface{})["data"].(string))
			if err != nil {
				return false, err
			}
			memo := string(memoByte)
			log.Printf("memo info %s \n", memo)
			if !strings.HasPrefix(memo,"chinajoy") {
				continue
			}
			//if !strings.HasPrefix(memo,"chinajoy") {
			//	continue
			//}
			sql := "insert into elastos_txblock (txid,height,memo,timestamp,blockhash) values('" + txId + "'," + strconv.Itoa(blockHeight) + ",'" + memo + "'," + timestamp + ",'" + blockHash + "')"
			_, err = db.Dia.Exec(sql)
			if err != nil {
				return false, err
			}
			// TODO update table status
		}
		if height == end && height > syncHeight {
			syncHeight = height
		}
	}

	return true, nil
}

func isValid(memo string) bool {
	if (len(memo) > 0) {
		for i := 0; i < len(memo); i++ {
			if (memo[i] > 102 || (memo[i] < 97 && memo[i] > 57) || memo[i] < 48) {
				return false
			}
		}
	}
	return true
}

