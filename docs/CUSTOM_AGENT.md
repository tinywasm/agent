# **Paradigmas Arquitectónicos para Sistemas de IA Agéntica Personalizados: Un Análisis Fundacional de Patrones de Diseño, Multi-Tenencia y Orquestación de Herramientas**

## **1\. Resumen Ejecutivo**

La transición de la Inteligencia Artificial Generativa hacia la IA Agéntica representa un cambio fundamental en la arquitectura de software contemporánea. Mientras que los Modelos de Lenguaje Grande (LLMs) tradicionales funcionan como motores pasivos de generación de texto, los sistemas de IA Agéntica introducen la capacidad de razonamiento autónomo, toma de decisiones y, crucialmente, interacción con el entorno a través de la ejecución de herramientas. Para aplicaciones empresariales que requieren soporte para múltiples clientes, aislamiento estricto de datos y una fiabilidad determinista, la dependencia de marcos de trabajo comerciales o de código abierto (frameworks "off-the-shelf") a menudo introduce capas de abstracción innecesarias que oscurecen el flujo de control y complican la depuración. En consecuencia, una implementación personalizada —construida sobre patrones arquitectónicos fundamentales en lugar de librerías de alto nivel— ofrece la precisión quirúrgica requerida para entornos de producción sensibles.

Este informe proporciona un plano arquitectónico exhaustivo para el diseño de un sistema de agente de IA personalizado y multi-inquilino. Deconstruye las arquitecturas cognitivas centrales (como ReAct y Plan-and-Execute), analiza la mecánica teórica del uso de herramientas (protocolos de Function Calling) y detalla la infraestructura de datos necesaria para una gestión de sesiones robusta y un aislamiento de memoria garantizado. Al sintetizar conocimientos de la investigación actual y despliegues industriales, este documento sirve como una guía definitiva para arquitectos de software que buscan diseñar sistemas de agentes resilientes, escalables y autónomos desde los primeros principios, sin la dependencia de librerías externas que limiten la personalización.

El análisis revela que el éxito en la implementación de una solución personalizada ("bare metal") no reside en la elección del modelo subyacente, sino en la ingeniería del sistema de orquestación que lo envuelve: cómo se gestiona el estado, cómo se estructuran los bucles de retroalimentación y cómo se segregan los contextos de memoria entre distintos usuarios para evitar la contaminación de datos cruzada.

## **2\. La Fisiología del Razonamiento Agéntico: Patrones Arquitectónicos Nucleares**

El comportamiento de un agente de IA no está definido únicamente por el modelo que utiliza, sino por la "arquitectura cognitiva" que orquesta su proceso de razonamiento. Cuando se construye un agente personalizado sin librerías externas, el arquitecto debe definir explícitamente el bucle de control que gobierna cómo el LLM percibe las tareas, planifica las acciones y reflexiona sobre los resultados. Este capítulo examina los patrones predominantes que los marcos actuales utilizan, permitiendo su replicación en una solución personalizada.

### **2.1 El Patrón ReAct (Razonamiento \+ Actuación)**

El patrón ReAct, introducido para resolver los problemas de alucinaciones y propagación de errores en el razonamiento de cadena de pensamiento (Chain-of-Thought), se erige como el estándar *de facto* para agentes de propósito general. Este enfoque combina trazas de razonamiento con la ejecución de acciones, permitiendo al modelo "pensar" sobre lo que necesita hacer, "actuar" llamando a una herramienta, y luego "observar" la salida de esa herramienta para ajustar su siguiente paso.1

#### **2.1.1 Flujo Lógico Arquitectónico**

En una implementación personalizada, el bucle ReAct no es una función mágica importada, sino un bucle while estructurado que gestiona el estado de la conversación. La lógica fluye de la siguiente manera:

1. **Procesamiento de Entrada:** El sistema recibe una consulta de un usuario específico.  
2. **Generación de Pensamiento (Thought):** Se solicita al LLM que genere un "Pensamiento" sobre el estado actual del problema. Este es un paso de razonamiento donde el modelo analiza la brecha entre la solicitud del usuario y la información que posee actualmente.3  
3. **Determinación de Acción (Action):** Basado en el pensamiento, el modelo decide si se requiere una acción externa. Si es así, emite un comando de "Acción" estructurado (a menudo un nombre de herramienta específico y parámetros en formato JSON).5  
4. **Ejecución de Herramienta (The Act):** El sistema de control (el código personalizado) intercepta esta acción, detiene la generación del LLM y ejecuta el código específico (por ejemplo, una consulta a base de datos o una llamada a API).  
5. **Inyección de Observación (Observation):** La salida de la herramienta es capturada y anexada al historial de la conversación como una "Observación".  
6. **Bucle o Terminación:** El historial actualizado (Entrada del Usuario \+ Pensamiento \+ Acción \+ Observación) se retroalimenta al LLM. El modelo entonces genera el siguiente Pensamiento. Este ciclo se repite hasta que el modelo determina que tiene suficiente información para generar una "Respuesta Final".6

#### **2.1.2 Consideraciones de Implementación Personalizada**

Para una construcción personalizada, el desafío de ingeniería crítico es la gestión de la **Secuencia de Parada** (Stop Sequence). Se debe instruir al LLM (vía el prompt del sistema) para que deje de generar texto inmediatamente después de emitir una Acción. Sin esto, el modelo podría alucinar la Observación misma, inventando un resultado ficticio para la herramienta que acaba de solicitar. El orquestador debe detectar esta secuencia de parada, analizar la acción, ejecutar la herramienta y luego, explícitamente, incitar al modelo con el encabezado de Observación para que continúe el proceso.7

