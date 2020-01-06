package kubernetes

import (
	"encoding/json"
	"sync"

	log "github.com/hashicorp/go-hclog"
	sr "github.com/hashicorp/vault/serviceregistration"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func Factory(_ map[string]string, _ log.Logger) (sr.ServiceRegistration, error) {
	// TODO - if possible, strip the client at the end because it has too many dependencies
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	pod, err := clientset.CoreV1().Pods("").Get("vault", metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}

	// TODO
	// To put or to patch, that is the question.
	// https://kubernetes.io/docs/tasks/run-application/update-api-object-kubectl-patch/

	// TODO this is the update method we could use but may result in the pod being torn down and rescheduled, which is bad
	//pod.Labels["foo"] = "bar"
	//if _, err := clientset.CoreV1().Pods("").Update(pod); err != nil {
	//	return nil, err
	//}

	// TODO this is likely the way to patch it but needs testing
	patch := map[string]string{
		"op": "add",
		"path": "/spec/template/metadata/labels/this",
		"value": "that",
	}
	data, err := json.Marshal([]interface{}{patch})
	if err != nil {
		return nil, err
	}
	if _, err := clientset.CoreV1().Pods("").Patch(pod.Name, types.JSONPatchType, data); err != nil {
		return nil, err
	}
	return &handler{}, nil
}

type handler struct {}

func (h *handler) NotifyActiveStateChange() error {
	// TODO
	return nil
}

func (h *handler) NotifySealedStateChange() error {
	// TODO
	return nil
}

func (h *handler) NotifyPerformanceStandbyStateChange() error {
	// TODO
	return nil
}

// TODO hopefully we won't need this
func (h *handler) RunServiceRegistration(
waitGroup *sync.WaitGroup, shutdownCh sr.ShutdownChannel, redirectAddr string,
activeFunc sr.ActiveFunction, sealedFunc sr.SealedFunction, perfStandbyFunc sr.PerformanceStandbyFunction) error {
	// TODO
	return nil
}