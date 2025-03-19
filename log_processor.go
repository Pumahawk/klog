package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"text/template"

	"github.com/itchyny/gojq"
)

func JsonLogProcessEncode(value any) (string, error) {
	var result bytes.Buffer
	if err := json.NewEncoder(&result).Encode(value); err != nil {
		return "", err
	}
	return result.String(), nil
}

func JsonLogProcessDeconder(jsonStr string) (map[string]any, error) {
	var result map[string]any
	bf := bytes.NewBufferString(jsonStr)
	if err := json.NewDecoder(bf).Decode(&result); err != nil {
		return nil, err
	} else {
		return result, nil
	}
}

func MapAdd(mp map[string]any, key string, value any) map[string]any {
	if mp == nil {
		mp = make(map[string]any)
	}
	mp[key] = value
	return mp
}

func ProcessLogWithJQ(jsonStr, jqTemplate string) (any, error) {
	var logObj map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &logObj); err != nil {
		return "", fmt.Errorf("errore nel parsing del log: %v", err)
	}
	query, err := gojq.Parse(jqTemplate)
	if err != nil {
		return "", fmt.Errorf("errore nel parsing della query jq: %v", err)
	}

	iter := query.Run(logObj)
	var result any
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return "", fmt.Errorf("errore nell'esecuzione di jq: %v", err)
		}
		result = v
	}

	return result, nil
}

func getLogMessage(name string, namespace string, podName string, vars map[string]any, log string) LogMessage {
	re := regexp.MustCompile(`^(\S+)\s(.*)`)
	match := re.FindStringSubmatch(log)
	return LogMessage{
		Name:      name,
		Namespace: namespace,
		PodName:   podName,
		Time:      match[1],
		Message:   match[2],
		Vars:      vars,
	}
}

type LogProcessor struct {
	Template       string
	vars           map[string]any
	templateEngine *template.Template
}

func LogProcessorNew(templateMessage string, vars map[string]any) (*LogProcessor, error) {
	funcMap := template.FuncMap{
		"jq":         ProcessLogWithJQ,
		"jsonDecode": JsonLogProcessDeconder,
		"jsonEncode": JsonLogProcessEncode,
		"mapAdd":     MapAdd,
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
		vars:           vars,
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
	Vars      map[string]any
}

func (log *LogMessage) ToString() string {
	return fmt.Sprint(log.Message)

}