**Limitaciones:** Aunque ReAct proporciona alta interpretabilidad e interactividad, sufre de ineficiencia en tareas complejas. Ejecuta de manera serial, lo que significa que los pasos que podrían paralelizarse se manejan uno por uno, aumentando la latencia y los costos de tokens. Además, puede quedar atrapado en "bucles de razonamiento" donde repite la misma acción ineficaz si la observación no es útil.8

### **2.2 Arquitectura Planificar y Ejecutar (Plan-and-Execute)**

Para consultas complejas y de múltiples pasos que se encuentran a menudo en el soporte al cliente empresarial (por ejemplo, "Verifique mis últimas tres facturas, calcule el IVA total y envíeme un resumen"), el patrón ReAct puede volverse inmanejable y propenso a perder el contexto. El patrón Planificar y Ejecutar separa el razonamiento de alto nivel de la ejecución de bajo nivel, una distinción crítica para sistemas robustos.10

#### **2.2.1 La División Planificador-Ejecutor**

Esta arquitectura utiliza dos indicaciones distintas o incluso dos modelos diferentes:

1. **El Planificador:** Analiza la solicitud del usuario y genera un Gráfico Acíclico Dirigido (DAG) completo o una lista secuencial de pasos necesarios para resolver el problema. No ejecuta herramientas; solo estructura el flujo de trabajo.  
2. **El Ejecutor:** Toma el plan y ejecuta los pasos. En una variante de "Replanteamiento" (Replanning), el planificador revisa el estado después de cada pocos pasos para ajustar el plan si la situación cambia o si surgen nuevos datos.9

#### **2.2.2 Ventajas para la Implementación Personalizada**

Separar la planificación de la ejecución mejora significativamente la fiabilidad para consultas complejas. Evita que el agente "pierda el hilo" a mitad de una larga cadena de llamadas a herramientas. En una arquitectura personalizada, el "Plan" puede analizarse en un objeto estructurado (por ejemplo, una lista JSON de tareas), permitiendo que el sistema ejecute tareas independientes en paralelo —algo que un bucle ReAct estándar no puede hacer fácilmente— reduciendo así el tiempo total de respuesta.10

### **2.3 Patrones de Reflexión y Autocorrección**

Un componente vital en arquitecturas modernas, a menudo omitido en implementaciones simples pero esencial para la calidad empresarial, es la **Reflexión**.

* **Mecanismo:** Antes de entregar una respuesta final al usuario, el sistema invoca un paso de "Crítica". Se le pide al LLM (o a un modelo diferente actuando como supervisor) que evalúe su propia respuesta propuesta o el resultado de una herramienta. "¿Esta respuesta satisface completamente la solicitud del usuario?", "¿Hay errores lógicos?".  
* **Valor Arquitectónico:** Este patrón reduce las alucinaciones y mejora la precisión en tareas que requieren exactitud, como cálculos financieros o recuperación de datos específicos del cliente. En una implementación personalizada, esto se manifiesta como un paso adicional en la máquina de estados antes de la transición al estado de "Respuesta Final".2

### **2.4 Matriz de Decisión para su Caso de Uso**

Dado que su objetivo es implementar un agente para "recibir a varios clientes con diferentes preguntas":

| Característica de la Consulta | Patrón Recomendado | Justificación |
| :---- | :---- | :---- |
| **Dinámica / Exploratoria** (ej. "Mi internet no funciona, ¿qué hago?") | **ReAct** | Requiere un diagnóstico paso a paso donde el siguiente paso depende estrictamente del resultado anterior. |
| **Procedural / Estándar** (ej. "Dame mi saldo y envíame la factura") | **Plan-and-Execute** | El flujo es predecible y optimizable. Se puede planificar de antemano para eficiencia. |
| **Crítica / Alta Precisión** (ej. "Cambia mi plan de facturación") | **Reflexion \+ ReAct** | Requiere verificación doble antes de ejecutar acciones de escritura (efectos secundarios). |

Un enfoque híbrido, utilizando un **Enrutador** (discutido en la Sección 6\) para clasificar la intención y despachar la consulta a un agente ReAct para solución de problemas o a un motor Plan-and-Execute para tareas procedimentales, suele ser la arquitectura más robusta.13

## **3\. La Fisiología del Uso de Herramientas: Protocolos y Mecanismos**

La característica definitoria de un agente es su capacidad para usar herramientas. En una implementación personalizada sin librerías como LangChain o Semantic Kernel, entender el protocolo "crudo" de **Function Calling** (Uso de Herramientas) es primordial. Esto no es magia; es un protocolo de procesamiento de texto estructurado entre el sistema orquestador y el LLM.15

### **3.1 El Ciclo de Vida de la Llamada a Función**

El mecanismo de uso de herramientas opera en un ciclo de vida específico que el orquestador personalizado debe gestionar explícitamente. Este ciclo consta de cuatro fases críticas:

1. **Definición de Herramienta (Inyección de Esquema):** Las herramientas deben definirse en un formato que el LLM comprenda, típicamente un esquema JSON. Este esquema describe el nombre de la función, una descripción de lo que hace y los parámetros que acepta (tipos, obligatorios vs. opcionales). Estas definiciones se inyectan en el prompt del sistema o en el parámetro tools de la API del modelo.17  
   * *Insight Estratégico:* La calidad del campo "descripción" en el esquema es tan crítica como el código mismo. El LLM utiliza esta descripción semántica para decidir *si* y *cuándo* llamar a la herramienta. Descripciones ambiguas conducen a alucinaciones de herramientas o mal uso.  
