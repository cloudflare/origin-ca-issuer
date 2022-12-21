{{/*
Expand the name of the chart.
*/}}
{{- define "origin-ca-issuer.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 50 chars because some Kubernetes name fields are limited to 63 (by the DNS naming spec),
and we append in some places.
*/}}
{{- define "origin-ca-issuer.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 50 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 50 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 50 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "origin-ca-issuer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "origin-ca-issuer.serviceAccountName" -}}
{{- if .Values.controller.serviceAccount.create -}}
    {{ default (include "origin-ca-issuer.fullname" .) .Values.controller.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.controller.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Namespace for all resources to be installed into
If not defined in values file then the helm release namespace is used
By default this is not set so the helm release namespace will be used
This gets around an problem within helm discussed here
https://github.com/helm/helm/issues/5358
*/}}
{{- define "origin-ca-issuer.namespace" -}}
    {{ .Values.namespace | default .Release.Namespace }}
{{- end -}}
