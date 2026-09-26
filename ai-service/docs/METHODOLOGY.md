# Metodología del motor de anomalías

Este documento explica **cómo** calcula el motor cada resultado y **qué principio** (estadístico, eléctrico o de diseño) respalda cada decisión. Todas las cifras de ejemplo salen del dataset real (12 medidores, 1–14 de septiembre de 2026, lecturas horarias).

## 1. Estadística robusta

### El problema

Para decidir si un valor es "anormal" hay que compararlo con lo "normal". La estadística clásica usa **media** y **desviación estándar**, pero ambas se contaminan con los mismos valores anómalos que queremos detectar:

```
consumos:  43, 44, 44, 45, 110
media    = 57.2    ← arrastrada por el 110
mediana  = 44      ← prácticamente no se mueve
```

Si el valor de referencia se contamina, la anomalía se "esconde a sí misma". Por eso el motor usa **estadística robusta**: medidas con un *punto de ruptura* alto, es decir, que toleran una fracción grande de valores extremos sin distorsionarse (la mediana tolera hasta el 50%).

### Mediana y MAD

- **Mediana:** el valor central de los datos ordenados. Sustituye a la media.
- **MAD** (*Median Absolute Deviation*): la mediana de las distancias de cada valor a la mediana. Sustituye a la desviación estándar.

```
valores:     43, 44, 44, 45, 110
mediana:     44
distancias:   1,  0,  0,  1,  66
MAD:          1        ← el 66 no la afecta
```

### Z-score robusto

Mide cuántas "desviaciones típicas robustas" se aleja un valor del centro:

```
z = (valor − mediana) / (1.4826 × MAD)
```

La constante **1.4826** hace que, para datos con distribución normal, `1.4826 × MAD` sea un estimador consistente de la desviación estándar. Así un z robusto de 3 se interpreta igual que en la estadística clásica: un valor que el azar produciría muy raramente. Si `MAD = 0` (todos los valores del baseline idénticos), el z se define como 0 para evitar la división por cero.

### Variación porcentual

```
variación % = (observado − baseline) / baseline × 100
```

Ejemplos del dataset: M-109 **+109,8%**, M-104 **+47,2%**, caída de M-106 **−85,8%**. Es la misma fórmula de la función SQL `fn_meter_consumption_stats` del backend, de modo que las cifras del motor y del dashboard coinciden.

## 2. Baseline: qué es "normal" para cada medidor

### Principio

**"Normal" es relativo a cada medidor, no absoluto.** M-107 consume ~20 kWh/h y M-106 ~56 kWh/h, y ambos son normales. Por eso cada medidor se compara contra su **propio** comportamiento histórico, nunca contra otros medidores ni contra un umbral fijo de consumo.

### Ventana de referencia

El baseline es el **primer tramo de datos del medidor**: 7 días, o la mitad de la serie si el medidor tiene menos de 14 días de historia (así siempre queda un tramo posterior que analizar).

- Una semana completa cubre **todos los días de la semana**, lo que captura el ciclo operativo completo de una planta (días laborables y fines de semana).
- Es la misma ventana que usa el backend para el `baselineKwh` de la tabla de medidores y para la curva "normal" de la gráfica: todas las pantallas cuentan la misma historia.
- Supuesto explícito: el primer tramo representa operación normal. En el dataset se cumple para los cuatro casos de interés, cuyos cambios ocurren a partir del 8 de septiembre (M-106 el día 8, M-104 el 11, M-109 el 12, M-112 el 13).

### Baseline por hora del día (en hora local)

El consumo industrial sigue un ciclo diario. Perfil normal de M-109 (mediana de la primera semana):

```
03:00  33.2 kWh   carga base nocturna
08:00  51.3 kWh   turno de día
14:00  51.7 kWh   turno de día
19:00  44.2 kWh   turno de tarde
```

Comparar cada lectura con **la mediana de su misma hora** evita dos errores: marcar como anómalo el pico normal del turno de día, o no ver un aumento nocturno que, comparado contra el promedio diario, parecería normal.

Las horas se calculan **en la zona horaria del sitio** (`timezone` de la petición, por defecto `America/Bogota`). Los datos viajan en UTC; sin esta conversión, "las 14:00" serían las 14:00 UTC (09:00 en Bogotá) y el perfil quedaría desplazado 5 horas.

