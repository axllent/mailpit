# Changelog

Notable changes to Mailpit will be documented in this file.

{{ if .Versions -}}
{{ if .Unreleased.CommitGroups -}}
## [Unreleased]

{{ if .Unreleased.CommitGroups -}}
{{ range .Unreleased.CommitGroups -}}
### {{ .Title }}
{{ range .Commits -}}
- {{ if .Scope }}**{{ .Scope }}:** {{ end }}{{ .Subject }}
{{ end }}
{{ end }}
{{ end -}}
{{ end -}}
{{ end -}}

{{ range .Versions }} 
{{- if .CommitGroups -}}
## [{{ .Tag.Name }}]

{{ if .NoteGroups -}}
{{ range .NoteGroups -}}
### {{ .Title }}
{{ range .Notes }}
{{ .Body }}
{{ end -}}
{{ end }}
{{ end -}}
{{ end -}}

{{ range .CommitGroups -}}
### {{ .Title }}
{{ range .Commits -}}
- {{ if .Scope }}**{{ .Scope }}:** {{ end }}{{ .Subject }}
{{ end }}
{{ end }}

{{- if .MergeCommits -}}
### Pull Requests
{{ range .MergeCommits -}}
- {{ .Header }}
{{ end }}
{{ end }}
{{ end -}}
