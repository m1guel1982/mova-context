# Configurar CI/CD
Proyecto: `{{PROJECT}}` · Stack: `{{STACK}}` · CI: `{{CI_PROVIDER}}` · Cloud: `{{CLOUD_PROVIDER}}`
Ockham: ver `ockham-core.md`.

Entrega: pipeline lint→test→build→scan→deploy · secrets fuera de texto plano · health check + rollback automático · dev/staging/prod · Dockerfile si aplica. Mismo artefacto de staging pasa a prod. Restricciones: `{{CONSTRAINTS}}`.