Con 7 días de baseline, cada hora tiene **7 muestras**. La MAD por hora mide la estabilidad de esa franja: en M-109, entre 0,6 y 1,4 kWh, es decir, un comportamiento muy regular.

### Referencias eléctricas

Además del consumo, el baseline guarda la **mediana** de voltaje, corriente y factor de potencia de la ventana de referencia. M-109: 219,7 V · 195,9 A · PF 0,94. Los detectores eléctricos comparan contra estos valores.

## 3. Detectores

Cada detector busca **un tipo de hecho** y produce **señales** (`Signal`). Un detector no clasifica: solo observa. La clasificación combina señales y contexto. Todos implementan la misma interfaz (`Detector.detect(series) -> list[Signal]`), lo que permite añadir detectores nuevos sin modificar los existentes.

### 3.1 Aumento sostenido de consumo (CUSUM)

**Objetivo:** detectar un **cambio de nivel sostenido** (M-109, M-104) y **el momento exacto** en que empezó, ignorando picos aislados y la variación normal.

**Principio: CUSUM** (*Cumulative Sum*, Page, 1954), técnica clásica de control estadístico de procesos. En lugar de evaluar cada lectura por separado, **acumula** las desviaciones: un pico aislado suma poco y se "disipa", mientras que un desplazamiento sostenido suma hora tras hora hasta cruzar un umbral.

```
desviación relativa:  d_t = (consumo_t − esperado_t) / esperado_t
suma acumulada:       S_t = max(0, S_{t−1} + d_t − k)
alarma cuando:        S_t > h
```

- `esperado_t` es la mediana del baseline para **la misma hora local** (sección 2).
- `k` (*drift*) es la desviación "tolerada" por hora: lo que la resta impide acumular. Se fija en la **mitad del cambio mínimo que interesa detectar** (regla estándar de diseño de CUSUM, `k = δ/2`): para cambios ≥ 25%, `k = 0,125`.
- `max(0, ...)` reinicia la suma cuando el consumo vuelve a lo normal, así el ruido no se acumula indefinidamente.
- `h = 1,0`: la alarma salta cuando el exceso acumulado sobre la tolerancia equivale al 100% de una hora de consumo.
- **Punto de cambio:** la primera hora del tramo en que `S` dejó de ser 0 y creció sin interrupción hasta la alarma.

**Por qué desviación relativa (%) y no z-score.** La versión clásica normaliza por el ruido (`d_t / σ`). Se probó con los datos reales y falló: los perfiles horarios son tan regulares (ruido ≈ 1 kWh) que variaciones irrelevantes del 1–3% acumulaban alarmas en 10 de 12 medidores, y M-109 se detectaba 3 días antes de su salto real. Expresar la desviación en **porcentaje del esperado** alinea el detector con la pregunta de negocio ("¿subió el consumo un 25% o más?") y hace el umbral comparable entre medidores grandes y pequeños.

**Validaciones posteriores**, para confirmar que el cambio es real y sostenido:
- el tramo desde el punto de cambio dura **al menos 24 h**;
- la mediana del consumo en ese tramo supera la mediana esperada en **al menos un 25%**.

**Resultado con el dataset:**

| Medidor | Punto de cambio | Variación | CUSUM máximo |
|---|---|---|---|
| M-109 | 12-sep 14:00 | +116,9% | 1,01 (alarma) |
| M-104 | 11-sep 00:00 | +49,0% | 1,02 (alarma) |
| Resto | — | — | ≤ 0,09 |

Los puntos de cambio coinciden **exactamente** con los eventos registrados (`UNKNOWN` de M-109 y `OPERATIONAL_CHANGE` de M-104), y los medidores normales quedan un orden de magnitud por debajo del umbral.

### 3.2 Caída de consumo (rachas consecutivas)

**Objetivo:** detectar períodos en que el medidor consume **muy por debajo** de lo normal (M-106) y registrar **cuándo empezó y cuándo se recuperó**.

**Principio: detección por rachas** (*run-length*). Una hora baja aislada puede ser ruido o un corte breve de la red; varias horas **consecutivas** bajas indican un estado distinto del equipo (apagado, mantenimiento, falla). Exigir una racha mínima filtra los eventos triviales sin necesidad de promediar.

