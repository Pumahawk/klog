package main

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

type Flags struct {
	Burst          int
	ConfigPath     string
	Debug          bool
	Follow         bool
	Info           bool
	KubeconfigPath string
	Name           []string
	NumThread      int
	QPS            float64
	RefreshSeconds int
	SinceSeconds   *int64
	SinceTime      *time.Time
	Sort           bool
	Tags           []string
	TagsOr         []string
	TailLines      *int64
	Template       *string
}

var GlobalFlags = Flags{}

func ParseAndValidateGlobalFlags() error {
	flag.StringVar(&GlobalFlags.KubeconfigPath, "kubeconfig", "", "Kubeconfig path")
	flag.StringVar(&GlobalFlags.ConfigPath, "config", "config.yaml", "Config path")
	flag.BoolVar(&GlobalFlags.Follow, "follow", false, "follow logs")
	flag.BoolVar(&GlobalFlags.Sort, "sort", false, "sort log stream")
	flag.BoolVar(&GlobalFlags.Info, "info", false, "print config info")
	flag.BoolVar(&GlobalFlags.Debug, "debug", false, "Debug logging")
	flag.IntVar(&GlobalFlags.NumThread, "n-thread", 10, "Number thread load pods informations")
	flag.IntVar(&GlobalFlags.RefreshSeconds, "r-seconds", 20, "Refresh seconds")
	flag.Float64Var(&GlobalFlags.QPS, "qps", 100, "kubernates clients QPS")
	flag.IntVar(&GlobalFlags.Burst, "burst", 100, "kubernates clients Burst")
	nameFlag := flag.String("name", "", "Name configuration")
	templateFlag := flag.String("template", "", "Go template")
	tailLinesFlag := flag.Int64("tail", -1, "tail lines")
	sinceSeconds := flag.Int64("since", -1, "since seconds")
	sinceTimeFlag := flag.String("since-time", "", "Since time")
	tagsFlag := flag.String("t", "", "Tags")
	tagsOrFlag := flag.String("tor", "", "Tags OR")

	flag.Parse()

	if *templateFlag != "" {
		GlobalFlags.Template = templateFlag
	}

	if *nameFlag != "" {
		GlobalFlags.Name = strings.Split(*nameFlag, ",")
	}

	if *sinceTimeFlag != "" {
		var time, err = time.Parse(time.RFC3339, *sinceTimeFlag)
		if err != nil {
			return fmt.Errorf("Unable to read since-time, %s", *sinceTimeFlag)
		}
		GlobalFlags.SinceTime = &time

	}

	if *tailLinesFlag != -1 {
		GlobalFlags.TailLines = tailLinesFlag
	}

	if *sinceSeconds != -1 {
		GlobalFlags.SinceSeconds = sinceSeconds
	}

	if *tagsFlag != "" {
		GlobalFlags.Tags = strings.Split(*tagsFlag, ",")
	}

	if *tagsOrFlag != "" {
		GlobalFlags.TagsOr = strings.Split(*tagsOrFlag, ",")
	}

	return nil
}
