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
	"math"
	"github.com/xiaomingfuckeasylife/job/conf"
	"sync"
	"errors"
	"lottery/models"
	"github.com/astaxie/beego"
	"time"
)

var (
	seed = "1234567890abcdefghijklmnopqrstuvwxyz"
	syncTxlock sync.RWMutex
	feeLock sync.RWMutex
	syncHeight int
)

var (
	GET_BEST_HEIGHT_URL             = conf.Config.ChainApi.GetBestHeight
	GET_BLOCK_BY_HEIGHT_URL         = conf.Config.ChainApi.GetBlockByHeight
	GET_BLOCK_BY_HASH               = conf.Config.ChainApi.GetBlockByHash
	GET_TX_URL                      = conf.Config.ChainApi.GetTransactionByHash
	SEND_TRANSFER_URL               = conf.Config.ChainApi.SendTransfer
	TX_PERIOD                       = conf.Config.Job.TxPeriod
	FEE_PERIOD                      = conf.Config.Job.FeePeriod
	FEE_NUM                         = conf.Config.Fee.FeeNum
	FEE_AMT                         = conf.Config.Fee.FeeAMT
	SENDER_PUBADDR                  = conf.Config.Fee.SenderPubAddr
	SENDER_PRIVKEY                  = conf.Config.Fee.SenderPrivKey
	SELA                    float64 = 100000000
	InitialHeight                   = conf.Config.InitialHeight
)

type Job_info struct {
	Name string
	Renew_interval int
	Exec func()
	sync.Mutex
}

func Process()  {

	jobs ,err:= processActivityInfo()
	if err == nil {
		for _ ,v := range jobs {
			go cron.AddScheduleBySec(int64(v.Renew_interval),v.Exec)
		}
	}

	go cron.AddScheduleBySec(TX_PERIOD, func() {
		syncTxlock.Lock()
		log.Println("sync block start")
		defer syncTxlock.Unlock()
		_, err := processTx()
		if err != nil {
			log.Fatal(err)
		}
	})

	go cron.AddScheduleByHours(FEE_PERIOD, func() {
		// TODO get addresses from tables
		feeLock.Lock()
		log.Println("send Fee start")
		defer feeLock.Unlock()
		ri := make([]map[string]string, 1)
		m := make(map[string]string)
		m["address"] = "EKDb9T8hDgT5CwrvxRuoCeKN3WcAKCShB2"
		m["amount"] = fmt.Sprintf("%.8f", math.Round(FEE_AMT*FEE_NUM*SELA)/SELA)
		ri[0] = m

		b, err := json.Marshal(ri)
		if err != nil {
			log.Fatal("json marshal error ", err)
		}
		fmt.Println(string(b))
		processFee(string(b))
	})

}

func processActivityInfo() ([]Job_info , error ){
	qS := db.Orm.QueryTable("activity_info")
	var infos []models.Activity_info
	_ , err := qS.All(&infos)
	if err != nil {
		beego.Error(err.Error())
		return nil , err
	}
	var jobs []Job_info
	for _,v := range infos {
		j := Job_info{}
		j.Name = "vld_code"
		j.Renew_interval = v.RenewInteval
		if v.RenewInteval == 0 {
			continue
		}
		j.Exec = func() {
			j.Lock()
			defer j.Unlock()
			v.VldCode = random(10)
			models.UpdatetTimestampLock.Lock()
			db.Orm.Update(&v,"VldCode")
			models.UpdateTimestamp = time.Now().Unix()
			models.UpdatetTimestampLock.Unlock()
		}
		jobs = append(jobs,j)
		models.ValidateCodeInterval = v.RenewInteval
	}
	return jobs , nil

}

func random(n int) string{
	num := len(seed)
	var buf []byte
	for i:=0 ;i <n;i++{
		buf = append(buf,seed[rand.Intn(num)])
	}
	sha256 := sha256.Sum256(buf)
	return hex.EncodeToString(sha256[:])
}


func processFee(receivInfo string) (bool, error) {
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
	log.Printf("ret Msg : %s \n", string(bytes))
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
		list, err := db.Dia.Query(" select height from tx_info order by height desc limit 1")
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
			start = start + 1
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
			if !isValid(memo) {
				continue
			}
			//if !strings.HasPrefix(memo,"chinajoy") {
			//	continue
			//}
			sql := "insert into tx_info (txid,height,memo,timestamp,blockhash) values('" + txId + "'," + strconv.Itoa(blockHeight) + ",'" + memo + "'," + timestamp + ",'" + blockHash + "')"
			_, err = db.Dia.Save(sql)
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
