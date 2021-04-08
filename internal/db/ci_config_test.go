package db

import (
	"log"
	"runtime"
	"testing"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
)

func TestReadConf(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	log.Print(filename)
	err := conf.Init("")
	if err != nil {
		panic(err)
	}
	ciConfig, err := ReadConf("loheagn", "ThirdRepo", "master")
	if err != nil {
		panic(err)
	}
	log.Print(ciConfig)
}
