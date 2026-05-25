# Helm chart

Install:

```bash
helm install apsystems ./helm/apsystems-exporter -f my-values.yaml
```

See [values.yaml](values.yaml) and the [project README](../README.md).

## Required values

- `apsystems.sid`
- `apsystems.ecuId`
- `apsystems.existingSecret` (recommended) **or** `apsystems.createSecret` + inline credentials

## Prometheus Operator

```yaml
serviceMonitor:
  enabled: true
  additionalLabels:
    release: monitoring
```
