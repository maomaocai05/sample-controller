/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	"k8s.io/sample-controller/pkg/signals"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	slb "github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kubeinformers "k8s.io/client-go/informers"
)

func main() {

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	var regionId = viper.GetString("region_id")
	var accessKeyId = viper.GetString("access_key_id")
	var accessKeySecret = viper.GetString("access_key_secret")
	slbClient, err := slb.NewClientWithAccessKey(regionId, accessKeyId, accessKeySecret)

	if err != nil {
		// Handle exceptions
		panic(err)
	}
	fmt.Println(slbClient.SourceIp)

	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the shutdown signal gracefully
	ctx := signals.SetupSignalHandler()
	logger := klog.FromContext(ctx)

	var kubeconfig *string
	// home是家目录，如果能取得家目录的值，就可以用来做默认值
	if home := homedir.HomeDir(); home != "" {
		// 如果输入了kubeconfig参数，该参数的值就是kubeconfig文件的绝对路径，
		// 如果没有输入kubeconfig参数，就用默认路径~/.kube/config
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		// 如果取不到当前用户的家目录，就没办法设置kubeconfig的默认目录了，只能从入参中取
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	// 从本机加载kubeconfig配置文件，因此第一个参数为空字符串
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)

	if err != nil {
		logger.Error(err, "Error building kubeconfig")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	kubeClient, err := kubernetes.NewForConfig(config)

	if err != nil {
		logger.Error(err, "Error building kubernetes clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	svcController := NewSVCController(kubeInformerFactory, *kubeClient)
	kubeInformerFactory.Start(ctx.Done())
	stop := make(chan struct{})
	defer close(stop)
	svcController.Run(stop)
	// 根据 serviceGetName 参数是否为空来决定显示单个 Service 信息还是所有 Service 信息
	serviceList, err := kubeClient.CoreV1().Services("").List(ctx, metav1.ListOptions{})

	for _, service := range serviceList.Items {
		if service.Spec.Type == "LoadBalancer" {
			selector := service.Spec.Selector
			podList, err1 := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{LabelSelector: labels.SelectorFromSet(selector).String()})
			if err1 != nil {
				fmt.Println(err1)
			}
			klog.Info(selector)
			externalIPs := service.Spec.ExternalIPs
			for _, pod := range podList.Items {
				if pod.Status.Phase == "Running" {

				}
				log.Println(pod.Status.PodIP)
			}
			if len(externalIPs) == 0 {

			} else {

			}
		}
		// // 格式化 ServicePort
		// servicePorts := make([]string, 0, len(service.Spec.Ports))
		// for _, p := range service.Spec.Ports {
		// 	servicePorts = append(servicePorts, fmt.Sprintf("%d:%d/%s", p.Port, p.NodePort, p.Protocol))
		// }

		// // 格式化 External IPs
		// externalIPs := make([]string, 0, len(service.Spec.ExternalIPs))
		// for _, ip := range service.Spec.ExternalIPs {
		// 	externalIPs = append(externalIPs, ip)
		// }
		// var externalIPsStr = ""
		// if len(externalIPs) > 0 {
		// 	externalIPsStr = strings.Join(externalIPs, ",")
		// }
		// fmt.Println(externalIPsStr)
	}
	if err != nil {
		fmt.Println("Err:", err)
		return
	}

	// 调用 List 接口获取 Service 列表

	// exampleClient, err := clientset.NewForConfig(cfg)
	// if err != nil {
	// 	logger.Error(err, "Error building kubernetes clientset")
	// 	klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	// }

	// kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	// exampleInformerFactory := informers.NewSharedInformerFactory(exampleClient, time.Second*30)

	// controller := NewController(ctx, kubeClient, exampleClient,
	// 	kubeInformerFactory.Apps().V1().Deployments(),
	// )

	// // notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(ctx.done())
	// // Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	// kubeInformerFactory.Start(ctx.Done())
	// exampleInformerFactory.Start(ctx.Done())

	// if err = controller.Run(ctx, 2); err != nil {
	// 	logger.Error(err, "Error running controller")
	// 	klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	// }
}
