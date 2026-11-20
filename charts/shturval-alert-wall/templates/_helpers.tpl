{{- define "alertwall.name" -}}shturval-alert-wall{{- end -}}
{{- define "alertwall.namespace" -}}{{ default .Release.Namespace .Values.namespaceOverride }}{{- end -}}