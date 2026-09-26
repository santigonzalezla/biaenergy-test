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
