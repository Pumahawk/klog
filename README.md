```bash
# Install
go install github.com/pumahawk/klog@latest
```

```txt
Usage of klog
  -config string
        Config path (default "config.json")
  -follow
        follow logs
  -kubeconfig string
        Kubeconfig path
  -since int
        since seconds
  -since-time string
        Since time
  -sort
        sort log stream
  -tail int
        tail lines
```

## config.json

```json
{
  "namespace": "default",
  "jqtemplate": "\"\\(.timestamp) \\(.level) \\(.message) \\(.\"error.stack_trace\" // \"\")\"",
  "logs": [
    {
      "name": "auth-provider  ",
      "labels": "app.kubernetes.io/name=authentication-provider"
    },
    {
      "name": "users-roles    ",
      "namespace": "custom",
      "jqtemplate": ".customjqtemplate",
      "labels": "app.kubernetes.io/name=users-roles"
    }
  ]
}
```