```
hora baja:   consumo_t < 50% × esperado_t
caída:       ≥ 3 horas bajas consecutivas
fin:         la primera hora que vuelve a superar el 50% (recuperación)
```

- Se compara contra el baseline **de la misma hora local**, igual que el detector de aumentos: consumir poco de madrugada es normal, a mediodía no.
- A diferencia de CUSUM, aquí interesa la **duración exacta**, no solo el inicio: registra `started_at` y `ended_at` (la hora de recuperación, con el mismo criterio de intervalo semiabierto `[inicio, fin)` del backend). Si la caída sigue activa al final de los datos, `ended_at` queda vacío.
- Un medidor puede tener **varias caídas** separadas; cada una produce su propia señal.

**Calibración con el dataset:** en los medidores normales, el consumo nunca baja del **85%** de lo esperado. La caída de M-106 se mantiene entre el **17% y el 21%**. El umbral del 50% queda con amplio margen respecto a ambos grupos.

**Resultado:** una sola caída, en M-106: del 8-sep 00:00 al 8-sep 12:00 (**12 horas**, ~80% por debajo de lo esperado). Coincide exactamente con el evento `SCHEDULED_OUTAGE` ("Scheduled maintenance outage for 12 hours"). El detector **no decide** que sea un falso positivo: solo reporta la caída y su duración. Esa conclusión es de la clasificación, que la cruza con el evento.

### 3.3 Voltaje: fuera de tolerancia e inestabilidad

**Principio eléctrico.** El voltaje lo impone la **red**, no la carga: con una alimentación sana se mantiene cerca de su valor nominal aunque el consumo cambie. Por eso un voltaje anómalo apunta a un problema de **suministro** o, si el consumo se mantiene estable, de **medición** (sensor, transformador de medida, conexiones). A diferencia del consumo, aquí la referencia **no es el baseline del medidor sino su especificación**: el voltaje nominal (`nominalVoltage`, 220 V por defecto).

El detector busca dos fallas distintas, que producen dos tipos de señal:

**a) Fuera de tolerancia (`VOLTAGE_OUT_OF_RANGE`)**

```
lectura fuera de tolerancia:  |voltaje − nominal| / nominal > 5%     → a 220 V: fuera de [209, 231] V
señal:                        ≥ 3 lecturas fuera de tolerancia
magnitud:                     la mayor desviación, en % del nominal
```

±5% es la tolerancia de servicio habitual en las normas de suministro (por ejemplo, ANSI C84.1, rango A). Algunas normas nacionales admiten un margen mayor por debajo del nominal; ±5% es un criterio conservador que no afecta el resultado en este dataset, donde los medidores sanos se mantienen entre 216 y 225 V.

**b) Inestabilidad (`VOLTAGE_INSTABILITY`)**

```
salto:     |voltaje_t − voltaje_{t−1}| > 5% del nominal     → a 220 V: más de 11 V de una hora a la siguiente
señal:     ≥ 3 saltos
magnitud:  el mayor salto (V), comparado con el paso típico (mediana de los pasos)
```

Mide la **variabilidad entre horas consecutivas**, no el nivel. Detecta un voltaje que oscila aunque cada lectura, por separado, pudiera estar dentro de la tolerancia. Una red real no alterna decenas de voltios de una hora a otra con una carga estable: ese patrón es típico de una conexión intermitente o de un sensor defectuoso.

**Duración del episodio.** Como las fallas son intermitentes (alternan lecturas buenas y malas), un episodio se considera **activo** si la última ocurrencia está dentro de las últimas 24 h de datos. Si no, se cierra una hora después de la última ocurrencia. Esta regla es común a los detectores eléctricos (`intermittent_episode_end`).

**Umbrales relativos al nominal.** Tolerancia y salto se expresan en % del voltaje nominal del medidor, así el mismo detector sirve para medidores de 120, 220 o 440 V.

**Calibración y resultado con el dataset:**

| | Lecturas fuera de ±5% | Saltos > 11 V | Salto máximo | Paso típico |
|---|---|---|---|---|
| 11 medidores sanos | 0 | 0 | 4,6 – 5,6 V | ~1,5 V |
| M-112 | **16** (peor: 241,2 V, +9,6%) | **32** | **25,5 V** | 1,7 V |