2. **Detección de Intención y Generación:** Cuando el LLM procesa un prompt, evalúa si el texto implica una necesidad de usar una herramienta. Si es así, en lugar de generar una respuesta de texto, genera un objeto estructurado de "Llamada a Herramienta" (Tool Call), generalmente en JSON, que contiene el nombre de la función y los argumentos a pasar.19  
   * *Detalle de Implementación Personalizada:* En una implementación cruda, se debe verificar la razón de finalización (finish\_reason) de la respuesta. Si indica tool\_calls, el sistema debe pausar la renderización de texto al usuario y entrar en la fase de ejecución.19  
3. **Ejecución y Serialización:** El sistema analiza los argumentos JSON. Es crítico validar estos argumentos contra el esquema esperado antes de la ejecución para prevenir errores en tiempo de ejecución (por ejemplo, asegurar que una cadena de fecha sea realmente una fecha). La función se ejecuta (ejecutando su código Python/SQL interno), y el valor de retorno (escalar, JSON o texto) debe serializarse de nuevo a un formato de cadena.20  
4. **Inyección de Contexto (Cierre del Bucle):** La salida de la herramienta no se muestra al usuario inmediatamente. En su lugar, se anexa al historial de mensajes con un rol específico (por ejemplo, tool o function). El LLM se invoca nuevamente con este nuevo historial. "Ve" que solicitó una acción y que la acción produjo un resultado. Luego usa este resultado para generar la respuesta final en lenguaje natural para el usuario.21

### **3.2 Gestión de Herramientas Agnóstica al Protocolo**

Un desafío mayor en la arquitectura personalizada es evitar el bloqueo de proveedores (vendor lock-in). Los diferentes modelos (OpenAI, Anthropic, Llama, Mistral) tienen formatos para llamar a herramientas que varían ligeramente.

* **OpenAI** utiliza un array tools específico y un campo de respuesta tool\_calls.  
* **Anthropic** utiliza etiquetas basadas en XML o estructuras de herramientas específicas en modelos más nuevos.  
* **Llama/Mistral** a menudo dependen de estrategias de prompting donde las herramientas se definen en el prompt del sistema y el modelo genera una secuencia de tokens específica para invocarlas.

*Recomendación Arquitectónica:* Implemente una capa de abstracción de **Registro de Herramientas**. Este componente debe aceptar una función estándar (en su lenguaje de programación, ej. Python), generar automáticamente el esquema JSON (usando introspección o librerías de validación de tipos), y luego permitir "Adaptadores" que formateen este esquema para el LLM específico que se esté utilizando. Esto asegura que la lógica central de su agente permanezca agnóstica al proveedor del modelo subyacente, permitiéndole cambiar de GPT-4 a Claude 3 o a un modelo local Llama 3 sin reescribir sus herramientas.17

### **3.3 El Protocolo de Contexto de Modelo (MCP)**

Un estándar emergente y relevante para arquitecturas de 2025 es el **Model Context Protocol (MCP)**. MCP tiene como objetivo estandarizar cómo los agentes se conectan a fuentes de datos, reemplazando integraciones frágiles y ad-hoc. En una construcción personalizada, adoptar una estructura similar a MCP —donde las herramientas se tratan como "servidores" distintos que exponen capacidades al "cliente" (el agente)— puede mejorar significativamente la modularidad. Esto permite escribir una herramienta una vez y potencialmente reutilizarla a través de diferentes agentes o incluso compartirla entre diferentes sistemas de agentes, facilitando la escalabilidad y el mantenimiento.16

## **4\. Arquitectura para Multi-Tenencia y Aislamiento de Sesiones**

Servir a "varios clientes" (multi-tenencia) introduce requisitos estrictos de aislamiento de datos. En un sistema de agentes personalizado, la "memoria" no es un monolito; es una infraestructura segmentada y segura. El fallo en aislar adecuadamente las sesiones puede llevar a fugas de datos, donde el contexto de un cliente (ej. datos de precios, historial personal) se filtra en la interacción de otro.25

### **4.1 La Jerarquía de la Memoria Agéntica**

Para gestionar el estado de múltiples clientes de manera efectiva, la memoria debe diseñarse en capas jerárquicas:

1. **Memoria a Corto Plazo (Memoria de Trabajo):**  
   * *Alcance:* Efímera, existe solo durante la duración de un hilo de conversación único.  
   * *Almacenamiento:* Almacenes clave-valor de alta velocidad (ej. Redis) o estructuras en memoria si se usan conexiones con estado (WebSockets).  
   * *Mecanismo:* Almacena la ventana deslizante del diálogo inmediato (Consulta del Usuario, Pensamientos del Agente, Salidas de Herramientas).  
   * *Aislamiento:* Estrictamente indexada por session\_id. Cuando una sesión termina o expira, estos datos se archivan o eliminan.27  
2. **Memoria a Largo Plazo (Memoria Episódica):**  
   * *Alcance:* Persistente, recuerda interacciones pasadas con el mismo usuario a lo largo de días o meses.  
   * *Almacenamiento:* Bases de Datos Vectoriales (ej. pgvector, Milvus, Chroma) para recuperación semántica, más bases de datos SQL para registros estructurados.  
   * *Mecanismo:* Antes de responder a una nueva consulta, el agente consulta esta memoria en busca de contexto relevante ("¿Ha preguntado este usuario sobre este error antes?").  
   * *Aislamiento:* Esta es la frontera de seguridad más crítica. Cada incrustación vectorial (embedding) y entrada de registro debe estar etiquetada con un tenant\_id y un user\_id. Las consultas a la base de datos vectorial *deben* incluir un pre-filtro que imponga estos IDs. **Nunca** realice una búsqueda de similitud global para luego filtrar los resultados; esto es ineficiente y un riesgo de privacidad.29  
