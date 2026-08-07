{{ range . }}
## {{ .Name }} ({{ .LicenseName }})

{{ .LicenseText }}
{{ end }}
