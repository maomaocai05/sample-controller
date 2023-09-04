package aliyunutil

import (
	"fmt"
	"sync"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/spf13/viper"
)

var ecsClient *ecs.Client
var err error
var once sync.Once

func GetECSClient() (client *ecs.Client, errmsg error) {
	once.Do(func() {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		var regionId = viper.GetString("region_id")
		var accessKeyId = viper.GetString("access_key_id")
		var accessKeySecret = viper.GetString("access_key_secret")

		ecsClient, err = ecs.NewClientWithAccessKey(regionId, accessKeyId, accessKeySecret)

	})
	if err != nil {
		fmt.Printf("创建ecs客户端失败： %s\n", err)
		return nil, err
	}
	return ecsClient, err
}
