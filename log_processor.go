package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"text/template"

	"github.com/itchyny/gojq"
)

func ProcessLogWithJQ(jsonStr, jqTemplate string) (string, error) {
	var logObj map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &logObj); err != nil {
		return "", fmt.Errorf("errore nel parsing del log: %v", err)
	}
	query, err := gojq.Parse(jqTemplate)
	if err != nil {
		return "", fmt.Errorf("errore nel parsing della query jq: %v", err)
	}

	iter := query.Run(logObj)
	var result string
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return "", fmt.Errorf("errore nell'esecuzione di jq: %v", err)
		}
		result = fmt.Sprintf("%v", v)
	}

	return result, nil
}

func getLogMessage(name string, namespace string, podName string, log string) LogMessage {
	re := regexp.MustCompile(`^(\S+)\s(.*)`)
	match := re.FindStringSubmatch(log)
	return LogMessage{
		Name:      name,
		Namespace: namespace,
		PodName:   podName,
		Time:      match[1],
		Message:   match[2],
	}
}

type LogProcessor struct {
	Template       string
	templateEngine *template.Template
}

func LogProcessorNew(templateMessage string) (*LogProcessor, error) {
	funcMap := template.FuncMap{
		"jq": ProcessLogWithJQ,
	}
	tpl := template.New("message")
	tpl = tpl.Funcs(funcMap)
	tpl, err := tpl.Parse(templateMessage)
	if err != nil {
		return nil, fmt.Errorf("Unable to Log Processor: %v", err)
	}
	return &LogProcessor{
		Template:       templateMessage,
		templateEngine: tpl,
	}, nil
}

func (pr *LogProcessor) Log(lm LogMessage) (string, error) {
	var buf bytes.Buffer
	err := pr.templateEngine.Execute(&buf, lm)
	if err != nil {
		return "", fmt.Errorf("Unable execute message. %v", err)
	}
	return buf.String(), nil
}

type LogMessage struct {
	Name      string
	Namespace string
	PodName   string
	Time      string
	Message   string
}

func (log *LogMessage) ToString() string {
	return fmt.Sprint(log.Message)
}
