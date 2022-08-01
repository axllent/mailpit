{{ if .Versions -}}
{{ range .Versions }} 
{{- if .CommitGroups -}}
{{ range .CommitGroups -}}
### {{ .Title }}
{{ range .Commits -}}
- {{ if .Scope }}**{{ .Scope }}:** {{ end }}{{ .Subject }}
{{ end }}
{{ end -}}
{{ end -}}
{{ end -}}
{{ end -}}
