#!/bin/bash

# 启动 etcd (用于测试)
echo "Starting etcd for testing..."
docker run -d \
  --name etcd-test \
  --publish 2379:2379 \
  --publish 2380:2380 \
  --env ALLOW_NONE_AUTHENTICATION=yes \
  --env ETCD_ADVERTISE_CLIENT_URLS=http://0.0.0.0:2379 \
  bitnami/etcd:latest

echo "Waiting for etcd to start..."
sleep 5

# 启动 Control Plane with etcd backend
echo "Starting Control Plane with etcd backend..."
cd /home/huiguo/wasm_fault_injection/control-plane
./control-plane -storage=etcd -etcd-endpoints=localhost:2379 -listen=0.0.0.0:8080
