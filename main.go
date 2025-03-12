package main

import (
	"fmt"
	"log"
	"slices"
	"sync"
)

func main() {
	ParseAndValidateGlobalFlags()

	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	clientset, err := GetKubernetesClient()
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	var logsc []chan LogMessage
	for _, logConfig := range config.Logs {
		if len(GlobalFlags.Tags) > 0 && !hasAllTags(logConfig.Tags, GlobalFlags.Tags) {
			continue
		}

		jqTemplate := config.JQTemplate;
		if jqTemplate == nil {
			jqTemplate = logConfig.JQTemplate
		}
		namespace := config.Namespace;
		if namespace == nil {
			namespace = logConfig.Namespace
		}
		pods, err := GetPodsByLabel(clientset, *namespace, logConfig.Labels)
		if err != nil {
			log.Printf("Error retrieving pods for namespace %s and labels %s: %v", *namespace, logConfig.Labels, err)
			continue
		}

		for _, podName := range pods {
			lc := make(chan LogMessage, 200)
			logsc = append(logsc, lc)
			go func(pod string, cfg LogConfig) {
				defer close(lc)
				err := StreamPodLogs(clientset, logConfig.Name, *namespace, pod, *jqTemplate, lc)
				if err != nil {
					log.Printf("Error handling logs for pod %s: %v", pod, err)
				}
			}(podName, logConfig)
		}
	}

	logStream := make(chan LogMessage, 200)
	go func() {
		defer close(logStream)
		if GlobalFlags.Sort {
			LogSort(logsc, logStream)
		} else {
			LogNotSort(logsc, logStream)
		}
	}()
	for log := range logStream {
		fmt.Println(log.ToString())
	}
}

func LogSort(chans []chan LogMessage, logStream chan LogMessage) {
	if len(chans) == 0 {
		return
	}

	logs := make([]*LogMessage, len(chans))

	BaseLoop:
	for {
		endOfLogs := true
		for i, c := range chans {
			if logs[i] == nil {
				if log, more := <- c; more {
					logs[i] = &log
					endOfLogs = false
				}
			} else {
				endOfLogs = false
			}
			if l := logs[i]; l != nil {
			}
		}
		if endOfLogs {
			break BaseLoop
		}

		lowerI := 0
		for i, log := range logs[1:] {
			i = i + 1
			if logs[lowerI] == nil {
				lowerI = i
			} else {
				if log != nil && log.Time < logs[lowerI].Time {
					lowerI = i
				}
			}
		}

		if logs[lowerI] != nil {
			logStream <- *logs[lowerI]
			logs[lowerI] = nil
		}
	}
}

func LogNotSort(chans []chan LogMessage, logStream chan LogMessage) {
	gr := sync.WaitGroup{}
	for _, logc := range chans {
		gr.Add(1)
		go func() {
			defer gr.Done()
			for log := range logc {
				logStream <- log
			}
		}()
	}
	gr.Wait()
}

func hasAllTags(logTags []string, tags []string) bool {
	for _, tag := range tags {
		if !slices.Contains(logTags, tag) {
			return false
		}
	}
	return true
}
