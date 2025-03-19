package main

import (
	"bufio"
	"context"
	"fmt"
	"log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func GetKubernetesClient() (*kubernetes.Clientset, error) {
	var kubeconfig = GlobalFlags.KubeconfigPath
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("errore nel caricamento del kubeconfig: %v", err)
	}
	config.QPS = float32(GlobalFlags.QPS)
	config.Burst = GlobalFlags.Burst
	return kubernetes.NewForConfig(config)
}

func GetPodsByLabel(clientset *kubernetes.Clientset, namespace, labelSelector string) ([]string, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	var podNames []string
	for _, pod := range pods.Items {
		podNames = append(podNames, pod.Name)
	}

	return podNames, nil
}

func StreamPodLogs(clientset *kubernetes.Clientset, name, namespace, podName, template string, logchan chan LogMessage, vars map[string]any, templates map[string]string) error {
	podLogOptions := generatePodLogOptions()
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOptions)

	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return fmt.Errorf("Unable to retrieve logs %s: %v", podName, err)
	}
	defer podLogs.Close()

	scanner := bufio.NewScanner(podLogs)
	lp, err := LogProcessorNew(template, vars, templates)
	if err != nil {
		log.Fatalf("Unable to processor: %v", err)
	}
	for scanner.Scan() {
		logLine := getLogMessage(name, namespace, podName, vars, scanner.Text())
		formatted, err := lp.Log(logLine)
		if err == nil {
			logLine.Message = formatted
		} else {
			logDebug(err.Error())
		}
		logchan <- logLine
	}

	return scanner.Err()
}

func generatePodLogOptions() v1.PodLogOptions {
	podLogOptions := v1.PodLogOptions{
		Follow:       GlobalFlags.Follow,
		Timestamps:   true,
		TailLines:    GlobalFlags.TailLines,
		SinceSeconds: GlobalFlags.SinceSeconds,
	}

	if GlobalFlags.SinceTime != nil {
		time := metav1.NewTime(*GlobalFlags.SinceTime)
		podLogOptions.SinceTime = &time
	}

	return podLogOptions
}
