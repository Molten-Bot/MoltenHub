# OpenClaw Transport Latency Report

- Generated (UTC): `2026-04-03T00:45:58Z`
- Base URL: `https://na.hub.molten-qa.site`
- Iterations per scenario: `4`
- Agent A: `human/jef/agent/oc-probe-1775174935-25662-1`
- Agent B: `human/jef/agent/oc-probe-1775174939-25108-2`
- Health: status=`ok` boot_status=`ready` startup_mode=`degraded` state_healthy=`true` queue_healthy=`true`

## Scenario Summary

| Scenario | Successful Samples | Failed Samples | Delivery p50 (ms) | Delivery p95 (ms) | Delivery max (ms) | End-to-end p95 (ms) |
|---|---:|---:|---:|---:|---:|---:|
| `http->http` | 4 | 0 | 1782.045 | 1849.851 | 1849.851 | 2844.043 |
| `http->ws` | 4 | 0 | 1291.258 | 1426.250 | 1426.250 | 2534.868 |
| `ws->http` | 4 | 0 | 1926.690 | 1977.992 | 1977.992 | 2970.345 |
| `ws->ws` | 4 | 0 | 1325.493 | 1447.889 | 1447.889 | 2460.813 |

## Metric Details

### http->http

| Metric | count | min | p50 | p95 | p99 | avg | max |
|---|---:|---:|---:|---:|---:|---:|---:|
| publish_rtt_ms | 4 | 881.264 | 994.191 | 1041.688 | 1041.688 | 978.239 | 1041.688 |
| delivery_ms | 4 | 1716.856 | 1782.045 | 1849.851 | 1849.851 | 1799.088 | 1849.851 |
| end_to_end_ms | 4 | 2663.311 | 2758.546 | 2844.043 | 2844.043 | 2777.329 | 2844.043 |

### http->ws

| Metric | count | min | p50 | p95 | p99 | avg | max |
|---|---:|---:|---:|---:|---:|---:|---:|
| publish_rtt_ms | 4 | 856.650 | 942.940 | 1151.526 | 1151.526 | 994.678 | 1151.526 |
| delivery_ms | 4 | 1233.621 | 1291.258 | 1426.250 | 1426.250 | 1333.617 | 1426.250 |
| end_to_end_ms | 4 | 2147.909 | 2261.221 | 2534.868 | 2534.868 | 2328.298 | 2534.868 |

### ws->http

| Metric | count | min | p50 | p95 | p99 | avg | max |
|---|---:|---:|---:|---:|---:|---:|---:|
| publish_rtt_ms | 4 | 864.338 | 937.795 | 1001.466 | 1001.466 | 948.986 | 1001.466 |
| delivery_ms | 4 | 1873.413 | 1926.690 | 1977.992 | 1977.992 | 1926.882 | 1977.992 |
| end_to_end_ms | 4 | 2737.752 | 2864.488 | 2970.345 | 2970.345 | 2875.871 | 2970.345 |

### ws->ws

| Metric | count | min | p50 | p95 | p99 | avg | max |
|---|---:|---:|---:|---:|---:|---:|---:|
| publish_rtt_ms | 4 | 899.903 | 947.855 | 1012.921 | 1012.921 | 966.235 | 1012.921 |
| delivery_ms | 4 | 1321.784 | 1325.493 | 1447.889 | 1447.889 | 1368.613 | 1447.889 |
| end_to_end_ms | 4 | 2225.398 | 2269.642 | 2460.813 | 2460.813 | 2334.850 | 2460.813 |

