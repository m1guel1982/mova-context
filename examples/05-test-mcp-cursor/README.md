Evidencia de Dry-run (Prueba en Cline — Anthropic Haiku)

Esta prueba verifica el control de gobernanza de Mova usando egress_audit.dry_run: true sobre el proyecto /projects/02-pii-compliance-governance.

Mova procesó los archivos fuente (customers.json, customer-profile.pdf, etc.), aplicó el enmascarado de PII y midió el tamaño del contexto (7.787 tokens). Al estar activo el modo dry-run, generó el reporte de evidencia (egress.md) en disco sin enviar ningún token al modelo externo.

Claude Anthropic Haiku (a través de Cline) funcionó únicamente como la interfaz cliente: recibió la directiva de seguridad del servidor MCP de Mova y mostró el diagnóstico en pantalla sin intentar leer archivos locales ni saltarse el bloqueo. El motor que midió, sanitizó y aplicó el control fue Mova.

"egress_audit": {
  "dry_run": true,
  "output_file": "egress_sanitized.md"
}