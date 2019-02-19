
# # LiveConfig

## Features

 1. Runtime config changes
 2. Support for etcd

### Example
```
papackage main  
  
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
  LogLevel string `etcd:"log_level" json:"log_level"`  
  LogFilepath string `etcd:"log_filepath" json:"log_filepath"`  
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
  
   ok := lConfig.AddReloadCallback("log/log_level", func(ctx context.Context){  
      fmt.Println("Test",generalConfig.Log.LogLevel)  
   })  
  
   err = lConfig.OverrideConfigValues(&generalConfig)  
   if err != nil{  
      fmt.Printf("OverrideConfigValues Error : %v \n", err)  
      return  
  }  
   lConfig.Watch(&generalConfig)
}
```
