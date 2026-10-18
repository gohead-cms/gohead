{{/* Chart name. */}}
{{- define "gohead.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Release-qualified name, e.g. "my-release-gohead". */}}
{{- define "gohead.fullname" -}}
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

{{- define "gohead.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Common labels. */}}
{{- define "gohead.labels" -}}
helm.sh/chart: {{ include "gohead.chart" . }}
app.kubernetes.io/part-of: gohead
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
{{- end }}

{{/*
Selector labels of a component. Call with (dict "ctx" $ "component" "api").
*/}}
{{- define "gohead.selectorLabels" -}}
app.kubernetes.io/name: {{ include "gohead.name" .ctx }}
app.kubernetes.io/instance: {{ .ctx.Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{- define "gohead.componentLabels" -}}
{{ include "gohead.labels" .ctx }}
{{ include "gohead.selectorLabels" . }}
{{- end }}

{{- define "gohead.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "gohead.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "gohead.image" -}}
{{- printf "%s:%s" .Values.image.repository (default .Chart.AppVersion .Values.image.tag) }}
{{- end }}

{{- define "gohead.adminImage" -}}
{{- printf "%s:%s" .Values.admin.image.repository (default .Chart.AppVersion .Values.admin.image.tag) }}
{{- end }}

{{- define "gohead.secretName" -}}
{{- default (printf "%s-secrets" (include "gohead.fullname" .)) .Values.existingSecret }}
{{- end }}

{{- define "gohead.redisAddress" -}}
{{- if .Values.redis.address }}
{{- .Values.redis.address }}
{{- else if .Values.redis.enabled }}
{{- printf "%s-redis:6379" (include "gohead.fullname" .) }}
{{- else }}
{{- fail "redis: set redis.address or redis.enabled=true (GoHead needs Redis)" }}
{{- end }}
{{- end }}

{{/* Public URL of the API, for the admin UI. */}}
{{- define "gohead.apiUrl" -}}
{{- if .Values.admin.apiUrl }}
{{- .Values.admin.apiUrl }}
{{- else if and .Values.api.ingress.enabled .Values.api.ingress.hosts }}
{{- $host := (first .Values.api.ingress.hosts).host }}
{{- printf "%s://%s" (ternary "https" "http" (gt (len .Values.api.ingress.tls) 0)) $host }}
{{- else }}
{{- /* For `kubectl port-forward svc/<release>-gohead-api 8080:80`. */}}
{{- print "http://localhost:8080" }}
{{- end }}
{{- end }}

{{/*
Environment shared by every GoHead container (API and background processes).
*/}}
{{- define "gohead.env" -}}
envFrom:
  - configMapRef:
      name: {{ include "gohead.fullname" . }}-config
  - secretRef:
      name: {{ include "gohead.secretName" . }}
{{- with .Values.extraEnv }}
env:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- end }}

{{/* Pod annotations that roll the pods when configuration changes. */}}
{{- define "gohead.configChecksums" -}}
checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
{{- if not .Values.existingSecret }}
checksum/secret: {{ include (print $.Template.BasePath "/secret.yaml") . | sha256sum }}
{{- end }}
{{- end }}
