{{/*
Expand the name of the chart.
*/}}
{{- define "operator-os.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "operator-os.fullname" -}}
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

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "operator-os.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "operator-os.labels" -}}
helm.sh/chart: {{ include "operator-os.chart" . }}
{{ include "operator-os.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "operator-os.selectorLabels" -}}
app.kubernetes.io/name: {{ include "operator-os.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Gateway selector labels
*/}}
{{- define "operator-os.gateway.selectorLabels" -}}
{{ include "operator-os.selectorLabels" . }}
app.kubernetes.io/component: gateway
{{- end }}

{{/*
Worker selector labels
*/}}
{{- define "operator-os.worker.selectorLabels" -}}
{{ include "operator-os.selectorLabels" . }}
app.kubernetes.io/component: worker
{{- end }}

{{/*
Service account name
*/}}
{{- define "operator-os.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "operator-os.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
