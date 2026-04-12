# DevForge MCP — Presentación Profesional

**Autor:** Ing. Gustavo Gutiérrez · Bogotá, Colombia
**Versión:** 2.1.4 · 2026

---

## El problema

Los desarrolladores interrumpen su flujo de trabajo constantemente para realizar operaciones auxiliares:

- Abrir una web externa para hashear un string, formatear SQL o convertir un JSON
- Escribir un script de Python o Bash para una tarea puntual
- Instalar una dependencia solo para usar una función una vez
- Calcular manualmente algo que debería ser automático

> Cada interrupción cuesta contexto, tiempo y concentración.

---

## ¿Qué es DevForge MCP?

**DevForge MCP** es un servidor de herramientas para agentes de IA que expone **más de 70 utilidades de desarrollo** directamente dentro de la conversación con el agente.

- El agente deja de recomendar acciones — las ejecuta
- Sin instalaciones adicionales en el entorno del desarrollador
- Sin cambio de contexto: todo ocurre dentro del flujo de trabajo actual
- Compatible con Claude Desktop, Cursor y cualquier cliente MCP

> **Del consejo a la ejecución.**

---

## ¿Cómo funciona?

DevForge implementa el **Model Context Protocol (MCP)**, un estándar abierto que permite a los modelos de lenguaje invocar funciones tipadas de forma determinista.

```
Desarrollador  →  Agente IA  →  DevForge MCP  →  Resultado
    prompt           razona         ejecuta        devuelve
```

### Propiedades clave

- **Sin estado** — cada invocación es independiente
- **Determinista** — misma entrada, misma salida siempre
- **Contratos tipados** — entradas y salidas validadas por JSON Schema
- **Sin dependencias en el cliente** — todo se ejecuta en el servidor

---

## Arquitectura

DevForge se compone de tres binarios independientes:

| Componente | Rol |
|------------|-----|
| `devforge-mcp` | Servidor MCP vía stdio — lo consume el agente IA |
| `devforge` | CLI/TUI interactiva para uso directo desde la terminal |
| `dpf` (DevPixelForge) | Motor nativo en Rust para procesamiento de imágenes, video, audio y documentos |

Los tres son **sin estado**: sin base de datos, sin archivos temporales, sin efectos secundarios.

---

## Las 70+ herramientas

Organizadas en 10 dominios funcionales:

| Dominio | Ejemplos |
|---------|----------|
| **Criptografía** | Hash, JWT, HMAC, contraseñas seguras, generación de claves |
| **Datos** | JSON/YAML/CSV, JSONPath, validación de esquemas, diff |
| **HTTP** | Peticiones, conversión de curl, URLs firmadas, webhooks |
| **Frontend** | Colores, unidades CSS, breakpoints, regex, formatos locales |
| **Imágenes** | Redimensionar, convertir, recortar, srcset, marcas de agua |
| **Código** | Formateo, métricas, plantillas |
| **Backend** | SQL, cadenas de conexión, logs, variables de entorno, MQ |
| **Tiempo** | Timestamps, cron, rangos de fechas, duraciones |
| **Texto** | UUID, slugs, Base64, codificación URL, normalización |
| **Video y Audio** | Transcodificación, recorte, miniaturas, normalización |

---

## Del consejo a la ejecución

Sin DevForge, el agente solo puede recomendar. Con DevForge, el agente actúa:

| Operación | Sin DevForge | Con DevForge |
|-----------|-------------|--------------|
| Hashear una contraseña | "Usa bcrypt con factor 12" | Ejecuta `crypto_password` y devuelve el hash |
| Convertir YAML a JSON | "Usa `js-yaml` o una web" | Ejecuta `data_yaml_convert` y devuelve el JSON |
| Redimensionar una imagen | "Usa ImageMagick: `convert -resize`" | Ejecuta `image_resize` y devuelve la ruta |
| Formatear SQL | "Pégalo en sqlformat.org" | Ejecuta `backend_sql_format` y devuelve el SQL |

---

## Composición de herramientas

Las herramientas se pueden **encadenar dentro de un mismo turno** del agente, formando pipelines completos sin intervención del usuario:

```
Ejemplo: preparar un artefacto de despliegue

  1. data_json_format    →  validar y normalizar el JSON de configuración
  2. data_yaml_convert   →  serializar como YAML para el manifiesto
  3. crypto_hash         →  calcular checksum SHA-256 del payload
  4. http_signed_url     →  generar URL firmada con expiración
  5. image_resize        →  generar variantes responsivas del asset

Resultado: pipeline completo ejecutado en un solo turno de conversación.
```

---

## Casos de uso por rol

| Rol | Herramientas de mayor valor |
|-----|-----------------------------|
| **Backend** | `backend_sql_format`, `crypto_jwt`, `backend_log_parse`, `backend_env_inspect` |
| **Frontend** | `frontend_color`, `frontend_breakpoint`, `frontend_css_unit`, `frontend_locale_format` |
| **DevOps / SRE** | `backend_env_inspect`, `http_signed_url`, `time_cron`, `backend_mq_payload` |
| **Full-Stack** | Todo lo anterior + `image_resize`, `video_transcode`, `audio_normalize` |
| **Cualquier rol** | `text_uuid`, `crypto_hash`, `text_base64`, `time_convert`, `data_yaml_convert` |

---

## Herramientas que reemplaza

DevForge elimina la necesidad de instalar estas herramientas en el entorno local:

```
openssl       →  crypto_hash, crypto_keygen, crypto_hmac
jq            →  data_jsonpath, data_json_format
yq            →  data_yaml_convert
ImageMagick   →  image_resize, image_convert, image_crop
FFmpeg        →  video_transcode, audio_normalize
uuidgen       →  text_uuid
base64        →  text_base64
date / GNU    →  time_convert, time_diff, time_date_range
```

> Un solo servidor reemplaza un ecosistema completo de herramientas dispersas.

---

## Instalación

### Homebrew (Linux y macOS)

```bash
brew install GustavoGutierrez/devforge/devforge
```

Instala los tres binarios (`devforge`, `devforge-mcp`, `dpf`) en un solo comando.

### Configuración del cliente MCP

```json
{
  "mcpServers": {
    "devforge": {
      "command": "/usr/local/bin/devforge-mcp"
    }
  }
}
```

Compatible con **Claude Desktop**, **Cursor** y cualquier cliente MCP con transporte stdio.

---

## Por qué DevForge

- **Un solo punto de acceso** a más de 70 herramientas deterministas
- **Sin interrupciones** — el desarrollador no sale del flujo de trabajo
- **Sin dependencias** en el entorno local para la mayoría de operaciones
- **Auditabilidad natural** — todas las entradas y salidas quedan en el contexto de la conversación
- **Composable** — los pipelines multi-paso se ejecutan en un único turno del agente

> DevForge convierte al agente de IA en un colaborador operativo,  
> no solo en un asistente de texto.

---

## Referencias

| Recurso | Enlace |
|---------|--------|
| Repositorio DevForge MCP | [github.com/GustavoGutierrez/devforge](https://github.com/GustavoGutierrez/devforge) |
| Motor de medios DevPixelForge | [github.com/GustavoGutierrez/devpixelforge](https://github.com/GustavoGutierrez/devpixelforge) |
| Especificación MCP | [modelcontextprotocol.io](https://modelcontextprotocol.io) |

---

*DevForge MCP — Ing. Gustavo Gutiérrez · Bogotá, Colombia · 2026*
