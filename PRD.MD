# Documento de Arquitectura y Planificación: Proyecto SINGULARITY

**Singularity** es un motor de memoria de estado y orquestación para agentes de IA (integrable vía MCP a OpenCode y otros editores). A diferencia de los sistemas de memoria tradicionales que actúan como "buscadores de texto", Singularity funciona como una **Máquina de Estados y Pizarra (Blackboard)** diseñada específicamente para maximizar la autonomía de sistemas multi-agente y optimizar radicalmente el consumo de *requests* de las APIs modernas (como Minimax).

## 1. Filosofía y Principios Fundamentales

* **El Request es Rey, el Token es Barato:** El sistema asume que el costo y el cuello de botella residen en la cantidad de interacciones de red (requests), no en la longitud del contexto. Se prohíbe el "ping-pong" exploratorio. Toda la información necesaria se inyecta de golpe, y la respuesta se consolida de una sola vez.
* **Patrón Blackboard (La Pizarra Muta):** Los agentes no se comunican entre sí. Singularity actúa como una pizarra centralizada. Un agente lee la pizarra al nacer, hace su trabajo aislado, escribe el resultado final en la pizarra, y muere.
* **Jerarquía de Contextos (Aislamiento Estricto):** Para evitar la "amnesia del modelo" y las alucinaciones por exceso de ruido, el sistema distingue entre roles. El *Orquestador* solo ve resúmenes de alto nivel y mapas del proyecto. Los *Sub-agentes* solo ven el código específico de su tarea aislada. 
* **Cero Dependencias, Máximo Rendimiento:** Construido en Go puro con BadgerDB. Un solo binario hiper-optimizado, sin bases de datos externas pesadas, sin librerías de C (CGO) y con un consumo de recursos ínfimo en la máquina del usuario.

## 2. Arquitectura Conceptual

Singularity se compone de tres pilares abstractos:

1. **El Motor de Persistencia (BadgerDB - Key-Value Store):**
   * **Cerebro Activo (Contexto Denso):** Claves cortas y altamente actualizadas que representan el "Estado del Mundo" (ej. tareas pendientes, decisiones arquitectónicas vigentes, bloqueos actuales).
   * **Archivo Profundo (Contexto Gordo):** Claves pesadas que almacenan volcados completos de código, logs de errores pasados o tareas finalizadas. Solo se acceden si es estrictamente necesario.

2. **El Servidor MCP (El Puente de Comunicación):**
   * Expone una interfaz mínima a OpenCode. En lugar de ofrecer decenas de herramientas de lectura/escritura granulares, ofrece "Herramientas de Transacción Masiva" para leer y escribir estados completos de una sola vez.

3. **El Inyector de Protocolo (Memory & State Protocol):**
   * La lógica interna que formatea los datos de BadgerDB y los convierte en un *Mega-Prompt* estructurado que obliga al modelo (Minimax) a usar su Cadena de Pensamiento (*Chain of Thought*) internamente antes de emitir un solo request de salida.

## 3. Funciones Clave (Qué debe hacer)

* **Front-Loading Automático:** Al iniciar una sesión en OpenCode, Singularity debe ser capaz de ensamblar el "Cerebro Activo" y pasarlo como contexto inicial sin que el agente tenga que gastar un request preguntando "¿dónde nos quedamos?".
* **Consolidación de Estado (`commit_world_state`):** Una única y poderosa herramienta MCP. El agente le pasará un objeto JSON masivo detallando qué tarea completó, qué aprendió, qué código nuevo generó y cuál es el resumen para el Orquestador. Singularity se encarga de parsear esto y actualizar múltiples claves en BadgerDB en una sola transacción.
* **Recuperación de Archivo (`fetch_deep_context`):** Una herramienta de emergencia por si el agente realmente necesita el código de una tarea antigua que no está en el resumen activo.
* **Gestión de Tareas (DAG Manager):** Singularity debe tener la capacidad interna de entender qué tareas están "pendientes", "en progreso" o "completadas", permitiendo que el agente Orquestador planifique el futuro sin perderse.

## 4. Flujo de Trabajo y Ciclo de Vida (Autonomía)

El ciclo de vida de una sesión en Singularity sigue este patrón inmutable:

1. **Arranque (Spawn):** Singularity lee la Pizarra (BadgerDB). Empaqueta la misión actual y el estado del código en un bloque denso.
2. **Inyección:** OpenCode envía este bloque en el *Request #1* a Minimax.
3. **Razonamiento Interno:** Minimax (aprovechando el coste cero de pensar) evalúa el problema, escribe código mentalmente, se autocorrige y llega a la solución óptima.
4. **Impacto (Commit):** Minimax llama a la herramienta `commit_world_state`.
5. **Cierre de Sesión:** Singularity actualiza la Pizarra, guarda el archivo profundo, genera el resumen limpio y prepara el terreno para la siguiente interacción (o para despertar al Orquestador).

## 5. Estrategia de Implementación (Fases de Desarrollo)

* **Fase 1: Core de Almacenamiento (La Pizarra).** Configurar el proyecto en Go, implementar BadgerDB y crear la lógica para guardar/leer estructuras JSON en jerarquías de claves (Cerebro Activo vs. Archivo Profundo).
* **Fase 2: La Interfaz MCP (El Puente).** Implementar el servidor Model Context Protocol en Go. Definir los esquemas JSON (JSON Schema) exactos que aceptará la herramienta `commit_world_state`.
* **Fase 3: Integración y Protocolo (El Cerebro).** Crear los *System Prompts* (las reglas del juego) que se inyectarán en OpenCode para domar a Minimax y obligarlo a respetar la regla de "Pensar mucho, pedir poco".
* **Fase 4: Lógica Multi-Agente.** Añadir la capacidad a Singularity de gestionar colas de tareas, aislando el contexto del Orquestador (planificación) del contexto de los Sub-agentes (ejecución de código puro).
