package main

import (
	"context"
	"fmt"
	"github.com/kyawmyintthein/liveconfig"
)

type LogConfig struct{
	LogLevel string `etcd:"level" json:"level"`
	LogFilepath string  `etcd:"filepath" json:"filepath"`
}
type GeneralConfig struct{
	Log LogConfig `etcd:"log" json:"log"`
}

func main(){
	generalConfig := GeneralConfig{
		LogConfig{
			"debug",
			"test/user_service.log",
		},
	}

	lConfig, err := liveconfig.NewConfig(
		&generalConfig,
		"/user-service/config",
		liveconfig.WithHosts("10.30.1.65:2379","10.30.1.66:2379","10.30.1.67:2379"),
		liveconfig.WithUsername("userservice"),
		liveconfig.WithPassword("circles@123"),
		liveconfig.WithRequesttimeout(20),
		liveconfig.WithDialtimeout(30))
	if err != nil{
		fmt.Errorf("Error : %v", err)
		return
	}
	fmt.Printf("Config %+v \n", generalConfig)

	go lConfig.Watch(&generalConfig)

	lConfig.AddReloadCallback("/user-config/config/log/level", func(ctx context.Context) error {
		fmt.Println(generalConfig.Log.LogLevel)
		return nil
	})
}