Todo ocurre el 13 y 14 de septiembre, en coincidencia con el evento `DATA_QUALITY` del 13-sep ("Intermittent readings and abnormal electrical jumps"). El episodio sigue activo al final de los datos.

### 3.4 Factor de potencia bajo

**Principio eléctrico.** En corriente alterna, no toda la corriente que circula hace trabajo útil:

```
potencia aparente  S = V × I            (kVA)  lo que la red debe entregar
potencia activa    P = V × I × PF       (kW)   lo que hace trabajo útil (y se mide en kWh)
potencia reactiva  Q = √(S² − P²)       (kvar) la que magnetiza motores y transformadores
factor de potencia PF = P / S = cos φ          fracción útil de la corriente (0 a 1)
```

Las cargas **inductivas** (motores, compresores, bombas) necesitan energía reactiva para crear su campo magnético; los **bancos de condensadores** la compensan. Un PF bajo significa que, para entregar la misma energía útil, circula más corriente: más pérdidas en los cables, transformadores más cargados y, habitualmente, **recargos de la distribuidora por energía reactiva** (en Colombia, la regulación cobra la reactiva que excede el 50% de la activa, equivalente a PF ≈ 0,89).

Un PF que **cae de forma brusca** en un equipo que antes era estable suele indicar un **motor sobrecargado o con fallas**, o un **banco de condensadores dañado o desconectado**.

**Regla:**

```
lectura con PF bajo:   PF < 0,80
señal:                 ≥ 3 lecturas con PF bajo
magnitud:              variación de la mediana del PF bajo respecto al PF del baseline
persistencia:          lecturas con PF bajo / horas transcurridas desde la primera
```

El umbral de 0,80 marca un PF **claramente anormal** para una instalación industrial compensada, con margen respecto al mínimo de los medidores sanos (0,86).

**Persistencia: continuo frente a intermitente.** Además de *cuánto* baja el PF, importa *cómo*: un PF bajo en **todas** las horas desde que empezó describe un equipo en falla sostenida; un PF bajo **salteado** entre horas normales es más propio de un problema de medición. El detector reporta ese porcentaje como evidencia para la clasificación.

**Resultado con el dataset:**

| | PF baseline | PF observado (mediana) | Mínimo | Horas con PF < 0,80 | Persistencia |
|---|---|---|---|---|---|
| 10 medidores sanos | 0,91 – 0,96 | — | ≥ 0,86 | 0 | — |
| M-109 | 0,94 | 0,74 (−21,3%) | 0,71 | 58 de 58 (desde 12-sep 14:00) | **100%** |
| M-112 | 0,95 | 0,65 (−31,7%) | 0,58 | 12 de 45 (desde 13-sep 03:00) | **27%** |

**La evidencia clave de M-109.** Compárese con M-104, que también aumentó su consumo (+49%) por una nueva línea de producción: su PF se mantiene en su rango normal (≥ 0,86). **Más producción con equipos sanos no degrada el PF**. En M-109, el consumo se duplica **y a la vez** el PF cae desde la misma hora. Esa combinación es la firma de un equipo que trabaja mal, no de una planta que produce más.

### 3.5 Sobrecorriente

**Principio eléctrico.** Como `P = V × I × PF`, la corriente sube cuando sube la potencia **o** cuando baja el factor de potencia:

```
I = P / (V × PF)
```

Conductores, interruptores y transformadores se dimensionan para una corriente máxima. Superar de forma sostenida la corriente que la instalación maneja habitualmente implica calentamiento, envejecimiento acelerado del aislamiento, riesgo de disparo de protecciones y, en el peor caso, de falla.

**Referencia: el pico histórico, no la mediana.** La corriente sigue el ciclo diario del consumo (M-109: mediana de 196 A, pero 242 A en su turno normal). Comparar contra la mediana marcaría cada mediodía como sobrecorriente. La pregunta correcta es si circula **más corriente que la máxima que la instalación manejó en su período normal** (su demanda máxima):

