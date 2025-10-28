apiVersion: v1
kind: Secret
metadata:
  name: {{ include "agentchart.certsecretname" . }}
  namespace: {{ include "agentchart.namespace" . }}
data:
  device-rsa.key: |
{{- .Files.Get "device-rsa.key" | nindent 4 }}
  device-rsa.crt: |
{{- .Files.Get "device-rsa.crt" | nindent 4 }}
  device-ecdsa.key: |
{{- .Files.Get "device-ecdsa.key" | nindent 4 }}
  device-ecdsa.crt: |
{{- .Files.Get "device-ecdsa.crt" | nindent 4 }}