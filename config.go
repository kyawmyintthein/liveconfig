package liveconfig

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/kyawmyintthein/liveconfig/option"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/spf13/viper"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"github.com/coreos/etcd/clientv3"
)

const DefaultConfigType = "yaml"

// LiveConfig is read the config struct and generate etcd keys according to specification.
// Then, retrieve values from etcd server and override into config struct's value.
// It can also watch the changes of etcd keys by prefix and save into config struct.
type LiveConfig interface {
	Watch(configStructPtr interface{})
	OverrideConfigValues(configStructPtr interface{}) error
	// AddReloadCallback register reinitilization function for specific etcd key
	AddReloadCallback(etcdKey string, fn ReloadCallbackFunc) bool
}

// ReloadCallbackFunc is callback function type. It will get called
// when the value of related etcd key is change.
type ReloadCallbackFunc func(ctx context.Context)

// ConfigJsonKeyWithDataType mapping between config struct json key and data type (reflect.Type)
// example:
// ConfigJsonKeyWithDataType{
//     Key: "logging/log_level"
//     Type: string
// }
type ConfigJsonKeyWithDataType struct {
	// json tag name in struct field
	Key string
	// type of struct field
	Type reflect.Type
}

// Implementation of LiveConfig interface
// It will store mapping between json tag, etcd key and data type on memory.
// It will also keep etcd key and related callback function to reload specific object.
type liveConfig struct {
	// etcd options
	options option.Options

	// etcd client
	etcdCli *clientv3.Client

	// json tag name, type and etcd key mapping
	configJsonEtcdKeyMap map[string]ConfigJsonKeyWithDataType

	// etcd key and callback function mapping
	etcdKeyCallbackFuncMap map[string]ReloadCallbackFunc

	// root path for etcd directory
	prefix string
}

// NewConfig: create new liveConfig object
// init etcd connection according to options
// generate etcd keys from config struct and keep in as map
func NewConfig(configStructPtr interface{}, prefix string, opts ...option.Option) (LiveConfig, error) {
	options := option.NewOptions(opts...)
	liveConfig := &liveConfig{
		prefix:        prefix,
		options:                options,
		configJsonEtcdKeyMap:   make(map[string]ConfigJsonKeyWithDataType),
		etcdKeyCallbackFuncMap: make(map[string]ReloadCallbackFunc),
	}

	err := liveConfig.loadViperConfig(configStructPtr)
	if err != nil {
		return liveConfig,err
	}

	err = liveConfig.initEtcdConn()
	if err != nil {
		return liveConfig,err
	}

	err = liveConfig.generateConfigETCDKeysFromConfig("", "", configStructPtr)
	if err != nil {
		return liveConfig, err
	}

	return liveConfig, nil
}

func (config *liveConfig) loadViperConfig(configStructPtr interface{}) error{

	// config filepath
	filepaths, _ := config.options.Context.Value(filepathsKey{}).([]string)
	if len(filepaths) == 0{
		return fmt.Errorf("empty config file")
	}
	viper.SetConfigFile(filepaths[0])

	// config option
	configType, ok := config.options.Context.Value(configTypeKey{}).(string)
	if !ok {
		configType = DefaultConfigType
	}
	viper.SetConfigType(configType)


	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("Fatal error config file: %s \n", err)
	}

	for _, filepath := range filepaths[1:] {
		err = func(filepath string) error{
			f, err := os.Open(filepath)
			if err != nil{
				return fmt.Errorf("Fatal error read config file: %s \n", err)
			}
			defer f.Close()
			err = viper.MergeConfig(f)
			if err != nil {
				return fmt.Errorf("Fatal error mergeing config file: %s \n", err)
			}
			return nil
		}(filepath)
		if err != nil{
			return err
		}
	}

	err = viper.Unmarshal(configStructPtr)
	if err != nil {
		return fmt.Errorf("Fatal error marshal config file: %s \n", err)
	}

	return nil
}

// initEtcdConn start new etcd connection
func (config *liveConfig) initEtcdConn() error {

	etcdHosts, ok := config.options.Context.Value(hostsKey{}).([]string)
	if !ok {
		return fmt.Errorf("invalid etcd hosts.")
	}

	etcdDialTimeout, _ := config.options.Context.Value(dealTimeoutKey{}).(time.Duration)
	etcdUsername, _ := config.options.Context.Value(usernameKey{}).(string)
	etcdPassword, _ := config.options.Context.Value(passwordKey{}).(string)

	etcdCli, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdHosts,
		DialTimeout: etcdDialTimeout * time.Second,
		Username:    etcdUsername,
		Password:    etcdPassword,
	})
	if err != nil {
		return err
	}
	config.etcdCli = etcdCli

	return nil
}

