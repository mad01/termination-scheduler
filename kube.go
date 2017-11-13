package main

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"encoding/json"
	"fmt"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

func k8sGetClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func k8sGetClient(kubeconfig string) (*kubernetes.Clientset, error) {
	config, err := k8sGetClientConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	// Construct the Kubernetes client
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func newKube(kubeconfig, fromCronTime, toCronTime string) (*Kube, error) {
	client, err := k8sGetClient(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %v", err.Error())
	}
	a, err := newAnnotations(fromCronTime, toCronTime)
	if err != nil {
		return nil, fmt.Errorf("failed to create new kube: %v", err.Error())
	}
	k := Kube{
		client:      client,
		annotations: a,
	}
	return &k, nil
}

// Kube kubernetes connection struct
type Kube struct {
	client      *kubernetes.Clientset
	annotations *Annotations
}

func (k *Kube) getNodes(selector string) (*v1.NodeList, error) {
	nodes, err := k.client.Core().Nodes().List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes list %v", err.Error())
	}
	return nodes, nil
}

// TODO: implement to take maintain window from to see issue on terminator
// TODO: implement with patch do add annotations
func (k *Kube) annotateNodes(nodeList *v1.NodeList) error {
	return nil
}

func (k *Kube) annotatePatchNode(node *v1.Node) error {
	oldData, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to Marshal old node %v", err.Error())
	}

	node = node.DeepCopy()
	node.Annotations[nodeAnnotationFromWindow] = fmt.Sprintf("%v", k.annotations.timeWindow.from)
	node.Annotations[nodeAnnotationToWindow] = fmt.Sprintf("%v", k.annotations.timeWindow.to)
	node.Annotations[nodeAnnotationReboot] = fmt.Sprintf("%v", k.annotations.reboot)

	newData, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to Marshal new node %v", err.Error())
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, v1.Node{})
	if err != nil {
		return fmt.Errorf("failed to create patch %v", err.Error())
	}

	_, err = k.client.Core().Nodes().Patch(node.GetName(), types.StrategicMergePatchType, patchBytes)
	if err != nil {
		return fmt.Errorf("failed to patch node %v %v", node.GetName(), err.Error())
	}

	return nil
}
