package devops

import (
	"gogs.io/gogs/internal/conf"
	"log"
	"runtime"
	"testing"
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
