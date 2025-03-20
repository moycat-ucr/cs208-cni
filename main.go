package main

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"os"
)

func main() {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("cannot get in-cluster config: %v", err)
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("cannot create rest client: %v", err)
	}
	instance, err := NewPlugin(client, os.Getenv("NODE_NAME"))
	if err != nil {
		log.Fatalf("cannot create cni plugin instance: %v", err)
	}
	if err := instance.Listen(); err != nil {
		log.Fatal(err)
	}
}