3. **Memoria Semántica (Conocimiento):**  
   * *Alcance:* Solo lectura o acceso restringido, contiene conocimiento del dominio (documentos, FAQs).  
   * *Almacenamiento:* Base de Datos Vectoriales (RAG).  
   * *Aislamiento:* Puede ser compartida (Base de Conocimiento Global) o Específica del Inquilino (Documentación Privada). La arquitectura debe soportar "Namespaces" o "Colecciones" para segregar física o lógicamente los datos de cada inquilino.30

### **4.2 Diseño de Esquema de Base de Datos para Agentes Personalizados**

Una implementación robusta requiere un esquema relacional para rastrear el estado complejo de los agentes. Una tabla simple de "mensajes" es insuficiente para flujos de trabajo agénticos que incluyen pensamientos internos y llamadas a herramientas.

**Estructura de Esquema Propuesta:**

| Tabla | Propósito | Columnas Críticas para Aislamiento y Control |
| :---- | :---- | :---- |
| tenants | Raíz de multi-tenencia | id, name, config (límites, herramientas permitidas) |
| users | Usuarios finales dentro de inquilinos | id, tenant\_id, permissions |
| sessions | Contenedores de conversación | id, user\_id, status (activa/cerrada), created\_at |
| threads | Historial lineal de mensajes | id, session\_id, parent\_thread\_id (para ramificaciones/branching) |
| messages | Unidades atómicas de interacción | id, thread\_id, role (user/assistant/system/tool), content, token\_count |
| tool\_calls | Registro de auditoría de acciones | id, message\_id, tool\_name, input\_params, output\_result, execution\_time |
| memories | Almacén vectorial a largo plazo | id, user\_id, tenant\_id, embedding (vector), content\_chunk |

*Insight:* Separar tool\_calls de los messages genéricos permite una auditoría granular. Puede consultar "¿Cuántas veces usó el Inquilino A la herramienta de Búsqueda?" o "¿Cuál fue la latencia promedio de la herramienta API?". Esta observabilidad es crucial para sistemas en producción.31

### **4.3 Ingeniería de Contexto y Gestión de Ventana**

En una construcción personalizada, debe gestionar manualmente la ventana de contexto. No puede simplemente anexar cada nuevo mensaje para siempre.

* **FIFO (First-In-First-Out):** El método más simple, eliminando los mensajes más antiguos.  
* **Resumen (Summarization):** Comprimir periódicamente los turnos más antiguos en un mensaje de sistema narrativo ("El usuario preguntó sobre X y el agente respondió Y...").  
* **Presupuesto de Tokens:** Un módulo personalizado de "Gestor de Contexto" debe calcular el conteo de tokens del prompt del sistema \+ definiciones de herramientas \+ contexto RAG recuperado \+ historial de chat. Luego poda el historial de chat para que se ajuste al límite del modelo (dejando un búfer para la respuesta).  
* *Nota de Seguridad:* Asegúrese de que la inyección del tenant\_id o datos sensibles en el prompt del sistema esté codificada en la capa de confianza ("hardcoded" en el backend), no accesible a través de la entrada del usuario, para prevenir el "salto de inquilino" mediante inyección de prompt.32

## **5\. Orquestación: El Sistema Nervioso del Agente**

Construir un agente personalizado implica construir un **Orquestador**. Esta es la lógica de control que gobierna las transiciones de estado del agente. Reemplaza al "AgentExecutor" que se encuentra en librerías como LangChain.

### **5.1 Orquestador vs. Enrutador (Router)**

Es vital distinguir entre Enrutamiento y Orquestación, ya que los sistemas personalizados a menudo necesitan ambos para escalar.

* **Enrutamiento:** El punto de entrada. Un paso de clasificación ligero (usando un LLM más pequeño o lógica de palabras clave) que determina la intención del usuario. "¿Quiere este usuario restablecer una contraseña (Procedimental) o hacer una pregunta de análisis compleja (Razonamiento)?". El enrutador envía la solicitud al agente o flujo de trabajo especializado apropiado.13  
* **Orquestación:** El administrador del flujo de trabajo activo. Una vez que una solicitud se enruta a un agente, el orquestador gestiona el bucle ReAct, maneja los errores de ejecución de herramientas, gestiona la recuperación de memoria y asegura que el agente se mantenga en la tarea.35

### **5.2 Máquinas de Estados Finitos (FSM) para Control Determinista**

Para la fiabilidad empresarial, los agentes puramente probabilísticos (donde el LLM decide todo en cada paso) son riesgosos. Se recomienda una arquitectura de **Máquina de Estados Finitos (FSM)** para construcciones personalizadas.

* *Concepto:* Defina el comportamiento del agente como un gráfico de estados (ej. Inactivo \-\> RecopilandoInfo \-\> EjecutandoAccion \-\> Verificando \-\> Respondiendo).  
* *Implementación:* El LLM actúa como el motor de transición. Decide *a qué* estado moverse a continuación, pero las *acciones permitidas* en cada estado están estrictamente definidas por código.  
* *Beneficio:* Si un agente está en el estado RecopilandoInfo, la FSM puede restringirlo para usar solo herramientas de lectura, evitando que ejecute accidentalmente una acción de "Eliminar" hasta que transicione a un estado de Accion confirmado. Este enfoque híbrido (Restricciones de Código \+ Razonamiento LLM) proporciona la fiabilidad de "Suelo Alto, Techo Alto" mencionada en las comparaciones de marcos.37

### **5.3 Manejo de Paralelismo y Concurrencia**

A diferencia de un script simple, un agente que maneja "varios clientes" debe ser asíncrono. El orquestador personalizado debe construirse sobre un modelo impulsado por eventos (ej. usando asyncio de Python o una cola de mensajes como RabbitMQ/Kafka para escalas mayores).

