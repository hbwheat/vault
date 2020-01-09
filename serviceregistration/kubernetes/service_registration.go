package kubernetes

import (
	"fmt"

	log "github.com/hashicorp/go-hclog"
	kubeHlpr "github.com/hashicorp/vault/sdk/helper/kubernetes"
	sr "github.com/hashicorp/vault/serviceregistration"
)

const (
	defaultVaultPodName = "vault"

	labelVaultVersion = "vault-version"
	labelActive       = "vault-ha-active"
	labelSealed       = "vault-ha-sealed"
	labelPerfStandby  = "vault-ha-perf-standby"
	labelInitialized  = "vault-ha-initialized"
)

func NewServiceRegistration(shutdownCh <-chan struct{}, config map[string]string, logger log.Logger, state *sr.State, _ string) (sr.ServiceRegistration, error) {
	client, err := kubeHlpr.NewLightWeightClient()
	if err != nil {
		return nil, err
	}

	// Perform an initial labelling of Vault as it starts up.
	namespace := config["namespace"]
	podName := config["pod_name"]
	if podName == "" {
		podName = defaultVaultPodName
	}
	pod, err := client.GetPod(namespace, podName)
	if err != nil {
		return nil, err
	}
	for label, value := range map[string]string{
		labelVaultVersion: state.VaultVersion,
		labelActive:       toString(state.IsActive),
		labelSealed:       toString(state.IsSealed),
		labelPerfStandby:  toString(state.IsPerformanceStandby),
		labelInitialized:  toString(state.IsInitialized),
	} {
		pod.Labels[label] = value
	}
	if err := client.UpdatePod(namespace, pod); err != nil {
		return nil, err
	}
	registration := &serviceRegistration{
		logger:    logger,
		podName:   podName,
		namespace: namespace,
		client: client,
	}

	// Run a background goroutine to leave labels in the final state we'd like
	// when Vault shuts down.
	go registration.onShutdown(shutdownCh)
	return registration, nil
}

type serviceRegistration struct {
	logger             log.Logger
	namespace, podName string
	client kubeHlpr.LightWeightClient
}

func (r *serviceRegistration) NotifyActiveStateChange(isActive bool) error {
	return r.client.PatchPodTag(r.namespace, r.podName, labelActive, toString(isActive))
}

func (r *serviceRegistration) NotifySealedStateChange(isSealed bool) error {
	return r.client.PatchPodTag(r.namespace, r.podName, labelSealed, toString(isSealed))
}

func (r *serviceRegistration) NotifyPerformanceStandbyStateChange(isStandby bool) error {
	return r.client.PatchPodTag(r.namespace, r.podName, labelPerfStandby, toString(isStandby))
}

func (r *serviceRegistration) NotifyInitializedStateChange(isInitialized bool) error {
	return r.client.PatchPodTag(r.namespace, r.podName, labelInitialized, toString(isInitialized))
}

func (r *serviceRegistration) onShutdown(shutdownCh <-chan struct{}) {
	<-shutdownCh
	pod, err := r.client.GetPod(r.namespace, r.podName)
	if err != nil {
		if r.logger.IsWarn() {
			r.logger.Warn(fmt.Sprintf("unable to get pod name %q in namespace %q on shutdown: %s", r.podName, r.namespace, err))
		}
		return
	}
	for label, value := range map[string]string{
		labelActive:      toString(false),
		labelSealed:      toString(true),
		labelPerfStandby: toString(false),
		labelInitialized: toString(false),
	} {
		pod.Labels[label] = value
	}
	if err := r.client.UpdatePod(r.namespace, pod); err != nil {
		if r.logger.IsWarn() {
			r.logger.Warn(fmt.Sprintf("unable to set final status on pod name %q in namespace %q on shutdown: %s", r.podName, r.namespace, err))
		}
		return
	}
}

// Converts a bool to "true" or "false".
func toString(b bool) string {
	return fmt.Sprintf("%t", b)
}
