```bash
# Install
go install github.com/pumahawk/klog@latest
```

```txt
Usage of klog:
  -burst int
        kubernates clients Burst (default 100)
  -config string
        Config path (default "config.json")
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
  -tor string
        Tags OR
```

## config.json

```json
{
  "namespace": "default",
  "template": "{{ .Name }} {{jq .Message .Vars.jqtemplate }}",
  "vars": {
    "jqtemplate": "\"\\(.timestamp) \\(.level) \\(.message) \\(.\"error.stack_trace\" // \"\")\""
  },
  "logs": [
    {
      "name": "auth-provider  ",
      "labels": "app.kubernetes.io/name=authentication-provider",
      "tags": "tag1, tag2"
    },
    {
      "name": "users-roles    ",
      "namespace": "custom",
      "template": "{{ ... }}",
      "labels": "app.kubernetes.io/name=users-roles",
      "tags": "tag1"
    }
  ]
}
```