* *Patrón:* La solicitud del usuario coloca un evento en una cola. El Trabajador del Agente (Agent Worker) lo recoge, carga el estado de la sesión (Memoria), realiza un "paso" de razonamiento (llamada al LLM), y luego suspende (esperando la salida de la herramienta) o completa.  
* *Serialización de Estado:* Entre pasos, el estado completo del agente (historial \+ pensamiento actual) debe serializarse a la base de datos. Esto permite que el sistema sea "sin estado" (stateless) en la capa de cómputo: cualquier nodo trabajador puede retomar el siguiente paso de la conversación, lo cual es esencial para escalar a muchos clientes simultáneos.39

## **6\. Hoja de Ruta de Implementación: Construyendo Sin Librerías**

Para implementar este agente personalizado, siga un enfoque de desarrollo en capas. Esto evita la complejidad de "caja negra" de las librerías mientras mantiene una arquitectura robusta.

### **Fase 1: El Motor de Ejecución Central**

Construya la clase AgentNode. Esta clase debe:

1. Aceptar una lista de objetos Tool (funciones Python).  
2. Generar el esquema JSON para estas herramientas dinámicamente.  
3. Gestionar la construcción del prompt (Prompt del Sistema \+ Historial).  
4. Manejar la llamada a la API del LLM.  
5. Implementar la lógica de análisis (parsing) para detectar tool\_calls.  
6. Implementar el despachador que mapea la cadena del nombre de la herramienta a la función Python real y la ejecuta.  
7. *Meta:* Un script de un solo hilo que puede tomar una consulta de usuario, llamar a una herramienta (ej. get\_weather), y responder.

### **Fase 2: El Gestor de Estado**

Introduzca la capa de base de datos.

1. Implemente el esquema relacional (Inquilinos, Sesiones, Mensajes).  
2. Cree una clase SessionManager que recupere el historial de la BD y lo formatee para el AgentNode.  
3. Implemente la lógica de "Ventana" para truncar mensajes antiguos.  
4. *Meta:* Un agente que recuerda el nombre del usuario a través de múltiples turnos de conversación.

### **Fase 3: El Enrutador y Despacho Multi-Agente**

Si diferentes clientes hacen diferentes tipos de preguntas (ej. Soporte Técnico vs. Ventas), crear un agente monolítico es ineficiente.

1. Construya un componente Router. Este puede ser un prompt simple ("Clasifica esta entrada en Categoría A o B") o un clasificador semántico.  
2. Defina Nodos de Agente especializados (ej. SalesAgent con herramientas CRM, TechAgent con herramientas de Logs).  
3. El Router envía la sesión al Nodo de Agente correcto.  
4. *Meta:* Un sistema donde "Quiero comprar" activa el flujo de Ventas y "Está roto" activa el flujo Técnico.

### **Fase 4: Fiabilidad y Barandillas (Guardrails)**

Los agentes empresariales necesitan redes de seguridad.

1. **Manejo de Errores de Herramientas:** Envuelva la ejecución de herramientas en bloques try/catch. Si una herramienta falla (ej. API caída), inyecte un mensaje de sistema en el historial: "La herramienta falló con el error X. Por favor reintente o pida aclaración al usuario." No permita que el agente se bloquee.41  
2. **Verificaciones de Alucinación:** Implemente un paso de "Reflexión". Antes de enviar la Respuesta Final, realice una llamada rápida y barata al LLM: "Verifica si esta respuesta está respaldada por las salidas de las herramientas. Responde Sí/No.".2  
3. **Límites de Tasa:** Implemente límites por tenant\_id para evitar que un solo cliente agote su cuota de API del LLM.

## **7\. Análisis Comparativo de Marcos Actuales (Aprendizaje del Mercado)**

Aunque el requisito es construir sin librerías externas, analizar los marcos existentes revela los patrones arquitectónicos que vale la pena copiar ("Conceptos a Robar") y las trampas a evitar.

* **LangChain / LangGraph:**  
  * *Concepto a Robar:* **Orquestación basada en Grafos.** Modelar el agente como un grafo (Nodos \= Razonamiento/Acciones, Bordes \= Lógica) es superior a las cadenas lineales. Permite bucles, condicionales y gestión de estado compleja.37  
  * *Qué Evitar:* La abstracción de "Chain" en sí misma. Ocultar la construcción del prompt y las llamadas a la API detrás de envoltorios gruesos hace que la depuración sea casi imposible. Mantenga su construcción de prompts visible y explícita.  
* **Microsoft AutoGen:**  
  * *Concepto a Robar:* **Patrones de Agente Conversacional.** AutoGen trata la coordinación multi-agente como una conversación. Un agente "User Proxy" puede simular el comportamiento del usuario para probar el agente principal. Este es un patrón poderoso para pruebas automatizadas de su sistema personalizado.43  
  * *Qué Evitar:* La complejidad de sus gestores de "Chat Grupal" específicos, a menos que realmente tenga más de 5 agentes debatiendo entre sí. Para la mayoría de los escenarios de soporte, un Enrutador determinista es mejor que un chat de forma libre.  
* **CrewAI:**  
  * *Concepto a Robar:* **Diseño Basado en Roles.** Definir agentes con "Roles", "Objetivos" e "Historias de Fondo" (Backstories) específicos en el prompt del sistema mejora significativamente el rendimiento. Incluso en un script personalizado, estructurar su prompt del sistema de esta manera ayuda al LLM a mantenerse en personaje.37  
* **Semantic Kernel:**  
  * *Concepto a Robar:* **El Concepto de Kernel.** Un registro central para todas las "Habilidades" (Herramientas) y "Memorias". Enfatiza que las herramientas deben ser tipadas y seguras. Adoptar un objeto "Kernel" estricto en su código que gestione todos los recursos para una solicitud es una buena arquitectura limpia.44

## **8\. Conclusión y Recomendaciones de Diseño**

