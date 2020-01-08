package kubernetes

// TODO make every effort to avoid using client-go.
// TODO
// To put or to patch, that is the question.
// https://kubernetes.io/docs/tasks/run-application/update-api-object-kubectl-patch/
import (
	"encoding/json"
	"fmt"

	log "github.com/hashicorp/go-hclog"
	sr "github.com/hashicorp/vault/serviceregistration"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	defaultVaultPodName = "vault"

	labelVaultVersion = "vault-version"
	labelActive       = "vault-ha-active"
	labelSealed       = "vault-ha-sealed"
	labelPerfStandby  = "vault-ha-perf-standby"
	labelInitialized  = "vault-ha-initialized"
)

func Factory(shutdownCh <-chan struct{}, config map[string]string, logger log.Logger, state *sr.State, _ string) (sr.ServiceRegistration, error) {
	conf, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return nil, err
	}

	// Perform an initial labelling of Vault as it starts up.
	podName := config["pod_name"]
	if podName == "" {
		podName = defaultVaultPodName
	}
	namespace := config["namespace"]
	pod, err := clientSet.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	pod.SetLabels(map[string]string{
		labelVaultVersion: state.VaultVersion,
		labelActive:       toString(state.IsActive),
		labelSealed:       toString(state.IsSealed),
		labelPerfStandby:  toString(state.IsPerformanceStandby),
		labelInitialized:  toString(state.IsInitialized),
	})
	if _, err := clientSet.CoreV1().Pods(namespace).Update(pod); err != nil {
		return nil, err
	}
	registration := &serviceRegistration{
		logger:    logger,
		podName:   podName,
		namespace: namespace,
		clientSet: clientSet,
	}

	// Run a background goroutine to leave labels in the final state we'd like
	// when Vault shuts down.
	go registration.onShutdown(shutdownCh)
	return registration, nil
}

type serviceRegistration struct {
	logger             log.Logger
	podName, namespace string
	clientSet          *kubernetes.Clientset
}

func (r *serviceRegistration) NotifyActiveStateChange(isActive bool) error {
	return r.patchTag(labelActive, isActive)
}

func (r *serviceRegistration) NotifySealedStateChange(isSealed bool) error {
	return r.patchTag(labelSealed, isSealed)
}

func (r *serviceRegistration) NotifyPerformanceStandbyStateChange(isStandby bool) error {
	return r.patchTag(labelPerfStandby, isStandby)
}

func (r *serviceRegistration) NotifyInitializedStateChange(isInitialized bool) error {
	return r.patchTag(labelInitialized, isInitialized)
}

func (r *serviceRegistration) onShutdown(shutdownCh <-chan struct{}) {
	<-shutdownCh
	pod, err := r.clientSet.CoreV1().Pods(r.namespace).Get(r.podName, metav1.GetOptions{})
	if err != nil {
		if r.logger.IsWarn() {
			r.logger.Warn(fmt.Sprintf("unable to get pod name %q in namespace %q on shutdown: %s", r.podName, r.namespace, err))
		}
		return
	}
	pod.SetLabels(map[string]string{
		labelActive:      toString(false),
		labelSealed:      toString(true),
		labelPerfStandby: toString(false),
		labelInitialized: toString(false),
	})
	if _, err := r.clientSet.CoreV1().Pods(r.namespace).Update(pod); err != nil {
		if r.logger.IsWarn() {
			r.logger.Warn(fmt.Sprintf("unable to set final status on pod name %q in namespace %q on shutdown: %s", r.podName, r.namespace, err))
		}
		return
	}
}

func (r *serviceRegistration) patchTag(key string, value bool) error {
	patch := map[string]string{
		"op":    "add",
		"path":  "/spec/template/metadata/labels/" + key,
		"value": toString(value),
	}
	data, err := json.Marshal([]interface{}{patch})
	if err != nil {
		return err
	}
	if _, err := r.clientSet.CoreV1().Pods(r.namespace).Patch(r.podName, types.JSONPatchType, data); err != nil {
		return err
	}
	return nil
}

// Converts a bool to "true" or "false".
func toString(b bool) string {
	return fmt.Sprintf("%t", b)
}
