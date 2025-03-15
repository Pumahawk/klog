package main

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"time"
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

	var logsc []logChanMessage
	var logscNames []string
	logscr := make(chan logChanMessage)
	logscrAdded := make(chan bool)
	go func() {
		for lc := range logscr {
			sign := fmt.Sprintf("%s%s", lc.PodInfo.PodNamespace, lc.PodInfo.PodName)
			if !slices.Contains(logscNames, sign) {
				logsc = append(logsc, lc)
				logscNames = append(logscNames, sign)
				logDebug(fmt.Sprintf("Add pod %s/%s to logs watch list", lc.PodInfo.PodNamespace, lc.PodInfo.PodName))
			}
			logscrAdded <- true
		}
	}()
	wg := sync.WaitGroup{}
	cl := make(chan LogConfig)
	for i := 0; i < GlobalFlags.NumThread; i++ {
		go func() {
			for logConfig := range cl {
				func() {
					defer func() {
						if !GlobalFlags.Follow {
							wg.Done()
						}
					}()
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
						logDebug(fmt.Sprintf("Find pod %s/%s", *namespace, podName))
						lc := make(chan LogMessage, 200)
						logscr <- logChanMessage{Channel: lc, PodInfo: podInfo{PodName: podName, PodNamespace: *namespace}}
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
	wg.Add(1)
	go func() {
		if GlobalFlags.Follow {
			wg.Done()
		}
		for i := 0;; time.Sleep(5 * time.Second) {
			i++
			logDebug(fmt.Sprintf("Search pods %d", i))
			for _, logConfig := range config.Logs {
				if !GlobalFlags.Follow {
					wg.Add(1)
				}
				cl <- logConfig

			}
			if !GlobalFlags.Follow {
				break
			}
		}
		wg.Done()
	}()
	wg.Wait()
	logDebug(fmt.Sprintf("Start logging"))

	logStream := make(chan LogMessage, 200)
	go func() {
		defer close(logStream)
		if GlobalFlags.Sort {
			logDebug(fmt.Sprintf("Sort logging enabled"))
			LogSort(logsc, logStream)
		} else {
			logDebug(fmt.Sprintf("Sort logging disabled"))
			LogNotSort(&logsc, logStream)
		}
	}()
	for log := range logStream {
		fmt.Println(log.ToString())
	}
}

func LogSort(chans []logChanMessage, logStream chan LogMessage) {
	if len(chans) == 0 {
		return
	}

	logs := make([]*LogMessage, len(chans))

	BaseLoop:
	for {
		endOfLogs := true
		for i, c := range chans {
			if logs[i] == nil {
				if log, more := <- c.Channel; more {
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

func LogNotSort(chans *[]logChanMessage, logStream chan LogMessage) {
	gr := sync.WaitGroup{}

	var readedChannels []string;
	for i := 0;;time.Sleep(5 * time.Second) {
		i++
		logDebug(fmt.Sprintf("%d: Logging for channels", i))
		for j, logc := range *chans {
			sign := fmt.Sprintf("%s%s", logc.PodInfo.PodNamespace, logc.PodInfo.PodName)
			if slices.Contains(readedChannels, sign) {
				logDebug(fmt.Sprintf("Skip logging pod %d...", j))
				continue
			} else {
				logDebug(fmt.Sprintf("Start logging pod %d...", j))
				readedChannels = append(readedChannels, sign)
			}
			gr.Add(1)
			go func() {
				defer gr.Done()
				for log := range logc.Channel {
					logStream <- log
				}
			}()
		}
		if !GlobalFlags.Follow {
			break
		}
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

func matchName(confName string, nameFilter []string) bool {
	if len(nameFilter) > 0 && !slices.Contains(nameFilter, strings.Trim(confName, " ")) {
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

type logChanMessage struct {
	Channel chan LogMessage
	PodInfo podInfo
}

type podInfo struct {
	PodName string
	PodNamespace string
}

func logDebug(message string) {
	log.Println(message)
}
