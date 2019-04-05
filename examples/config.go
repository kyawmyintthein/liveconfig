package main

import (
	"context"
	"fmt"
	"github.com/kyawmyintthein/liveconfig"
	"time"
)

type LogInfo struct{
	LogRotation bool `etcd:"log_rotation" json:"log_rota" mapstructure:"log_rotation"`
}

type LogConfig struct{
	Info LogInfo 		`json:"info" mapstructure:"info"`
	LogLevel string     `json:"log_level" mapstructure:"log_level"`
	LogFilepath string  `json:"log_filepath" mapstructure:"log_filepath"`
}

type GeneralConfig struct{
	Log LogConfig `etcd:"log" json:"log" mapstructure:"log"`
}

func main(){
	generalConfig := new(GeneralConfig)

	lConfig, err := liveconfig.NewConfig(
		generalConfig,
		"/test-project/config",
		liveconfig.WithFilepaths([]string{"/Users/kyawmyintthein/go/src/github.com/kyawmyintthein/liveconfig/examples/config.yml"}),
		liveconfig.WithConfigType("yaml"),
		liveconfig.WithHosts("localhost:2379"),
		liveconfig.WithRequesttimeout(20),
		liveconfig.WithDialtimeout(30))
	if err != nil{
		fmt.Printf("Error : %v \n", err)
		return
	}

	lConfig.AddReloadCallback("log/log_level", func(ctx context.Context){
		fmt.Println("log/log_level changed to : ", generalConfig.Log.LogLevel)
	})

	err = lConfig.Start()
	if err != nil{
		fmt.Printf("Start Error : %v \n", err)
		return
	}

	for {
		time.Sleep(5 * time.Second)
		fmt.Printf("Config %+v \n", generalConfig)
	}
}