Implementar un sistema de IA Agéntica personalizado para múltiples clientes es un ejercicio de **Ingeniería de Sistemas**, no solo de modelado de IA. La inteligencia del sistema emerge no solo del LLM, sino de la orquestación rigurosa del estado, la ejecución determinista de herramientas y el aislamiento seguro de los datos de los inquilinos.

Al adoptar el **patrón ReAct** para razonamiento dinámico o **Plan-and-Execute** para flujos de trabajo estructurados, y al construir un orquestador robusto basado en **Máquinas de Estados Finitos (FSM)**, puede lograr un nivel de fiabilidad y control que los marcos de caja negra a menudo sacrifican por facilidad de uso. La clave del éxito reside en la gestión explícita del "Bucle Agéntico": un ciclo transparente y observable de Percepción, Razonamiento, Acción y Observación, fundamentado en una arquitectura de datos segura y multi-inquilino. Este enfoque asegura que a medida que su base de clientes crezca, el sistema permanezca escalable, auditable y alineado con la lógica de negocio precisa.

### **Perspectivas de Orden Superior e Implicaciones**

* **La Mercantilización de la Inteligencia vs. El Valor del Contexto:** La capacidad de razonamiento cruda (el LLM) se está convirtiendo en una mercancía ("commodity"). El verdadero valor en una arquitectura de agente personalizada reside en la **Ingeniería de Contexto**: cuán efectivamente el sistema puede recuperar, filtrar y presentar los datos aislados *correctos* al modelo en el momento *correcto*. La arquitectura descrita anteriormente prioriza esto a través de su memoria jerárquica y límites de sesión estrictos.  
* **Conflicto Determinista vs. Probabilístico:** Una tensión recurrente en el diseño de agentes es la imprevisibilidad de los LLMs. El cambio hacia arquitecturas **FSM (Máquina de Estados Finitos)** representa una "corrección" en la industria, reconociendo que mientras el razonamiento es probabilístico, los flujos de trabajo empresariales deben ser deterministas. Su construcción personalizada le permite imponer esta rigidez híbrida mejor que los marcos flexibles.  
* **El Ascenso del Diseño Primero-Protocolo:** La mención de MCP (Model Context Protocol) destaca un futuro donde los agentes se definen por sus interfaces (protocolos) en lugar de sus bases de código. Al construir su agente personalizado con definiciones de herramientas estrictas basadas en esquemas, lo prepara para el futuro, haciéndolo listo para integrarse con estándares emergentes que eventualmente reemplazarán las integraciones de API ad-hoc.

Este informe delinea el "Camino Difícil" de construir agentes, que paradójicamente es el "Camino Seguro" para la producción empresarial: control total, visibilidad total y cero magia.

#### **Fuentes citadas**