```
límite:     1,5 × pico histórico de la semana de referencia
señal:      ≥ 3 lecturas por encima del límite
magnitud:   variación de la mediana de esas lecturas respecto a la referencia
```

Si el medidor tiene una **corriente máxima nominal** configurada (`maxCurrent`), esa especificación reemplaza al pico histórico como límite: es la capacidad física real del equipo.

**Resultado con el dataset:**

| | Pico en baseline | Máximo posterior | Relación | Lecturas sobre el límite |
|---|---|---|---|---|
| 9 medidores sanos | — | — | ≈ 1,0× | 0 |
| M-112 | 156 A | 218 A | 1,39× | 0 |
| M-104 | 264 A | 388 A | 1,47× | 0 |
| M-109 | 242 A | **507 A** | **2,10×** | **46**, desde el 12-sep 14:00 |

M-109 supera el límite (363 A) en 46 horas; la mediana de esas horas es 485 A, **el doble** de su pico histórico. Son 46 y no las 58 horas del salto porque, de madrugada, la carga base sigue por debajo del límite aunque también esté elevada.

**Límite de este criterio (M-104).** La nueva línea de producción de M-104 lleva su corriente a 1,47× su pico, cerca del umbral. Es coherente: +47% de producción exige más corriente. El detector de sobrecorriente no es el que distingue a M-104 de M-109; esa distinción la dan el factor de potencia (sección 3.4) y la correlación con eventos. Si M-104 cruzara el umbral, su clasificación no cambiaría, porque su aumento está explicado por un `OPERATIONAL_CHANGE`.

## 4. Correlación con eventos

**Principio.** Una desviación solo es una anomalía si **no tiene explicación**. Los eventos operativos que registra la planta aportan ese contexto: el motor cruza cada señal con los eventos del **mismo medidor** para decidir si lo observado estaba previsto.

**Ventana temporal:** un evento se relaciona con una señal si ocurre dentro de **±6 horas** de su inicio. En el dataset, siete de las ocho señales comienzan a la misma hora que su evento; la octava (PF bajo de M-112) aparece 3 horas después, porque las fallas intermitentes tardan en manifestarse. La ventana cubre ese desfase sin relacionar sucesos de días distintos.

**Qué evento explica qué señal:**

| Señal | Eventos que la explican |
|---|---|
| Aumento de consumo | `OPERATIONAL_CHANGE` (nueva línea, cambio de turno, ampliación) |
| Caída de consumo | `SCHEDULED_OUTAGE`, `MAINTENANCE` |
| Señales eléctricas (voltaje, PF, corriente) | Ninguno: un evento no hace "normal" un voltaje de 241 V |

- La relación es **específica**: un corte programado no explica un aumento de consumo. Un evento del tipo equivocado cercano no basta.
- **`UNKNOWN` no explica nada.** Un registro de "No operational event reported" confirma justamente lo contrario: la planta **no** tiene una causa conocida para lo ocurrido. Es la clave de M-109.
- **`DATA_QUALITY` se relaciona, pero no explica.** No convierte las lecturas en normales; confirma que el problema está en la **medición**. La clasificación lo usa como evidencia de apoyo.
- Si hay varios eventos cercanos, se prefiere el que **explica** la señal y, entre iguales, el más cercano en el tiempo.

**Consistencia de duración.** Cuando el evento anuncia una duración ("... for 12 hours") y la señal ya terminó, se comparan ambas con una tolerancia de ±1 h. Que un corte anunciado de 12 horas coincida con una caída observada de 12 horas es evidencia fuerte de que se trata exactamente del evento planificado; si la caída hubiera durado 20 horas, algo más habría ocurrido y la explicación sería solo parcial.

**Resultado con el dataset:**

| Medidor | Señal | Evento relacionado | ¿Explica? | Duración |
|---|---|---|---|---|
| M-104 | Aumento +49% | `OPERATIONAL_CHANGE` (misma hora) | **Sí** | — |
| M-106 | Caída de 12 h | `SCHEDULED_OUTAGE` "for 12 hours" (misma hora) | **Sí** | **Coincide** |
| M-109 | Aumento, PF bajo, sobrecorriente | `UNKNOWN` "No operational event reported" | **No** | — |
| M-112 | Voltaje fuera de rango e inestable, PF bajo | `DATA_QUALITY` | No (confirma problema de medición) | — |

