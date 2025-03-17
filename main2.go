package main

import (
	"fmt"
	"log"
	"time"

	"k8s.io/client-go/kubernetes"
)

func main() {
	// load flags
	ParseAndValidateGlobalFlags()
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	if GlobalFlags.Info {
		printInfo(*config);
		return
	}

	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	logStreamChannels := make(chan []logChanMessage);
	go logStreamCrawlerThreadPool(logStreamChannels, config)

	startLogging(logStreamChannels)

	// load config
}

func GetKubernetesClientOrPanic() *kubernetes.Clientset {
	clientset, err := GetKubernetesClient()
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}
	return clientset
}

func logStreamCrawlerThreadPool(logStreamChannels chan []logChanMessage, config *Config) {
	logStream := make(chan logChanMessage)
	chanLogCongig := make(chan LogConfig)
	var logConfigs []LogConfig
	for _, logConfig := range config.Logs {
		if matchFlags(logConfig) {
			logConfigs = append(logConfigs, logConfig)
		}
	}
	go func() {
		if GlobalFlags.Follow {
			for lc := range logStream {
				logDebug("Find log stream. Follow=true")
				logStreamChannels <- []logChanMessage{lc}
			}
		} else {
			var lcms []logChanMessage
			for i := 0; i < len(logConfigs); i++ {
				lc := <- logStream
				logDebug("Find log stream. Follow=false")
				lcms = append(lcms, lc)
			}
			logDebug("Create slice logStreamChannels")
			logStreamChannels <- lcms
			close(logStreamChannels)
			logDebug("Write logStreamChannels to stream")
		}
	}()
	for i := 0; i < GlobalFlags.NumThread; i++ {
		go logStreamCrawler(config, logStream, chanLogCongig)
	}
	for {
		for i, conf := range logConfigs {
			logDebug(fmt.Sprintf("Start stream log config %d", i))
			chanLogCongig <- conf
		}
		if !GlobalFlags.Follow {
			break
		} else {
			time.Sleep(time.Duration(GlobalFlags.RefreshSeconds) * time.Second)
		}
	}
	logDebug("End stream all log configs")
}

func logStreamCrawler(config *Config, logStreamChannels chan logChanMessage, chanLogConfig chan LogConfig) {
	clientset := GetKubernetesClientOrPanic()
	for logConfig := range chanLogConfig {
		logDebug("Read log config")

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
			logStreamChannels <- logChanMessage{Channel: lc, PodInfo: podInfo{PodName: podName, PodNamespace: *namespace}}
			go func(pod string, cfg LogConfig) {
				defer close(lc)
				err := StreamPodLogs(clientset, logConfig.Name, *namespace, pod, *jqTemplate, lc)
				if err != nil {
					log.Printf("Error handling logs for pod %s: %v", pod, err)
				}
			}(podName, logConfig)
		}
		logDebug("End log config")
	}
}

func startLogging(logStreamChannels chan []logChanMessage) {
	logDebug("Start logging")
	if GlobalFlags.Sort {
		logSort(logStreamChannels)
	} else {
		logNotSort(logStreamChannels)
	}
}

func logSort(logStreamChannels chan []logChanMessage) {
	logs := make([]*LogMessage, 0)

	var chans []logChanMessage

	logDebug("Start sort logging. Waiting for first logStreamChannels")
	BaseLoop:
	for newChans := <- logStreamChannels;;newChans = func() []logChanMessage {
		select {
		case cs := <- logStreamChannels:
			return cs
		default:
			return nil
		}
	}(){
		if newChans != nil {
			logDebug("Find new channels")
			for _, c := range newChans {
				chans = append(chans, c)
				logs = append(logs, nil)
			}
		}
		
		if len(chans) == 0 {
			continue
		}
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
			logMessage(logs[lowerI])
			logs[lowerI] = nil
		}
	}
}

func logNotSort(streamLogChannels chan []logChanMessage) {
	for chans := range streamLogChannels {
		for _, c := range chans {
			go func() {
				for log := range c.Channel {
					logMessage(&log)
				}
			}()
		}
	}
}

func logMessage(log *LogMessage) {
	if log != nil {
		fmt.Println(log.ToString())
	}
}

func matchFlags(logConfig LogConfig) bool {
	if !matchName(logConfig.Name, GlobalFlags.Name) {
		return false
	}
	if !matchTags(logConfig.Tags, GlobalFlags.Tags, GlobalFlags.TagsOr) {
		return false
	}
	return true
}
