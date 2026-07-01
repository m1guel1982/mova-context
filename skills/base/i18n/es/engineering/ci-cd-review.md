# Objetivo
Revisar pipeline CI/CD. Provider: {{CI_PROVIDER}}
KISS+DRY: ver `kiss-dry-core.md`.

# Verificaciones
* Pipeline reproducible: mismo commit = mismo artefacto
* Tests completos antes de merge
* Secrets en secret manager, no en texto plano
* Artefacto inmutable, no reconstruir al deployar
* Health check post-deploy con rollback automático
* Ambientes progresivos dev → staging → prod

# Anti-patrones
Tests saltados en branch "urgente" · secrets visibles en logs · build distinto por ambiente · sin validación post-deploy

# Output
Gaps encontrados, pipeline refactorizado con stages correctos.