1. Demystifying Agents: ReAct-Style Agents vs “Agentic Workflows” | by Dan Giannone, acceso: febrero 14, 2026, [https://medium.com/@DanGiannone/demystifying-ai-agents-react-style-agents-vs-agentic-workflows-cedca7e26471](https://medium.com/@DanGiannone/demystifying-ai-agents-react-style-agents-vs-agentic-workflows-cedca7e26471)  
2. 7 Must-Know Agentic AI Design Patterns \- MachineLearningMastery.com, acceso: febrero 14, 2026, [https://machinelearningmastery.com/7-must-know-agentic-ai-design-patterns/](https://machinelearningmastery.com/7-must-know-agentic-ai-design-patterns/)  
3. Choose a design pattern for your agentic AI system | Cloud Architecture Center, acceso: febrero 14, 2026, [https://docs.cloud.google.com/architecture/choose-design-pattern-agentic-ai-system](https://docs.cloud.google.com/architecture/choose-design-pattern-agentic-ai-system)  
4. Building AI Agents from Scratch: A Guided Walkthrough | by Arun Narayanan | Medium, acceso: febrero 14, 2026, [https://medium.com/@arunnaray/building-ai-agents-from-scratch-a-guided-walkthrough-77a2e510bbbb](https://medium.com/@arunnaray/building-ai-agents-from-scratch-a-guided-walkthrough-77a2e510bbbb)  
5. ReAct \- Prompt Engineering Guide, acceso: febrero 14, 2026, [https://www.promptingguide.ai/techniques/react](https://www.promptingguide.ai/techniques/react)  
6. Understanding AI Agents through the Thought-Action-Observation Cycle \- Hugging Face, acceso: febrero 14, 2026, [https://huggingface.co/learn/agents-course/en/unit1/agent-steps-and-structure](https://huggingface.co/learn/agents-course/en/unit1/agent-steps-and-structure)  
7. Stop-Sequences \- Real World Use Cases : r/LocalLLaMA \- Reddit, acceso: febrero 14, 2026, [https://www.reddit.com/r/LocalLLaMA/comments/1lzfwdj/stopsequences\_real\_world\_use\_cases/](https://www.reddit.com/r/LocalLLaMA/comments/1lzfwdj/stopsequences_real_world_use_cases/)  
8. Agentic AI Design Patterns: Choosing the Right Multimodal & Multi-Agent Architecture (2022–2025) | by Balaram Panda | Medium, acceso: febrero 14, 2026, [https://medium.com/@balarampanda.ai/agentic-ai-design-patterns-choosing-the-right-multimodal-multi-agent-architecture-2022-2025-046a37eb6dbe](https://medium.com/@balarampanda.ai/agentic-ai-design-patterns-choosing-the-right-multimodal-multi-agent-architecture-2022-2025-046a37eb6dbe)  
9. ReWOO vs. ReAct: Choosing the right agent architecture for the job \- Nutrient, acceso: febrero 14, 2026, [https://www.nutrient.io/blog/rewoo-vs-react-choosing-right-agent-architecture/](https://www.nutrient.io/blog/rewoo-vs-react-choosing-right-agent-architecture/)  
10. ReAct vs Plan-and-Execute: A Practical Comparison of LLM Agent Patterns, acceso: febrero 14, 2026, [https://dev.to/jamesli/react-vs-plan-and-execute-a-practical-comparison-of-llm-agent-patterns-4gh9](https://dev.to/jamesli/react-vs-plan-and-execute-a-practical-comparison-of-llm-agent-patterns-4gh9)  
11. How to Build a Plan-and-Execute AI Agent \- Ema, acceso: febrero 14, 2026, [https://www.ema.co/additional-blogs/addition-blogs/build-plan-execute-agents](https://www.ema.co/additional-blogs/addition-blogs/build-plan-execute-agents)  
12. Chapter 3: Architectures for Building Agentic AI \- arXiv, acceso: febrero 14, 2026, [https://arxiv.org/html/2512.09458v1](https://arxiv.org/html/2512.09458v1)  
13. The Orchestrator Pattern: Routing Conversations to Specialized AI ..., acceso: febrero 14, 2026, [https://medium.com/@akki7272/the-orchestrator-pattern-routing-conversations-to-specialized-ai-agents-985fcdf0d8ad](https://medium.com/@akki7272/the-orchestrator-pattern-routing-conversations-to-specialized-ai-agents-985fcdf0d8ad)  
14. The Router Pattern: A Smarter Way to Build AI Agents | by Brian Jenney | Medium, acceso: febrero 14, 2026, [https://brianjenney.medium.com/the-router-pattern-a-smarter-way-to-build-ai-agents-dbdd2ee12656](https://brianjenney.medium.com/the-router-pattern-a-smarter-way-to-build-ai-agents-dbdd2ee12656)  
15. Function calling | OpenAI API, acceso: febrero 14, 2026, [https://developers.openai.com/api/docs/guides/function-calling/](https://developers.openai.com/api/docs/guides/function-calling/)  
16. Function calling using LLMs \- Martin Fowler, acceso: febrero 14, 2026, [https://martinfowler.com/articles/function-call-LLM.html](https://martinfowler.com/articles/function-call-LLM.html)  
17. Unified Tool Integration for LLMs: A Protocol-Agnostic Approach to Function Calling \- arXiv, acceso: febrero 14, 2026, [https://arxiv.org/html/2508.02979v1](https://arxiv.org/html/2508.02979v1)  
18. How Tools Are Called in AI Agents: Complete 2025 Guide (With Examples), acceso: febrero 14, 2026, [https://medium.com/@sayalisureshkumbhar/how-tools-are-called-in-ai-agents-complete-2025-guide-with-examples-42dcdfe6ba38](https://medium.com/@sayalisureshkumbhar/how-tools-are-called-in-ai-agents-complete-2025-guide-with-examples-42dcdfe6ba38)  
19. LLM Function Calling Explained: A Deep Dive into the Request and Response Payloads | by James Tang | Medium, acceso: febrero 14, 2026, [https://medium.com/@jamestang/llm-function-calling-explained-a-deep-dive-into-the-request-and-response-payloads-894800fcad75](https://medium.com/@jamestang/llm-function-calling-explained-a-deep-dive-into-the-request-and-response-payloads-894800fcad75)  
20. LLM Tool Calls Demystified How to Process and Execute Function Requests Eng, acceso: febrero 14, 2026, [https://www.youtube.com/watch?v=2YhizOvz6lA](https://www.youtube.com/watch?v=2YhizOvz6lA)  
21. Function Calling in AI Agents \- Prompt Engineering Guide, acceso: febrero 14, 2026, [https://www.promptingguide.ai/agents/function-calling](https://www.promptingguide.ai/agents/function-calling)  
22. Building a Simple AI Agent with Function Calling: A Learning-in-Public Project \- Medium, acceso: febrero 14, 2026, [https://medium.com/@garland3/building-a-simple-ai-agent-with-function-calling-a-learning-in-public-project-acf4cd8f18bd](https://medium.com/@garland3/building-a-simple-ai-agent-with-function-calling-a-learning-in-public-project-acf4cd8f18bd)  
23. Tool Calling for LLMs: A Detailed Tutorial | by Yasir Siddique | Medium, acceso: febrero 14, 2026, [https://medium.com/@yasir\_siddique/tool-calling-for-llms-a-detailed-tutorial-a2b4d78633e2](https://medium.com/@yasir_siddique/tool-calling-for-llms-a-detailed-tutorial-a2b4d78633e2)  
24. Agentic AI Architectures with Patterns, Frameworks & MCP, acceso: febrero 14, 2026, [https://mehmetozkaya.medium.com/agentic-ai-architectures-with-patterns-frameworks-mcp-25afcc97ae62](https://mehmetozkaya.medium.com/agentic-ai-architectures-with-patterns-frameworks-mcp-25afcc97ae62)  
25. Mastering Multi-Tenant Agent Deployment Patterns | Sparkco AI, acceso: febrero 14, 2026, [https://sparkco.ai/blog/mastering-multi-tenant-agent-deployment-patterns](https://sparkco.ai/blog/mastering-multi-tenant-agent-deployment-patterns)  
26. Security, Privacy & Data Isolation in AI Agents | by Pawan Kumar ..., acceso: febrero 14, 2026, [https://medium.com/illuminations-mirror/security-privacy-data-isolation-in-ai-agents-2161be1ee79b](https://medium.com/illuminations-mirror/security-privacy-data-isolation-in-ai-agents-2161be1ee79b)  
27. Three Types of AI Agent Memory, acceso: febrero 14, 2026, [https://cobusgreyling.medium.com/three-types-of-ai-agent-memory-4fd33457a821](https://cobusgreyling.medium.com/three-types-of-ai-agent-memory-4fd33457a821)  
28. How to Build AI Agents with Redis Memory Management, acceso: febrero 14, 2026, [https://redis.io/blog/build-smarter-ai-agents-manage-short-term-and-long-term-memory-with-redis/](https://redis.io/blog/build-smarter-ai-agents-manage-short-term-and-long-term-memory-with-redis/)  
29. How does AI Agent isolate data in a multi-tenant environment? \- Tencent Cloud, acceso: febrero 14, 2026, [https://www.tencentcloud.com/techpedia/126617](https://www.tencentcloud.com/techpedia/126617)  
30. Amazon Bedrock AgentCore Memory: Building context-aware agents | Artificial Intelligence, acceso: febrero 14, 2026, [https://aws.amazon.com/blogs/machine-learning/amazon-bedrock-agentcore-memory-building-context-aware-agents/](https://aws.amazon.com/blogs/machine-learning/amazon-bedrock-agentcore-memory-building-context-aware-agents/)  
31. Schema Design for Agent Memory and LLM History | by Pranav Prakash I GenAI I AI/ML I DevOps I | Medium, acceso: febrero 14, 2026, [https://medium.com/@pranavprakash4777/schema-design-for-agent-memory-and-llm-history-38f5cbc126fb](https://medium.com/@pranavprakash4777/schema-design-for-agent-memory-and-llm-history-38f5cbc126fb)  
32. Effective context engineering for AI agents \- Anthropic, acceso: febrero 14, 2026, [https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents)  
33. Architecting efficient context-aware multi-agent framework for production \- Google for Developers Blog, acceso: febrero 14, 2026, [https://developers.googleblog.com/architecting-efficient-context-aware-multi-agent-framework-for-production/](https://developers.googleblog.com/architecting-efficient-context-aware-multi-agent-framework-for-production/)  
34. Towards Generalized Routing: Model and Agent Orchestration for Adaptive and Efficient Inference \- arXiv, acceso: febrero 14, 2026, [https://arxiv.org/html/2509.07571v2](https://arxiv.org/html/2509.07571v2)  
35. The Complete Guide to Agentic AI (PART \#3): Advanced Multi-Agent Orchestration & Production…, acceso: febrero 14, 2026, [https://bishalbose294.medium.com/the-complete-guide-to-agentic-ai-part-3-advanced-multi-agent-orchestration-production-42c0ffb18033](https://bishalbose294.medium.com/the-complete-guide-to-agentic-ai-part-3-advanced-multi-agent-orchestration-production-42c0ffb18033)  
36. Multi-Agent AI Systems: Orchestrating AI Workflows \- V7 Go, acceso: febrero 14, 2026, [https://www.v7labs.com/blog/multi-agent-ai](https://www.v7labs.com/blog/multi-agent-ai)  
37. The AI Agent Framework Landscape in 2025: What Changed and What Matters \- Medium, acceso: febrero 14, 2026, [https://medium.com/@hieutrantrung.it/the-ai-agent-framework-landscape-in-2025-what-changed-and-what-matters-3cd9b07ef2c3](https://medium.com/@hieutrantrung.it/the-ai-agent-framework-landscape-in-2025-what-changed-and-what-matters-3cd9b07ef2c3)  
38. Designing a multi-stage real-estate LLM agent: single brain with tools vs. orchestrator \+ sub-agents? : r/AI\_Agents \- Reddit, acceso: febrero 14, 2026, [https://www.reddit.com/r/AI\_Agents/comments/1kw35g6/designing\_a\_multistage\_realestate\_llm\_agent/](https://www.reddit.com/r/AI_Agents/comments/1kw35g6/designing_a_multistage_realestate_llm_agent/)  
39. Four Design Patterns for Event-Driven, Multi-Agent Systems \- Confluent, acceso: febrero 14, 2026, [https://www.confluent.io/blog/event-driven-multi-agent-systems/](https://www.confluent.io/blog/event-driven-multi-agent-systems/)  
40. Stateful vs Stateless AI Agents: Which Architecture Fits Your System? | Tacnode Blog, acceso: febrero 14, 2026, [https://tacnode.io/post/stateful-vs-stateless-ai-agents-practical-architecture-guide-for-developers](https://tacnode.io/post/stateful-vs-stateless-ai-agents-practical-architecture-guide-for-developers)  
41. 7 Tips to Build Self-Improving AI Agents with Feedback Loops | Datagrid, acceso: febrero 14, 2026, [https://datagrid.com/blog/7-tips-build-self-improving-ai-agents-feedback-loops](https://datagrid.com/blog/7-tips-build-self-improving-ai-agents-feedback-loops)  
42. A Detailed Comparison of Top 6 AI Agent Frameworks in 2026 \- Turing, acceso: febrero 14, 2026, [https://www.turing.com/resources/ai-agent-frameworks](https://www.turing.com/resources/ai-agent-frameworks)  
43. Comparing Open-Source AI Agent Frameworks \- Langfuse Blog, acceso: febrero 14, 2026, [https://langfuse.com/blog/2025-03-19-ai-agent-comparison](https://langfuse.com/blog/2025-03-19-ai-agent-comparison)  
44. Agentic AI Frameworks | 2025 \- Flobotics, acceso: febrero 14, 2026, [https://flobotics.io/blog/agentic-ai-frameworks/](https://flobotics.io/blog/agentic-ai-frameworks/)  
45. AI Agent Frameworks: The Definitive Comparison for Builders in 2026 \- Arsum, acceso: febrero 14, 2026, [https://arsum.com/blog/posts/ai-agent-frameworks/](https://arsum.com/blog/posts/ai-agent-frameworks/)