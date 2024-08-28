# Environment Variables

## General

- `REDIS_PASSWORD`  
  Optional. The password for Redis.

- `REDIS_USE_SENTINEL`  
  Default: `true`. Set to `false` if not using Sentinel.

- `INTERVAL_TIME_MILLIS`  
  Default: `600000` The interval time for tests, in milliseconds.

## Sentinel Only

- `REDIS_MASTER_NAME`  
  The name of the Redis master.

- `SENTINEL_ADDRS`  
  Addresses of the Redis Sentinel instances.

## Single Host Only

- `REDIS_ADDR`  
  Address of the Redis server if not using Sentinel.


# Redis Benchmark Deployment Manifest

This document contains the Kubernetes deployment manifest for the `redis-benchmark` application.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis-benchmark
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis-benchmark
  template:
    metadata:
      labels:
        app: redis-benchmark
    spec:
      containers:
        - name: redis-benchmark
          image: lasdox/redis-benchmark:1.0.1-SNAPSHOT
          imagePullPolicy: IfNotPresent
          env:
            - name: SENTINEL_MASTER_NAME
              value: "harness-redis"
            - name: INTERVAL_TIME_MILLIS
              value: "10000"
            - name: SENTINEL_ADDRS
              value: "redis-sentinel-harness-announce-0:26379,redis-sentinel-harness-announce-1:26379,redis-sentinel-harness-announce-2:26379"
```
