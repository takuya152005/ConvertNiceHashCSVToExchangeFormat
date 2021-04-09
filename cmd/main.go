package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"time"
)

const (
	ExitOK = iota
	ExitNG
)

// niceHashのレポートは１日毎にまとめてあること
func main() {
	// 時間がないのでここに全部書くが後でリファクタリングすること
	fmt.Println("Start: Convert NiceHash transaction report CSV to CryptoLinC format")

	cmd := Command{}
	os.Exit(cmd.Run(os.Args))
}

type Command struct {}

func (c *Command) Run(args []string) (exit int) {
	var(
		niceHashCSVName string
		outputCSVName string
		cmdName = "ConvertNiceHashCSVToExchangeFormat"
	)

	// 引数チェック
	flags := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	flags.StringVar(&niceHashCSVName, "niceHashCSVName", "","niceHash CSV Name")
	flags.StringVar(&outputCSVName, "outputCSVName", "","output CSV Name")
	flags.Parse(args)
	fmt.Printf("target csv name: %s \n",niceHashCSVName)
	fmt.Printf("output csv name: %s \n",outputCSVName)
	if niceHashCSVName == "" || outputCSVName == "" {
		fmt.Println("Args is null")
		return ExitNG
	}

	// Open niceHash csv
	csv, err := c.readCSV(niceHashCSVName)
	if err != nil {
		fmt.Println(err)
		return ExitNG
	}

	// convert CSV
	ccsv, err := c.convertCryptoLinCCSV(csv)
	if err != nil {
		fmt.Println(err)
		return ExitNG
	}
	fmt.Println(ccsv)

	// output csv
	err = c.writeCSV(outputCSVName, ccsv)
	if err != nil {
		fmt.Println(err)
		return ExitNG
	}

	return ExitOK
}

// niceHash format
// Date time,Local date time,Purpose,Amount (BTC),Exchange rate,Amount (JPY)
// 2020-09-18 00:00:00 GMT,2020-09-18 09:00:00 GMT+09:00,Hashpower mining,0.00020929,1152112.74,241.13
// 2020-09-18 00:00:00 GMT,2020-09-18 09:00:00 GMT+09:00,Hashpower mining fee,-0.00000419,1152112.74,-4.82
type NiceHashCSV struct {
	// 必要なものだけ定義
	dateTime time.Time
	purpose Purpose
	amount float64
}
type Purpose string
const (
	HashpowerMining    = Purpose("Hashpower mining")
	HashpowerMiningFee = Purpose("Hashpower mining fee")
	WithdrawalComplete = Purpose("Withdrawal complete")
	WithdrawalFee      = Purpose("Withdrawal fee")
)
func stringToPurpose(str string) Purpose {
	switch str {
	case string(HashpowerMining):
		return HashpowerMining
	case string(HashpowerMiningFee):
		return HashpowerMiningFee
	case string(WithdrawalComplete):
		return WithdrawalComplete
	case string(WithdrawalFee):
		return WithdrawalFee
	default:
		panic("error string to purpose. str:" + str)
	}
}

func (c *Command) readCSV(name string) ([]NiceHashCSV, error){
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	var (
		line []string
		i = 0
		csv []NiceHashCSV
		layout = "2006-01-02 15:04:05 MST"
	)

	for {
		i++
		line, err = reader.Read()
		if err != nil {
			break
		}
		//fmt.Println(line)

		// Headerはいらない
		if i == 1 {
			continue
		}

		t, err := time.Parse(layout, line[0])
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		amount, err := strconv.ParseFloat(line[3], 64)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		ncsv := NiceHashCSV{
			t,
			stringToPurpose(line[2]),
			amount,
		}
		csv = append(csv, ncsv)
	}

	return csv, nil
}

type TransactionType string
const (
	MINING = TransactionType("MINING")
	SEND   = TransactionType("SEND")
)

