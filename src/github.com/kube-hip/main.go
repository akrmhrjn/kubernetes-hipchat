package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/util/wait"
)

func notifySlack(obj interface{}, action string) {
	pod := obj.(*api.Pod)

	room_id := ""
	auth_token := ""
	//Incoming Webhook URL
	url := "https://api.hipchat.com/v2/room/"+ room_id +"/notification?auth_token="+ auth_token

	//Form JSON payload to send to Slack
	json := `{"text": "Pod ` + action + ` in cluster: ` + pod.ObjectMeta.Name + `"}`

	//Post JSON payload to the Webhook URL
	client := http.Client{}

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(json))
	req.Header.Set("Content-Type", "application/json")

	_, err = client.Do(req)
	if err != nil {
		fmt.Println("Unable to reach the server.")
	}

}

func podCreated(obj interface{}) {
	notifySlack(obj, "created")
}

func podDeleted(obj interface{}) {
	notifySlack(obj, "deleted")
}

func watchPods(client *client.Client, store cache.Store) cache.Store {

	//Define what we want to look for (Pods)
	watchlist := cache.NewListWatchFromClient(client, "pods", api.NamespaceAll, fields.Everything())

	resyncPeriod := 30 * time.Minute

	//Setup an informer to call functions when the watchlist changes
	eStore, eController := framework.NewInformer(
		watchlist,
		&api.Pod{},
		resyncPeriod,
		framework.ResourceEventHandlerFuncs{
			AddFunc:    podCreated,
			DeleteFunc: podDeleted,
		},
	)

	//Run the controller as a goroutine
	go eController.Run(wait.NeverStop)
	return eStore
}

func main() {

	//Configure cluster info
	config := &restclient.Config{
		Host:     "https://xxx.yyy.zzz:443",
		Username: "kube",
		Password: "supersecretpw",
		Insecure: true,
	}

	//Create a new client to interact with cluster and freak if it doesn't work
	kubeClient, err := client.New(config)
	if err != nil {
		log.Fatalln("Client not created sucessfully:", err)
	}

	//Create a cache to store Pods
	var podsStore cache.Store

	//Watch for Pods
	podsStore = watchPods(kubeClient, podsStore)

	//Keep alive
	log.Fatal(http.ListenAndServe(":8080", nil))
}
