```bash
# Install
go install github.com/pumahawk/klog@latest
```

```txt
Usage of klog:
  -burst int
        kubernates clients Burst (default 100)
  -config string
        Config path (default "config.yaml")
  -debug
        Debug logging
  -follow
        follow logs
  -info
        print config info
  -kubeconfig string
        Kubeconfig path
  -n-thread int
        Number thread load pods informations (default 10)
  -name string
        Name configuration
  -qps float
        kubernates clients QPS (default 100)
  -r-seconds int
        Refresh seconds (default 20)
  -since int
        since seconds (default -1)
  -since-time string
        Since time
  -sort
        sort log stream
  -t string
        Tags
  -tail int
        tail lines (default -1)
  -template string
        Go template
  -tor string
        Tags OR
```

## config.yaml

```yaml
baseTemplate: '{{ template "withJq" . }}'
templates:
  basicMessage: "{{ .Name }} {{ .Message }}"
  withJq: "{{ .Name }} {{ jq .Message .Vars.jqtemplate }}"
  toJson: |
    {{with $mm := jq .Message .Vars.jqRoot -}}
      {{- mapAdd $mm "name" $.Name | jsonEncode -}}
    {{- else -}}
      NONE
    {{- end -}}
vars:
  jqtemplate: >
    "\(.timestamp) \(.level) \(.message) \(."error.stack_trace" // "")"
  jqRoot: .
logs:
  - namespace: "iaa-dstest-data-provider"
    name: "test-data-provider/authentication-provider    "
    labels: "app.kubernetes.io/name=authentication-provider"
    tags: ["participant", "env:test", "dstest-data-provider", "type:data-provider", "name:authentication-provider"]
  - namespace: "iaa-dstest-data-provider"
    template: '{{ template "basicMessage" . }}'
    name: "test-data-provider/echo-backend               "
    labels: "app.kubernetes.io/name=echo-backend"
    tags: ["participant", "env:test", "dstest-data-provider", "type:data-provider", "name:echo-backend"]
  - namespace: "iaa-dstest-data-provider"
    name: "test-data-provider/tier1-gateway              "
    labels: "app.kubernetes.io/name=tier1-gateway"
    tags: ["participant", "env:test", "dstest-data-provider", "type:data-provider", "name:tier1-gateway"]
```
