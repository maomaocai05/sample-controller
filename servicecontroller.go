package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	slb "github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	aliyunutil "k8s.io/sample-controller/pkg/aliyun/ecs"
	aliyun "k8s.io/sample-controller/pkg/aliyun/slb"
	"k8s.io/sample-controller/pkg/signals"
)

type Empty struct{}

type ServiceController struct {
	informerFactory informers.SharedInformerFactory
	svcInformer     coreinformers.ServiceInformer
	workqueue       workqueue.RateLimitingInterface
	kubeClient      kubernetes.Clientset
}

func (c *ServiceController) Run(stopCh chan struct{}) error {
	// Starts all the shared informers that have been created by the factory so
	// far.
	c.informerFactory.Start(stopCh)
	// wait for the initial synchronization of the local cache.
	if !cache.WaitForCacheSync(stopCh, c.svcInformer.Informer().HasSynced) {
		return fmt.Errorf("failed to sync")
	}
	fmt.Println("Starting workers")
	// Launch two workers to process Foo resources

	go wait.Until(c.runWorker, time.Second, stopCh)

	fmt.Println("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil

}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *ServiceController) runWorker() {
	for c.processNextWorkItem() {
	}
}
func (c *ServiceController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			//utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		// if err := c.syncHandler(key); err != nil {
		// 	// Put the item back on the workqueue to handle any transient errors.
		// 	c.workqueue.AddRateLimited(key)
		// 	return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		// }
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		//utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *ServiceController) handleObject(obj interface{}) {
	svc := obj.(*v1.Service)
	fmt.Print(svc)
}
func NewSVCController(informerFactory informers.SharedInformerFactory, kubeClient kubernetes.Clientset) *ServiceController {
	svcInformer := informerFactory.Core().V1().Services()

	c := &ServiceController{
		informerFactory: informerFactory,
		svcInformer:     svcInformer,
		workqueue:       workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	svcInformer.Informer().AddEventHandler(
		// Your custom resource event handlers.
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(o interface{}) {
				fmt.Printf("add func %s\n", o.(*v1.Service).Name)
				processUpdateSvc(o.(*v1.Service), &c.kubeClient)
			},
			UpdateFunc: func(old, new interface{}) {
				newSvc := new.(*v1.Service)
				//oldSvc := old.(*v1.Service)
				if newSvc.ResourceVersion == newSvc.ResourceVersion {
					// Periodic resync will send update events for all known Deployments.
					// Two different versions of the same Deployment will always have different RVs.
					return
				}
				fmt.Printf("updateFunc %s\n", new.(*v1.Service).Name)

				//c.handleObject(new)
			},
			DeleteFunc: func(o interface{}) {
				processDelSvc(o.(*v1.Service), &c.kubeClient)

			},
			// FilterFunc: func(obj interface{}) bool {
			// 	newSvc := obj.(*v1.Service)
			// 	if newSvc.Namespace != "default" {
			// 		return false
			// 	}
			// 	klog.Infof("filter: svc [%s/%s]\n", newSvc.Namespace, newSvc.Name)
			// 	return true
			// },
			// Handler: cache.ResourceEventHandlerFuncs{
			// 	AddFunc: func(obj interface{}) {
			// 		newSvc := obj.(*v1.Service)
			// 		klog.Infof("controller: add svc, svc [%s/%s]\n", newSvc.Namespace, newSvc.Name)
			// 	},

			// 	UpdateFunc: func(oldObj, newObj interface{}) {
			// 		newSvc := newObj.(*v1.Service)
			// 		klog.Infof("controller: Update svc, pod [%s/%s]\n", newSvc.Namespace, newSvc.Name)
			// 	},

			// 	DeleteFunc: func(obj interface{}) {
			// 		delSvc := obj.(*v1.Service)
			// 		klog.Infof("controller: Delete svc, pod [%s/%s]\n", delSvc.Namespace, delSvc.Name)
			// 	},
			// },
		},
	)

	return c
}

func processDelSvc(svc *v1.Service, kubeClient *kubernetes.Clientset) {
	if svc.Spec.Type == "LoadBalancer" {
		loadBalancerId, Ok := svc.Annotations["LoadBalancerId"]
		if !Ok {
			//不存在LoadBalancerId跳过

		} else {
			//存在则删除
			fmt.Printf("%s,%s", svc.Name, loadBalancerId)
			request := slb.CreateDeleteLoadBalancerRequest()
			request.LoadBalancerId = loadBalancerId
			slbClient, _ := aliyun.GetSLBClient()
			_, err := slbClient.DeleteLoadBalancer(request)
			if err != nil {
				fmt.Printf("deleteslb fail %s\n", err)
			}

		}
	}
}

