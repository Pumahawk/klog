# Table of contents
- [Installation](#rocket-installation)
- [Helpful Aliases](#wrench-helpful-aliases)
- [Usage](#wrench-usage)
  - [Kubernetes & klog Configuration](#kubernetes--klog-configuration)
  - [Inspecting Configuration with `-info`](#inspecting-configuration-with--info)
  - [`-sort`: Sort Logs by Time](#-sort-sort-logs-by-time)
  - [`-follow`: Follow Logs in Real-Time](#-follow-follow-logs-in-real-time)
  - [Filtering Configurations with `-t`, `-tor`, and `-name`](#filtering-configurations-with--t--tor-and--name)
- [klog Configuration File](#page_facing_up-klog-configuration-file)
  - [Summary](#summary)
- [Valid configuration example](#valid-configuration-example)

**klog** is a Go-based CLI application designed to simplify log management in **Kubernetes** environments.

- **Targeted monitoring**: Define a list of **Pods** to monitor using **label selectors**.
- **Automatic ordering**: Aggregates and sorts logs from multiple resources.
- **Flexible formatting**: Supports **Go templates** and jq for customizable log display.
- **JSON decoding**: If logs are in **JSON format**, they can be automatically decoded for better readability.

Upon execution, **klog** reads a configuration file that specifies the resources to monitor, retrieves logs
from the selected pods, and presents them in a structured, easy-to-read format.

---

# :rocket: Installation

You can install **klog** using one of the following methods:

:one: **Download from GitHub Releases**

The easiest way to install **klog** is to download a precompiled binary from the [Releases](https://github.com/Pumahawk/klog/releases) page.

- Visit the [Releases](https://github.com/Pumahawk/klog/releases) page.
- Download the binary for your operating system.
- Move it to a directory in your `$PATH`

:two: **Install via** `go install`

If you have Go installed, you can install **klog** directly from the repository:

```bash
go install github.com/pumahawk/klog@latest
```

Make sure that `$GOPATH/bin` is in your `$PATH` so you can run klog from anywhere.

:three: **Clone & Build Manually**

If you prefer, you can clone the repository and build the binary yourself:

```bash
git clone https://github.com/Pumahawk/klog.git
cd klog
go install .
```

---

# :wrench: Helpful Aliases

To streamline the usage of **klog**, you can define some handy Bash aliases.

:hourglass: **Defaulting the** `-since-time` **Parameter**

This alias sets a default time for retrieving logs from the last 10 minutes:

:wrench: **Pre-configuring klog Execution**

This alias simplifies the command by automatically:

- Using the `KUBECONFIG` environment variable.
- Setting a default `-since-time` value using `kloggetdate`.
- Sorting the log output.

```bash
alias klog='klog -kubeconfig "$KUBECONFIG" -since-time "${KLTIME-$(kloggetdate)}" -sort'
```

With this setup, running **klog** without extra parameters will retrieve and sort logs from the last 10 minutes.

:calendar: **Customizing the Time Range**

If you want to specify a different time for `-since-time`, you can set the `KLTIME` environment variable:

```bash
export KLTIME='2025-03-20T10:30:00+00:00'
```

This will override the default 10-minute window and use the specified date/time instead.

# :wrench: Usage

## **Kubernetes & klog Configuration**

To use klog, you need to specify both the **Kubernetes configuration file** and the **klog configuration file**.

- The **Kubernetes config** file (`-kubeconfig`) is required to connect to your Kubernetes cluster. This is the same file you would use with `kubectl`.
- The klog config file (`-config`) defines which resources (Pods) to monitor, how logs should be processed, and other custom settings for the tool.

```bash
klog -kubeconfig /path/to/kubeconfig -config /path/to/klog-config.yaml
```

## **Inspecting Configuration with** `-info`

If you have a large configuration file and want to inspect the available tags and configurations without
actually fetching any logs, you can use the `-info` flag. This flag will display the configuration details, such
as the available tags and resource configurations, without starting the log extraction process.

```bash
klog -kubeconfig ... -config ... -info
```

This will output the configuration details based on the filters you specify, such as:

- `-t` or `-tor`: Filters based on tags.
- `-name`: Filters based on the resource name.

This is especially useful when working with large configurations, as it allows you to verify the available tags
and configuration settings before running the log extraction.

**Example Output:**

```bash
klog -kubeconfig ... -config ... -info -t env:dev,participant -tor name:authentication-provider,name:echo-backend

# Output:

# Global namespace:  
# Global jqtemplate: {{ template "withJq" . }}
# Tags:
#         participant
#         env:dev
#         dsdev-data-provider
#         type:data-provider
#         name:authentication-provider
#         name:echo-backend
#         dsdev-consumer
#         type:consumer
# Logs conf:
#         dev-data-provider/authentication-provider
#         dev-data-provider/echo-backend
#         dev-consumer/authentication-provider
#         dev-consumer/echo-backend
```

## `-sort`: **Sort Logs by Time**

The `-sort` flag allows you to sort the logs in chronological order as they are retrieved. This ensures that log
entries are displayed in the correct time sequence, which is particularly useful when dealing with logs from
multiple resources.

By default, logs are retrieved in the order they are fetched. When `-sort` is enabled, the logs will be ordered
by timestamp as they are collected.

**Example:**

```bash
klog -kubeconfig ... -config ... -sort
```

## `-follow`: **Follow Logs in Real-Time**

The `-follow` flag allows you to continuously monitor the logs in real-time, similar to `kubectl logs -f`. This
option is useful when you want to keep watching the logs as they are updated, without needing to restart the
command.

When `-follow` is used, **klog** will keep fetching new logs from the Kubernetes cluster as they are generated,
making it suitable for ongoing monitoring of your resources.

**Example:**

```bash
klog -kubeconfig ... -config ... -follow -sort=false -tail 1
```

> Important:  
> When using `-follow`, it is recommended to also use `-tail 0` and disable `-sort` (by setting `-sort=false`) since
> the logs will already be sorted and continuously updated as they come in.  
> Using both `-follow` and `-sort` simultaneously may lead to unexpected behavior due to the conflicting nature of
> sorting dynamic logs in real-time.

## **Filtering Configurations with** `-t`, `-tor`, and `-name`

**klog** allows you to filter configurations based on tags and resource names using the following flags:

`-t`: **Filter by Tags (AND Condition)**

The `-t` flag enables filtering based on specific tags. Only configurations that contain **-t** all the tags you specify
will be considered. Tags are specified as a comma-separated list.

`-tor`: **Filter by Tags (OR Condition)**

The `-tor` flag allows filtering based on tags with an **OR** condition. Only configurations that contain **at least**
**one** of the specified tags will be considered. Tags are also specified as a comma-separated list.

**Example Combined Usage:**

```bash
klog -kubeconfig ... -config ... -info -t env:dev,participant -tor name:authentication-provider,name:echo-backend
```

This command will return configurations that match at least one tag from **name:authentication-provider,name:echo-backend** and have the
all tags **env:dev,participant**

# :page_facing_up: klog Configuration File

The **klog** configuration file allows you to define templates, variables, and logs. These configurations control
how logs are fetched, formatted, and displayed. Here's a breakdown of the structure and each section of the file:

1. **baseTemplate**

The `baseTemplate` defines the default template used for log formatting. It is generally used to wrap other
templates and can call different templates to process log data.

```bash
baseTemplate: '{{ template "withJq" . }}'
```

- In this example, the base template calls another template named `"withJq"` and passes the current context (`.`) to it.

2. **templates**

The templates section allows you to define custom Go templates that can be used for formatting logs.
You can define as many templates as you want.

Each template has a unique name and its own logic for formatting log data.

**Example Templates:**

```yaml
templates:
  basicMessage: "{{ .Name }} {{ .Message }}"
  withJq: "{{ .Name }} {{ jq .Message .Vars.jqtemplate }}"
  toJson: |
    {{- with $mm := jq .Message .Vars.jqRoot -}}
      {{- mapAdd $mm "micro" $.Name | jsonEncode -}}
    {{- else -}}
      NONE
    {{- end -}}
```

- `basicMessage`: This template simply formats the `Name` and `Message` fields.
- `withJq`: This template uses `jq` (a command-line JSON processor) to filter and format the `Message` field using the `jqtemplate` variable.
- `toJson`: This template uses `jq` to transform the `Message` and then converts it to JSON, adding additional data (like the `Name`).

3. **vars**

The `vars` section is where you define static variables that can be used in your templates. These variables can
be referenced inside any template to manipulate or format log data.

```yaml
vars:
  jqtemplate: >
    "\(.timestamp) \(.level) \(.message) \(."error.stack_trace" // "")"
  jqRoot: .
```
- `jqtemplate`: A Go string template for formatting log data using the `jq` processor.
- `jqRoot`: Defines the root element for `jq` processing, in this case, the root is the message itself (`.`).

4. **logs**

The `logs` section defines the Kubernetes resources (i.e., Pods) that klog should monitor. Each log entry can
define:

- **namespace:** The Kubernetes namespace where the Pod is located.
- **name:** The name of the Pod or application.
- **labels:** The label selector used to filter the Pods in the Kubernetes cluster.
- **tags:** An optional list of tags that can be used for filtering log data in klog.

Each log entry can also override the default template specified in the `baseTemplate`.

**Example Log Configuration:**

```yaml
logs:
  - namespace: "iaa-dstest-data-provider"
    name: "test-data-provider/authentication-provider"
    labels: "app.kubernetes.io/name=authentication-provider"
    tags: ["participant", "env:test", "dstest-data-provider", "type:data-provider", "name:authentication-provider"]

```

- **namespace:** Specifies the Kubernetes namespace where the Pod is running (e.g., `"iaa-dstest-data-provider"`).
- **name:** The name of the Pod or application (e.g., `"test-data-provider/authentication-provider"`).
- **labels:** The label selector used to find the Pod in Kubernetes (e.g., `"app.kubernetes.io/name=authentication-provider"`).
- **tags:** A list of custom tags that can be used for filtering logs (e.g., `"participant"`, `"env:test"`). These tags can be passed with the `-t` or `-tor` parameters for filtering.

Each entry can optionally specify a `template` to override the default log template.

## Summary
The klog configuration file allows you to fine-tune the way logs are fetched, filtered, and formatted.
By customizing the `templates`, `vars`, and `logs` sections, you can tailor klog to suit your specific Kubernetes logging needs.

# Valid configuration example

```yaml
templates:
  basicMessage: "{{ .Name }} {{.PodName}} {{ .Message }}"
  basicMessageAndPodInfo: '{{ printf "%s/%s" .Namespace .PodName }} {{ .Message }}'
  withJq: "{{ .Name }} {{ jq .Message .Vars.jqtemplate }}"
  withJqAndPodInfo: '{{ printf "%s/%s" .Namespace .PodName }} {{ jq .Message .Vars.jqtemplate }}'
  toJson: |
    {{with $mm := jq .Message .Vars.jqRoot -}}
      {{- mapAdd $mm "name" $.Name | jsonEncode -}}
    {{- else -}}
      NONE
    {{- end -}}
  withTextMessage: '{{ template "basicMessage" . }}'
  withJsonMessage: '{{ template "withJq" . }}'
vars:
  jqtemplate: >
    "\(.timestamp) \(.level) \(.message) \(."error.stack_trace" // "")"
  jqRoot: .
logs:
  - namespace: "iaa-dstest-data-provider"
    template: '{{ template "withJsonMessage" . }}'
    name: "test-data-provider/authentication-provider    "
    labels: "app.kubernetes.io/name=authentication-provider"
    tags: ["participant", "env:test", "dstest-data-provider", "type:data-provider", "name:authentication-provider"]
```
