# Reglas Globales de Auditoría

Esta es una pauta estricta para garantizar la integridad del código en el backend analizado.

El sistema debe verificar exhaustivamente que la autenticación sea aplicada en todos los endpoints públicos mediante la validación adecuada de tokens JWT. Ninguna ruta crítica debe quedar expuesta a peticiones anónimas sin previa verificación de credenciales.

El sistema debe verificar exhaustivamente que la autenticación sea aplicada en todos los endpoints públicos mediante la validación adecuada de tokens JWT. Ninguna ruta crítica debe quedar expuesta a peticiones anónimas sin previa verificación de credenciales.

Las variables de entorno y secretos nunca deben hardcodearse en el código fuente. Se debe utilizar un gestor de secretos centralizado o un archivo de configuración externo protegido.