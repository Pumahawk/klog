package main

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
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

	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	logStreamChannels := make(chan []logChanMessage);
	go logStreamCrawlerThreadPool(logStreamChannels, config)

	startLogging(logStreamChannels)
	logDebug("End klog")
}

func GetKubernetesClientOrPanic() *kubernetes.Clientset {
	clientset, err := GetKubernetesClient()
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}
	return clientset
}

func logStreamCrawlerThreadPool(logStreamChannels chan []logChanMessage, config *Config) {
	logStream := make(chan []logChanMessageFunc)
	chanLogCongig := make(chan LogConfig)
	var logConfigs []LogConfig
	for _, logConfig := range config.Logs {
		if matchFlags(logConfig) {
			logConfigs = append(logConfigs, logConfig)
		}
	}

	go collectLogStreamChannels(logConfigs, logStream, logStreamChannels)

	for i := 0; i < GlobalFlags.NumThread; i++ {
		go logStreamCrawler(config, logStream, chanLogCongig)
	}
	for {
		logDebug("Looking for new pods")
		for _, conf := range logConfigs {
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

func collectLogStreamChannels(logConfigs []LogConfig, logStream chan []logChanMessageFunc, logStreamChannels chan []logChanMessage) {
	var alreadyFindPod []string
	if GlobalFlags.Follow {
		for lcs := range logStream {
			logDebug("Find log stream. Follow=true")
			for _, lc := range lcs {
				if sign := lc.sign(); !slices.Contains(alreadyFindPod, sign) {
					logDebug(fmt.Sprintf("New pod %s", sign))
					alreadyFindPod = append(alreadyFindPod, sign)
					logStreamChannels <- []logChanMessage{lc.toLogChanMessage()}
				}
			}
		}
	} else {
		var lcms []logChanMessage
		for i := 0; i < len(logConfigs); i++ {
			lcs := <- logStream
			logDebug("Find log stream. Follow=false")
			for _, lc := range lcs {
				if sign := lc.sign(); !slices.Contains(alreadyFindPod, sign) {
					logDebug(fmt.Sprintf("New pod %s", sign))
					alreadyFindPod = append(alreadyFindPod, sign)
					lcms = append(lcms, lc.toLogChanMessage())
				}
			}
		}
		logDebug("Create slice logStreamChannels")
		logStreamChannels <- lcms
		close(logStreamChannels)
		logDebug("Write logStreamChannels to stream")
	}
}

func logStreamCrawler(config *Config, logStreamChannels chan []logChanMessageFunc, chanLogConfig chan LogConfig) {
	clientset := GetKubernetesClientOrPanic()
	for logConfig := range chanLogConfig {
		var lcms []logChanMessageFunc
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
			channelGetter := func() chan LogMessage {
				lc := make(chan LogMessage, 200)
				go func(pod string, cfg LogConfig) {
					defer close(lc)
					err := StreamPodLogs(clientset, logConfig.Name, *namespace, pod, *jqTemplate, lc)
					if err != nil {
						log.Printf("Error handling logs for pod %s: %v", pod, err)
					}
				}(podName, logConfig)
				return lc
			}
			lgm := logChanMessageFunc{ChannelFunc: channelGetter, PodInfo: podInfo{PodName: podName, PodNamespace: *namespace}}
			lcms = append(lcms, lgm)
		}
		logStreamChannels <- lcms
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
				logDebug(fmt.Sprintf("Start track pod sort=true %s", c.sign()))
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

func logNotSort(logStreamChannels chan []logChanMessage) {
	wg := sync.WaitGroup{}
	for chans := range logStreamChannels {
		logDebug("Start logging channel")
		for _, c := range chans {
			wg.Add(1)
			go func() {
				defer wg.Done()
				logDebug(fmt.Sprintf("Start track pod sort=false %s", c.sign()))
				for log := range c.Channel {
					logMessage(&log)
				}
			}()
		}
	}
	wg.Wait()
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
	logDebug("Logging for channels")
	for i := 0;;time.Sleep(1 * time.Second) {
		i++
		for j, logc := range *chans {
			sign := fmt.Sprintf("%s%s", logc.PodInfo.PodNamespace, logc.PodInfo.PodName)
			if slices.Contains(readedChannels, sign) {
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

type logChanMessageFunc struct {
	ChannelFunc func() chan LogMessage
	PodInfo podInfo
}

type logChanMessage struct {
	Channel chan LogMessage
	PodInfo podInfo
}

func (lcm *logChanMessage) sign() string {
	return fmt.Sprintf("%s/%s", lcm.PodInfo.PodNamespace, lcm.PodInfo.PodName)
}

func (lcm *logChanMessageFunc) toLogChanMessage() logChanMessage {
	return logChanMessage {
		PodInfo: lcm.PodInfo,
		Channel: lcm.ChannelFunc(),
	}
}

func (lcm *logChanMessageFunc) sign() string {
	return fmt.Sprintf("%s/%s", lcm.PodInfo.PodNamespace, lcm.PodInfo.PodName)
}

type podInfo struct {
	PodName string
	PodNamespace string
}

func logDebug(message string) {
	if GlobalFlags.Debug {
		log.Println(message)
	}
}
