{{ define "componentName" }}initializer{{ end }}
{{ define "componentType" }}initializer{{ end }}
{{ define "command" }}[dodoo-initializer, "--config", "/run/configs/odoo/", "--new-database", "{{ .Instance.Spec.HostName }}" {{ if .Instance.Spec.InitModules }}, "--modules", "{{ range $index, $element := .Instance.Spec.InitModules }}{{ if $index }},{{ end }}{{ $element }}{{ end }}"{{ end }}]{{ end }}
{{ template "job" . }}