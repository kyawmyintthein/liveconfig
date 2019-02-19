package main

import (
	"context"
	"fmt"
	"github.com/kyawmyintthein/liveconfig"
)

type LogInfo struct{
	LogRotation bool `etcd:"log_rotation" json:"log_rotation"`
}

type LogConfig struct{
	Info LogInfo `etcd:"info" json:"info"`
	LogLevel string     `etcd:"log_level" json:"log_level"`
	LogFilepath string  `etcd:"log_level" json:"log_filepath"`
}

type GeneralConfig struct{
	Log LogConfig `etcd:"log" json:"log"`
}

func main(){
	generalConfig := GeneralConfig{
			LogConfig{
				LogInfo{true},
				"debug",
			"test/test.log",
		},
	}

	lConfig, err := liveconfig.NewConfig(
		&generalConfig,
		"/test-project/config",
		liveconfig.WithHosts("localhost:2379"),
		liveconfig.WithRequesttimeout(20),
		liveconfig.WithDialtimeout(30))
	if err != nil{
		fmt.Printf("Error : %v \n", err)
		return
	}
	fmt.Printf("Config %+v \n", generalConfig)

	lConfig.Watch(&generalConfig)

	lConfig.AddReloadCallback("/test-project/config/log_level", func(ctx context.Context) error {
		fmt.Println(generalConfig.Log.LogLevel)
		return nil
	})

	select{}
}


