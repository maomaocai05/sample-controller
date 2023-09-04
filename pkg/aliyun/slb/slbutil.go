package aliyun

import (
	"fmt"
	"sync"

	slb "github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/spf13/viper"
)

var slbClient *slb.Client
var err error
var once sync.Once

func GetSLBClient() (client *slb.Client, errmsg error) {
	once.Do(func() {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		var regionId = viper.GetString("region_id")
		var accessKeyId = viper.GetString("access_key_id")
		var accessKeySecret = viper.GetString("access_key_secret")

		slbClient, err = slb.NewClientWithAccessKey(regionId, accessKeyId, accessKeySecret)
		//slbClient, err = slb.NewClientWithAccessKey("REGION_ID", "ACCESS_KEY_ID", "ACCESS_KEY_SECRET")
	})
	if err != nil {
		fmt.Printf("创建slb客户端失败： %s\n", err)
		return nil, err
	}
	return slbClient, nil
}
