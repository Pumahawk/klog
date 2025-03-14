package main

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
)

func main() {
	ParseAndValidateGlobalFlags()

	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	if GlobalFlags.Info {
		printInfo(*config);
		return
	}

	clientset, err := GetKubernetesClient()
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	var logsc []chan LogMessage
	logscr := make(chan chan LogMessage)
	logscrAdded := make(chan bool)
	go func() {
		for lc := range logscr {
			logsc = append(logsc, lc)
			logscrAdded <- true
		}
	}()
	wg := sync.WaitGroup{}
	cl := make(chan LogConfig)
	for i := 0; i < GlobalFlags.NumThread; i++ {
		go func() {
			for logConfig := range cl {
				func() {
					defer wg.Done()
					if !matchName(logConfig.Name, GlobalFlags.Name) {
						return
					}

					if !matchTags(logConfig.Tags, GlobalFlags.Tags, GlobalFlags.TagsOr) {
						return
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
						return
					}

					for _, podName := range pods {
						lc := make(chan LogMessage, 200)
						logscr <- lc
						<- logscrAdded
						go func(pod string, cfg LogConfig) {
							defer close(lc)
							err := StreamPodLogs(clientset, logConfig.Name, *namespace, pod, *jqTemplate, lc)
							if err != nil {
								log.Printf("Error handling logs for pod %s: %v", pod, err)
							}
						}(podName, logConfig)
					}
				}()
			}
		}()
	}
	for _, logConfig := range config.Logs {
		wg.Add(1)
		cl <- logConfig
	}
	wg.Wait()
	close(logscr)
	close(cl)

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

func hasAnyTags(logTags []string, tags []string) bool {
	for _, tag := range tags {
		if slices.Contains(logTags, tag) {
			return true
		}
	}
	return false
}

func hasAllTags(logTags []string, tags []string) bool {
	for _, tag := range tags {
		if !slices.Contains(logTags, tag) {
			return false
		}
	}
	return true
}

func matchTags(logsTags, tags, tagOr []string) bool {
	if len(GlobalFlags.Tags) > 0 && !hasAllTags(logsTags, tags) {
		return false
	}

	if len(GlobalFlags.TagsOr) > 0 && !hasAnyTags(logsTags, tagOr) {
		return false
	}

	return true
}

func matchName(confName string, nameFilter *string) bool {
	if nameFilter != nil && *nameFilter != strings.Trim(confName, " ") {
		return false
	}
	return true
}

func printInfo(config Config) {
	var globalNamespace string
	if config.Namespace != nil {
		globalNamespace = *config.Namespace
	}
	var globalJqTemplate string
	if config.JQTemplate != nil {
		globalJqTemplate = *config.JQTemplate
	}

	fmt.Printf("Global namespace:  %s\n", globalNamespace)
	fmt.Printf("Global jqtemplate: %s\n", globalJqTemplate)

	var tags []string
	var names []string
	for _, logConf := range config.Logs {
		if !matchName(logConf.Name, GlobalFlags.Name) {
			continue
		}

		if !matchTags(logConf.Tags, GlobalFlags.Tags, GlobalFlags.TagsOr) {
			continue
		}

		names = append(names, strings.Trim(logConf.Name, " "))
		for _, tag := range logConf.Tags {
			if !slices.Contains(tags, tag) {
				tags = append(tags, tag)
			}
		}
	}

	fmt.Println("Tags:")
	for _, tag := range tags {
		fmt.Printf("\t%s\n", tag)
	}

	fmt.Println("Logs conf:")
	for _, name := range names {
		fmt.Printf("\t%s\n", name)
	}
}