//generateConfigETCDKeysFromConfig read config struct and generate etcd keys
// configStructPtr should be pointer of struct type
func (config *liveConfig) generateConfigETCDKeysFromConfig(parentFieldJsonTag, parentFieldEtcdTag string, configStructPtr interface{}) error {
	valueOfIStructPointer := reflect.ValueOf(configStructPtr)

	if k := valueOfIStructPointer.Kind(); k != reflect.Ptr {
		return fmt.Errorf("config struct should be pointer type.")
	}

	valueOfIStructPointerElem := valueOfIStructPointer.Elem()

	if k := valueOfIStructPointerElem.Type().Kind(); k != reflect.Struct {
		return fmt.Errorf("config [%s] should be struct kind.", valueOfIStructPointerElem.Type())
	}

	// Below is a further (and definitive) check regarding settability in addition to checking whether it is a pointer earlier.
	if !valueOfIStructPointerElem.CanSet() {
		return fmt.Errorf("unable to set value to struct type!")
	}

	for index := 0; index < valueOfIStructPointerElem.NumField(); index += 1 {
		structField := valueOfIStructPointerElem.Type().Field(index)

		if structField.Anonymous {
			return fmt.Errorf("unsupported anonymous field %s", structField.Name)
		}

		structFieldJsonTag, structFieldEtcdTag := getStructTags(structField)
		if structFieldJsonTag == "" || structFieldEtcdTag == "" {
			continue
		}

		// nested struct
		if structField.Type.Kind() == reflect.Struct{
			valueField := valueOfIStructPointerElem.Field(index)
			jsonTag := structFieldJsonTag
			if parentFieldJsonTag != ""{
				jsonTag = fmt.Sprintf("%s.%s", parentFieldEtcdTag, structFieldJsonTag)
			}
			etcdTag := structFieldJsonTag
			if parentFieldEtcdTag != ""{
				etcdTag = fmt.Sprintf("%s/%s", parentFieldEtcdTag, structFieldEtcdTag)
			}
			config.generateConfigETCDKeysFromConfig(jsonTag, etcdTag, valueField.Addr().Interface())
		}else{
			jsonTag := structFieldJsonTag
			if parentFieldJsonTag != ""{
				jsonTag = fmt.Sprintf("%s.%s", parentFieldJsonTag, structFieldJsonTag)
			}

			jsonKeyType := ConfigJsonKeyWithDataType{
				Key:  jsonTag,
				Type: structField.Type,
			}

			etcdTag := structFieldJsonTag
			if parentFieldEtcdTag != ""{
				etcdTag = fmt.Sprintf("%s/%s", parentFieldEtcdTag, structFieldJsonTag)
			}

			etcdKey := fmt.Sprintf("%s/%s", config.prefix, etcdTag)
			config.configJsonEtcdKeyMap[etcdKey] = jsonKeyType
		}

	}
	return nil
}

// OverrideConfigValues read etcd valeus and sync into config struct
// call the reload callback function
func (config *liveConfig) OverrideConfigValues(configStructPtr interface{}) error {
	kv := config.etcdCli.KV

	etcdRequestTimeout, _ := config.options.Context.Value(requestTimeoutKey{}).(time.Duration)
	ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout*time.Second)
	defer cancel()

	results := make(map[string]interface{})
	for etcdKey, jsonKeyType := range config.configJsonEtcdKeyMap {
		res, err := kv.Get(ctx, etcdKey)
		if err != nil {
			return err
		}

		if len(res.Kvs) > 0 {
			err = convertETCDValueToOriginalType(jsonKeyType, res.Kvs[0], results)
			if err != nil {
				return err
			}
		}
	}

	// save values to struct
	err := config.syncEtcdDataToConfigStruct(results, configStructPtr)
	if err != nil {
		return err
	}

	// call reload callback function
	for etcdKey, _ := range config.configJsonEtcdKeyMap {
		config.runReloadCallbackFuncs(ctx, etcdKey)
	}

	return nil
}

// syncEtcdDataToConfigStruct unmarshal etcd values to config struct
func (config *liveConfig) syncEtcdDataToConfigStruct(results map[string]interface{}, configStructPtr interface{}) error {
	data, err := json.Marshal(results)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, configStructPtr)
	if err != nil {
		return err
	}

	return nil
}

// AddReloadCallback: register callback function with etcd key.
// It will get called when etcd server detect value changes for this key.
func (config *liveConfig) AddReloadCallback(etcdKey string, fn ReloadCallbackFunc) bool {
	etcdKeyWithPrefix := fmt.Sprintf("%s/%s", config.prefix, etcdKey)
	_, ok := config.configJsonEtcdKeyMap[etcdKeyWithPrefix]
	if ok {
		config.etcdKeyCallbackFuncMap[etcdKeyWithPrefix] = fn
	}
	return ok
}