func processUpdateSvc(svc *v1.Service, kubeClient *kubernetes.Clientset) {
	//pod ip到弹性网卡id的映射
	var eniMap = make(map[string]ecs.NetworkInterfaceSet)
	var eniSet = make(map[string]Empty)
	if svc.Spec.Type == "LoadBalancer" {
		slbClient, _ := aliyun.GetSLBClient()

		ecsClient, _ := aliyunutil.GetECSClient()

		ctx := signals.SetupSignalHandler()
		LoadBalancerId, Ok := svc.Annotations["LoadBalancerId"]
		selector := svc.Spec.Selector

		//根据selector获取对应pod
		podList, _ := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{LabelSelector: labels.SelectorFromSet(selector).String()})
		//遍历pod list ，根据pod ip获取弹性网卡id
		for _, item := range podList.Items {

			ip := item.Status.PodIP
			if len(strings.TrimSpace(ip)) != 0 {
				desNetrequest := ecs.CreateDescribeNetworkInterfacesRequest()
				desNetrequest.PrivateIpAddress = &[]string{ip}

				desNetresponse, err := ecsClient.DescribeNetworkInterfaces(desNetrequest)

				if err != nil {
					fmt.Printf("获取弹性网卡信息失败 : %s\n", err)
				}
				netWorkIntercacesSets := desNetresponse.NetworkInterfaceSets
				if len(netWorkIntercacesSets.NetworkInterfaceSet) > 0 {
					//eniId := netWorkIntercacesSets.NetworkInterfaceSet[0].NetworkInterfaceId

					eniMap[ip] = netWorkIntercacesSets.NetworkInterfaceSet[0]
					eniSet[netWorkIntercacesSets.NetworkInterfaceSet[0].NetworkInterfaceId] = Empty{}
				}
			}
			//不存在loadbalance 则创建
			if !Ok {
				//不存在LoadBalancerId 创建
				//1创建slb
				//var podPort = ""
				var loadbalancerId = ""
				request := slb.CreateCreateLoadBalancerRequest()
				response, err := slbClient.CreateLoadBalancer(request)
				slbIp := response.Address
				if err != nil {
					fmt.Printf("创建slb失败 : %s\n", err)
				} else {
					loadbalancerId = response.LoadBalancerId
					port := svc.Spec.Ports[0].Port
					//podPort = svc.Spec.Ports[0].TargetPort.StrVal

					crelbTCPListenerRequest := slb.CreateCreateLoadBalancerTCPListenerRequest()
					crelbTCPListenerRequest.LoadBalancerId = loadbalancerId
					crelbTCPListenerRequest.ListenerPort = requests.Integer(port)
					crelbTCPListenerRequest.BackendServerPort = requests.Integer(port)
					//svc.Annotations[""] = loadbalancerId
					//创建slb监听
					crelbTCPListenerResponse, err := slbClient.CreateLoadBalancerTCPListener(crelbTCPListenerRequest)
					if err != nil {
						fmt.Printf("创建slb监听失败： %s\n", err)
					}
					fmt.Printf("创建slb监听requestid=%s\n", crelbTCPListenerResponse.RequestId)
				}
				//创建slb backend
				l := []string{}
				for key, value := range eniMap {
					s := fmt.Sprintf("{\"ServerId\": \"%s\", \"Weight\": \"100\", \"Type\": \"eni\", \"ServerIp\": \"%s\", \"Port\":\"%d\",\"Description\":\"%s\"}",
						value.NetworkInterfaceId, value.PrivateIpAddress, svc.Spec.Ports[0].Port, key)
					l = append(l, s)
				}
				l1 := strings.Join(l, ",")
				l2 := "[" + l1 + "]"
				createAddBackendServersRequest := slb.CreateAddBackendServersRequest()
				createAddBackendServersRequest.BackendServers = l2
				createAddBackendServersRequest.LoadBalancerId = loadbalancerId
				r, err := slbClient.AddBackendServers(createAddBackendServersRequest)
				if err != nil {
					fmt.Printf("创建backend失败 :%s\n", err)
				} else {
					fmt.Printf("%s", r.RequestId)
				}
				svc.Annotations["LoadBalancerId"] = loadbalancerId
				svc.Spec.ExternalIPs = append(svc.Spec.ExternalIPs, slbIp)
				_, err = kubeClient.CoreV1().Services(svc.Namespace).Update(context.TODO(), svc, metav1.UpdateOptions{})
				if err != nil {
					fmt.Printf("update svc fail %s\n", err)
				}
			} else {
				//存在则需要更新
				request := slb.CreateDescribeLoadBalancerAttributeRequest()
				request.LoadBalancerId = LoadBalancerId
				response, err := slbClient.DescribeLoadBalancerAttribute(request)
				if err != nil {
					fmt.Printf("DescribeLoadBalancerAttribute error %s\n", err)
				}
				backendservers := response.BackendServers.BackendServer
				if len(backendservers) == 1 {
					//serverip := backendservers[0].ServerIp
					serverid := backendservers[0].ServerId
					if _, exist := eniMap[serverid]; exist {
						//无变化 不处理
					} else {
						//删除再添加
						s := fmt.Sprintf("[{\"ServerId\":%s , \"Type\": \"eni\",\"Weight\":\"100\"}]", serverid)
						request := slb.CreateRemoveBackendServersRequest()
						request.LoadBalancerId = LoadBalancerId
						request.BackendServers = s
						_, err := slbClient.RemoveBackendServers(request)
						if err != nil {
							fmt.Printf("RemoveBackendServers err %s\n", err)
						}
						l := []string{}
						for key, value := range eniMap {
							s := fmt.Sprintf("{\"ServerId\": \"%s\", \"Weight\": \"100\", \"Type\": \"eni\", \"ServerIp\": \"%s\", \"Port\":\"%s\",\"Description\":\"%s\"}",
								value.NetworkInterfaceId, value.PrivateIpAddress, svc.Spec.Ports[0].Port, key)
							l = append(l, s)
						}
						l1 := strings.Join(l, ",")
						l2 := "[" + l1 + "]"
						createAddBackendServersRequest := slb.CreateAddBackendServersRequest()
						createAddBackendServersRequest.BackendServers = l2
						createAddBackendServersRequest.LoadBalancerId = LoadBalancerId
						r, err := slbClient.AddBackendServers(createAddBackendServersRequest)
						if err != nil {
							fmt.Printf("创建backend失败 :%s\n", err)
						} else {
							fmt.Printf("%s", r.RequestId)
						}

					}
				} else {
					fmt.Printf("只处理单个实例")
				}

			}
		}
	} else {

	}
}
