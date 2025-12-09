# Service Discovery Makefile

.PHONY: build run test clean docker-build docker-run lint fmt help k3s-import deploy undeploy restart logs status check-redis

# 变量
BINARY_NAME=service-discovery
DOCKER_IMAGE=service-discovery
VERSION?=latest
NAMESPACE=boifi

# 默认目标
help:
	@echo "Service Discovery 构建工具"
	@echo ""
	@echo "可用目标:"
	@echo "  build         构建二进制文件"
	@echo "  run           运行服务"
	@echo "  test          运行测试"
	@echo "  test-unit     运行单元测试"
	@echo "  test-cover    运行测试并生成覆盖率报告"
	@echo "  clean         清理构建产物"
	@echo "  docker-build  构建 Docker 镜像"
	@echo "  k3s-import    导入镜像到 k3s"
	@echo "  deploy        部署到 Kubernetes"
	@echo "  undeploy      从 Kubernetes 删除"
	@echo "  restart       重启 Deployment"
	@echo "  logs          查看日志"
	@echo "  status        查看状态"
	@echo "  check-redis   检查 Redis 数据"
	@echo "  lint          运行代码检查"
	@echo "  fmt           格式化代码"# 构建
build:
	@echo "🏗️ 构建 $(BINARY_NAME)..."
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/$(BINARY_NAME) ./cmd
	@echo "✅ 构建完成: bin/$(BINARY_NAME)"

# 运行
run: build
	@echo "🚀 启动服务..."
	./bin/$(BINARY_NAME)

# 单次运行
run-once: build
	@echo "🚀 单次运行..."
	./bin/$(BINARY_NAME) --once

# 测试
test:
	@echo "🧪 运行所有测试..."
	go test -v ./...

test-unit:
	@echo "🧪 运行单元测试..."
	go test -v ./tests/unit/...

test-integration:
	@echo "🧪 运行集成测试..."
	go test -v ./tests/integration/...

test-cover:
	@echo "🧪 运行测试并生成覆盖率..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ 覆盖率报告: coverage.html"

# 清理
clean:
	@echo "🧹 清理构建产物..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "✅ 清理完成"

# Docker
docker-build:
	@echo "🐳 构建 Docker 镜像..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	@echo "✅ 镜像构建完成: $(DOCKER_IMAGE):$(VERSION)"

docker-run:
	@echo "🐳 运行 Docker 容器..."
	docker run --rm \
		-e BOIFI_JAEGER_URL=http://jaeger-query:16686 \
		-e BOIFI_REDIS_ADDR=redis:6379 \
		$(DOCKER_IMAGE):$(VERSION)

# 代码质量
lint:
	@echo "🔍 运行代码检查..."
	golangci-lint run ./...

fmt:
	@echo "📝 格式化代码..."
	go fmt ./...
	@echo "✅ 格式化完成"

# 依赖管理
deps:
	@echo "📦 下载依赖..."
	go mod download
	go mod tidy
	@echo "✅ 依赖下载完成"

# Kubernetes 部署
k3s-import: docker-build
	@echo "📦 导入镜像到 k3s..."
	docker save $(DOCKER_IMAGE):$(VERSION) | sudo k3s ctr images import -
	@echo "✅ 镜像已导入 k3s"

deploy: k3s-import
	@echo "🚀 部署到 Kubernetes..."
	kubectl apply -f k8s/deployment.yaml
	@echo "⏳ 等待部署就绪..."
	kubectl rollout status deployment/service-discovery -n $(NAMESPACE) --timeout=120s
	@echo "✅ 部署完成！"

undeploy:
	@echo "🗑️ 删除 Kubernetes 部署..."
	kubectl delete -f k8s/deployment.yaml --ignore-not-found=true
	@echo "✅ 删除完成"

restart:
	@echo "🔄 重启服务..."
	kubectl rollout restart deployment/service-discovery -n $(NAMESPACE)
	kubectl rollout status deployment/service-discovery -n $(NAMESPACE)
	@echo "✅ 重启完成"

logs:
	@echo "📋 查看日志..."
	kubectl logs -f -n $(NAMESPACE) -l app=service-discovery --tail=100

status:
	@echo "📊 部署状态:"
	@echo ""
	kubectl get deployment -n $(NAMESPACE) service-discovery
	@echo ""
	kubectl get pods -n $(NAMESPACE) -l app=service-discovery
	@echo ""
	@echo "最近事件:"
	kubectl get events -n $(NAMESPACE) --sort-by='.lastTimestamp' | tail -10

check-redis:
	@echo "🔍 检查 Redis 中的服务地图..."
	kubectl exec -n $(NAMESPACE) deploy/redis -- redis-cli GET boifi:service-map | jq '{timestamp, service_count: (.services | length), edge_count: (.topology | length), services: (.services | keys)}'