## 5. Clasificación: sistema de reglas

**Principio.** La conclusión (qué tipo de anomalía es y qué tan grave) la toma un **sistema de reglas determinista**, no un modelo estadístico ni un LLM. Cada regla tiene un identificador (`ruleId`) que viaja con el hallazgo, así cada conclusión es **reproducible** (misma entrada, mismo resultado) y **auditable** (se sabe exactamente qué regla la produjo y con qué evidencia).

**Evidencia.** Cada medidor llega con una lista de *evidencias*: cada señal de los detectores junto con su correlación con eventos (sección 4).

**Evaluación en orden de prioridad, consumiendo señales.** Las reglas se evalúan en orden; cuando una se cumple, genera un hallazgo y **consume** las señales que usó. Así:

- una señal nunca se cuenta dos veces (el PF bajo de M-109 forma parte de su hallazgo, no genera otro);
- ninguna señal queda sin clasificar (la última regla recoge cualquier señal eléctrica suelta);
- un medidor con dos problemas independientes produce dos hallazgos.

| Regla | Condición | Tipo | Severidad |
|---|---|---|---|
| **R1_MEASUREMENT_FAULT** | Voltaje fuera de tolerancia o inestable **sin** cambio de consumo | `DATA_QUALITY` | HIGH |
| **R2_SCHEDULED_OUTAGE** | Caída explicada por un corte o mantenimiento, ya recuperada, con duración consistente con la anunciada | `FALSE_POSITIVE` | LOW |
| **R3_EXPLAINED_SURGE** | Aumento explicado por un `OPERATIONAL_CHANGE`, **sin** degradación del PF | `EXPLAINABLE_ANOMALY` | MEDIUM |
| **R4_UNEXPLAINED_SURGE** | Cualquier otro aumento: sin causa conocida, o con el PF degradado | `REAL_ANOMALY` | HIGH si hay señales de equipo (PF, corriente) o el aumento es ≥ 50%; si no, MEDIUM |
| **R5_UNEXPLAINED_DROP** | Cualquier otra caída: sin evento planificado, o más larga que lo anunciado | `REAL_ANOMALY` | HIGH si sigue activa; si no, MEDIUM |
| **R6_EQUIPMENT_FAULT** | PF bajo o sobrecorriente sin cambio de consumo | `REAL_ANOMALY` | MEDIUM |

**Por qué este orden y estas condiciones:**

- **R1 va primero** porque, si la medición no es confiable, ninguna otra conclusión sobre ese medidor lo es. La condición "sin cambio de consumo" es la **incoherencia física** que delata el problema de medición: la red no alterna entre 201 y 241 V cada hora mientras la carga consume exactamente lo mismo.
- **R2 exige tres cosas**, no solo "hay un evento": que el evento explique una caída, que el consumo ya se haya recuperado y que la duración no contradiga lo anunciado. Una caída que se prolonga más de lo planificado deja de ser un falso positivo (pasa a R5).
- **R3 exige un comportamiento eléctrico sano.** Un aumento con justificación operativa pero con el PF degradado no se da por explicado: la nueva carga podría estar dañando equipos. Pasa a R4.
- **R4 marca HIGH cuando hay señales de equipo** porque son la evidencia física de una falla (sección 3.4), no solo de más consumo.
- **R6 cierra el sistema:** garantiza que ninguna señal eléctrica quede sin un hallazgo.

**Resultado con el dataset:**

| Medidor | Regla | Tipo | Severidad | Señales usadas |
|---|---|---|---|---|
| M-109 | R4_UNEXPLAINED_SURGE | `REAL_ANOMALY` | HIGH | Aumento +117%, PF bajo, sobrecorriente |
| M-104 | R3_EXPLAINED_SURGE | `EXPLAINABLE_ANOMALY` | MEDIUM | Aumento +49% |
| M-106 | R2_SCHEDULED_OUTAGE | `FALSE_POSITIVE` | LOW | Caída de 12 h |
| M-112 | R1_MEASUREMENT_FAULT | `DATA_QUALITY` | HIGH | Voltaje fuera de rango, voltaje inestable, PF bajo |
| 8 restantes | — | — | — | Sin señales, sin hallazgos |
