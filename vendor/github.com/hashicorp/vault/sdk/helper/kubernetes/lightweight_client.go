package kubernetes

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func NewLightWeightClient() (LightWeightClient, error) {
	conf, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return nil, err
	}
	return &lightWeightClient{}
}

// TODO audit all methods called on the client and
// add them to the interface here, then swap them out.
type LightWeightClient interface{}

type lightWeightClient struct{
	clientSet kubernetes.ClientSet
}