// WatchfromEtcd watch etcd key and sync into config struct.
// It wil also call reload callback function to reinitalize the module.
func (config *liveConfig) Watch(configStructPtr interface{}) {
	ctx := context.Background()
	go func() {
		watchChan := config.etcdCli.Watch(ctx, config.prefix, clientv3.WithPrefix())
		for true {
			select {
			case result := <-watchChan:
				for _, ev := range result.Events {
					if ev.Type == clientv3.EventTypePut{
						var results = make(map[string]interface{})
						configJsonKeyWithDataType, ok := config.configJsonEtcdKeyMap[string(ev.Kv.Key)]
						if ok {
							convertETCDValueToOriginalType(configJsonKeyWithDataType, ev.Kv, results)
							// save values to config struct
							config.syncEtcdDataToConfigStruct(results, configStructPtr)
						}
						// call reload callback functions
						config.runReloadCallbackFuncs(ctx, string(ev.Kv.Key))
					}
				}
			}
		}
	}()
}

//convertETCDValueToOriginalType get the predefiend type of etcd key and convert the etcd value([]byte) to its original type
// It doesn't support ptr type and custom type struct field.
func convertETCDValueToOriginalType(jsonKeyType ConfigJsonKeyWithDataType, kv *mvccpb.KeyValue, results map[string]interface{}) error {
	switch jsonKeyType.Type.Kind() {
	// String
	case reflect.String:
		rootKey, res, err := convertToMap(jsonKeyType.Key,  string(kv.Value), ".")
		if err != nil{
			return err
		}
		results[rootKey] = res[rootKey]
	// Float32
	case reflect.Float32:
		float32Val, err := strconv.ParseFloat(string(kv.Value), 32)
		if err != nil {
			return err
		}

		rootKey, res, err := convertToMap(jsonKeyType.Key,  float32Val, ".")
		if err != nil{
			return err
		}
		results[rootKey] = res[rootKey]

	// Float64
	case reflect.Float64:
		float64Val, err := strconv.ParseFloat(string(kv.Value), 64)
		if err != nil {
			return err
		}

		rootKey, res, err := convertToMap(jsonKeyType.Key,  float64Val, ".")
		if err != nil{
			return err
		}
		results[rootKey] = res[rootKey]
	// Bool
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(string(kv.Value))
		if err != nil {
			return err
		}
		rootKey, res, err := convertToMap(jsonKeyType.Key,  boolValue, ".")
		if err != nil{
			return err
		}
		results[rootKey] = res[rootKey]
	// Int
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.Atoi(string(kv.Value))
		if err != nil {
			return err
		}

		rootKey, res, err := convertToMap(jsonKeyType.Key,  val, ".")
		if err != nil{
			return err
		}
		results[rootKey] = res[rootKey]

	// Uint
	case reflect.Uint16:
		rootKey, res, err := convertToMap(jsonKeyType.Key,  binary.BigEndian.Uint16(kv.Value), ".")
		if err != nil{
			return err
		}
		results[rootKey] = res[rootKey]
	case reflect.Uint32:
		rootKey, res, err := convertToMap(jsonKeyType.Key,  binary.BigEndian.Uint32(kv.Value), ".")
		if err != nil{
			return err
		}
		results[rootKey] = res[rootKey]
	case reflect.Uint64:
		rootKey, res, err := convertToMap(jsonKeyType.Key,  binary.BigEndian.Uint64(kv.Value), ".")
		if err != nil{
			return err
		}
		results[rootKey] = res[rootKey]

	// Map
	case reflect.Map:
		mapReflectValue := reflect.New(jsonKeyType.Type).Interface()
		err := json.Unmarshal(kv.Value, mapReflectValue)
		if err != nil {
			return err
		}

		rootKey, res, err := convertToMap(jsonKeyType.Key, mapReflectValue, ".")
		if err != nil{
			return err
		}
		results[rootKey] = res[rootKey]

	// Slice
	case reflect.Slice:
		sliceReflectValue := reflect.New(jsonKeyType.Type).Interface()
		err := json.Unmarshal(kv.Value, sliceReflectValue)
		if err != nil {
			return err
		}
		rootKey, res, err := convertToMap(jsonKeyType.Key, sliceReflectValue, ".")
		if err != nil{
			return err
		}
		results[rootKey] = res[rootKey]
	}
	return nil
}

// runReloadCallbackFuncs When etcd client detect the value changes on this etcd key, it will call related
// callback function to reload some module or object.
func (config *liveConfig) runReloadCallbackFuncs(ctx context.Context, etcdKey string) {
	callbackFn, ok := config.etcdKeyCallbackFuncMap[etcdKey]
	if ok {
		callbackFn(ctx)
	}
}


func convertToMap(keyString string, value interface{}, delimiter string) (string, map[string]interface{}, error) {
	rootKey := strings.Split(keyString, ".")[0]

	if len(strings.Split(keyString, delimiter)) > 5 {
		return rootKey, nil, fmt.Errorf("Too many levels")
	}

	var currKey string
	result := make(map[string]interface{})
	idx := strings.Index(keyString, delimiter)
	if idx == -1 {
		result[keyString] = value
	} else {
		currKey = keyString[:idx]
		var err error
		_, result[currKey], err = convertToMap(keyString[idx+1:], value, delimiter)
		if err != nil {
			return rootKey, result, err
		}
	}

	return rootKey, result, nil
}