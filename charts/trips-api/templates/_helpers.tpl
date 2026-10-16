{{/*
Expand the name of the chart.
*/}}
{{- define "trips-api.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "trips-api.fullname" -}}
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

{{- define "trips-api.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "trips-api.labels" -}}
helm.sh/chart: {{ include "trips-api.chart" . }}
{{ include "trips-api.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "trips-api.selectorLabels" -}}
app.kubernetes.io/name: {{ include "trips-api.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: api
{{- end }}

{{- define "trips-api.postgresLabels" -}}
helm.sh/chart: {{ include "trips-api.chart" . }}
app.kubernetes.io/name: {{ include "trips-api.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: postgres
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "trips-api.postgresSelectorLabels" -}}
app.kubernetes.io/name: {{ include "trips-api.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: postgres
{{- end }}

{{- define "trips-api.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "trips-api.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "trips-api.postgresName" -}}
{{- printf "%s-postgres" (include "trips-api.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Host the API should dial. An explicit database.host wins. Otherwise use the bundled service.
*/}}
{{- define "trips-api.databaseHost" -}}
{{- if .Values.database.host }}
{{- .Values.database.host }}
{{- else }}
{{- include "trips-api.postgresName" . }}
{{- end }}
{{- end }}