func (c *Command) convertCryptoLinCCSV(csv []NiceHashCSV)([][]string, error){
	// niceHashのPurposeのをCryptoLinCの取引種別に変換する
	// 変換対象は下記の２つ
	// Hashpower mining, Hashpower mining fee => MINING
	// Withdrawal complete, Withdrawal fee => SEND

	// CryptoLinC format
	// No,,取引年,取引日,取引時間,取引所,国外取引,取引種別,取引通貨,決済通貨,取引通貨対JPYレート,決済通貨対JPYレート,決済代金,取引量,手数料通貨,手数料,備考
	// ,,2020,9/18,0:00,niceHash,,MINING,BTC,,,,,0.00020929,BTC,0.00000419,

	var (
		//m = make(map[mapKeys]int)
		m = make(map[string][]NiceHashCSV)
		returnCsv = [][]string{
			[]string{"No", "", "取引年", "取引日", "取引時間", "取引所", "国外取引", "取引種別", "取引通貨", "決済通貨", "取引通貨対JPYレート", "決済通貨対JPYレート", "決済代金", "取引量", "手数料通貨", "手数料", "備考"},
		}
	)

	for _, s := range csv {
		// timeだとmapのkeyで不都合があったのでstringに変換
		// mapのkey存在確認ができなかっt
		k := s.dateTime.String()
		m[k] = append(m[k], s)
	}

	// 日付をkeyにしてMapを作成
	mapKey := make([]string, len(m))
	i := 0
	for k := range m {
		mapKey[i] = k
		i++
	}
	sort.Strings(mapKey)



	for _, key := range mapKey {
		//MINING:取引量,手数料
		//SEND:取引量,手数料
		var (
			miningTradingVol float64
			miningFee float64
			sendTradingVol float64
			sendFee float64
		)

		for _, niceHashCSV := range m[key] {
			switch niceHashCSV.purpose {
			case HashpowerMining:
				miningTradingVol = miningTradingVol + niceHashCSV.amount
			case HashpowerMiningFee:
				// Feeは負数
				miningFee = miningFee + niceHashCSV.amount
			case WithdrawalComplete:
				// 正数にする必要があるので絶対値を取る
				sendTradingVol = sendTradingVol + math.Abs(niceHashCSV.amount)
			case WithdrawalFee:
				// Feeは負数
				sendFee = sendFee + niceHashCSV.amount
			default:
				panic("error string to purpose. str:" + niceHashCSV.purpose)
			}
		}

		// csv作成
		if miningTradingVol != 0 || miningFee != 0 {
			s := c.makeCryptoLinCFormat(
				MINING,
				key,
				miningTradingVol,
				miningFee)
			returnCsv = append(returnCsv, s)
		}
		if sendTradingVol != 0 || sendFee != 0 {
			s := c.makeCryptoLinCFormat(
				SEND,
				key,
				sendTradingVol,
				sendFee)
			returnCsv = append(returnCsv, s)
		}
	}

	return returnCsv,nil
}

func (c *Command) makeCryptoLinCFormat(
	transactionType TransactionType,
	dateStr string,
	tradingVol float64,
	fee float64) []string {

	layout := "2006-01-02 15:04:05 +0000 MST"
	t, _ := time.Parse(layout, dateStr)

	// ,,2020,9/18,0:00,niceHash,,MINING,BTC,,,,,0.00020929,BTC,0.00000419,
	s := []string{
		"",
		"",
		t.Format("2006"),
		t.Format("01/02"),
		t.Format("15:04"),
		"niceHash",
		"",
		string(transactionType),
		"BTC",
		"",
		"",
		"",
		"",
		strconv.FormatFloat(tradingVol, 'f', -1, 64),
		"BTC",
		strconv.FormatFloat(fee, 'f', -1, 64),
	}

	return s
}

func (c *Command) writeCSV(outputCSVName string, list [][]string) error{
	file, err := os.Create(outputCSVName)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.WriteAll(list)
	writer.Flush()

	return nil
}
