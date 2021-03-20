package main

import "testing"

func Test(t *testing.T) {

	var (
		niceHashCSVName = "../data/report_ALL-BTC-JPY-DAY_20200101-20201231.csv"
		outputCSVName = "../data/niceHash_20200101-20201231_to_CryptoLinC.csv"
		args = []string{"-niceHashCSVName=" + niceHashCSVName, "-outputCSVName=" + outputCSVName}
	)

	cmd := Command{}
	got := cmd.Run(args)
	if got != ExitOK {
		t.Fatalf("got:%d exit status but want:%d", got, ExitOK)
	}
}
