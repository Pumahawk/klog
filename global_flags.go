package main

import (
	"flag"
	"fmt"
	"time"
)

type Flags struct {
	KubeconfigPath string
	ConfigPath     string
	SinceTime      *time.Time
	Follow         bool
	Sort           bool
	TailLines      *int64
	SinceSeconds   *int64
}

var GlobalFlags = Flags{}

func ParseAndValidateGlobalFlags() error {
	flag.StringVar(&GlobalFlags.KubeconfigPath, "kubeconfig", "", "Kubeconfig path")
	flag.StringVar(&GlobalFlags.ConfigPath, "config", "config.json", "Config path")
	flag.BoolVar(&GlobalFlags.Follow, "follow", false, "follow logs")
	flag.BoolVar(&GlobalFlags.Sort, "sort", false, "sort log stream")
	tailLinesFlag := flag.Int64("tail", -1, "tail lines")
	sinceSeconds := flag.Int64("since", -1, "since seconds")
	sinceTimeFlag := flag.String("since-time", "", "Since time (Optional)")

	flag.Parse()

	if *sinceTimeFlag != "" {
		var time, err = time.Parse(time.RFC3339, *sinceTimeFlag);
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

	return nil
}
