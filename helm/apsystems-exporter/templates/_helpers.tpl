{{/*
Expand the name of the chart.
*/}}
{{- define "apsystems-exporter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "apsystems-exporter.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "apsystems-exporter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "apsystems-exporter.labels" -}}
helm.sh/chart: {{ include "apsystems-exporter.chart" . }}
{{ include "apsystems-exporter.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "apsystems-exporter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "apsystems-exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "apsystems-exporter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "apsystems-exporter.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "apsystems-exporter.secretName" -}}
{{- if .Values.apsystems.existingSecret }}
{{- .Values.apsystems.existingSecret }}
{{- else if .Values.apsystems.createSecret }}
{{- include "apsystems-exporter.fullname" . }}
{{- else }}
{{- fail "apsystems.existingSecret is required unless apsystems.createSecret is true" }}
{{- end }}
{{- end }}

{{- define "apsystems-exporter.layoutConfigMapName" -}}
{{- if .Values.panelsLayout.existingConfigMap }}
{{- .Values.panelsLayout.existingConfigMap }}
{{- else if .Values.panelsLayout.enabled }}
{{- printf "%s-layout" (include "apsystems-exporter.fullname" .) }}
{{- else }}
{{- "" }}
{{- end }}
{{- end }}

{{- define "apsystems-exporter.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}
