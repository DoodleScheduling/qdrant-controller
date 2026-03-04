# qdrant-controller helm chart

Installs the [qdrant-controller](https://github.com/DoodleScheduling/qdrant-controller).

## Installing the Chart

To install the chart with the release name `qdrant-controller`:

```console
helm upgrade --install qdrant-controller oci://ghcr.io/doodlescheduling/charts/qdrant-controller
```

This command deploys the qdrant-controller with the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

## Prometheus

The chart comes with a ServiceMonitor/PodMonitor for use with the [Prometheus Operator](https://github.com/coreos/prometheus-operator) which are disabled by default.
If you're not using the Prometheus Operator, you can populate the `podAnnotations` as below:

```yaml
podAnnotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "metrics"
  prometheus.io/path: "/metrics"
```

## Configuration

See customizing the chart before installing. To see all configurable options with detailed comments, visit the chart's values.yaml, or run the configuration command:

```sh
$ helm show values oci://ghcr.io/doodlescheduling/charts/qdrant-controller
```
