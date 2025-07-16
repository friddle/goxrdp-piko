# Makefile for goxrdp-piko-client
SHELL=/bin/bash
# 支持多平台编译

# 变量定义
BINARY_NAME=goxrdp
GUI_BINARY_NAME=goxrdp-gui
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 远程服务器配置变量
REMOTE_SERVER_HOST?=piko-upstream.friddle.me
REMOTE_SERVER_UPSTREAM_PORT?=8022
REMOTE_SERVER_BUSINESS_PORT?=8088
REMOTE_SERVER_URL?=https://$(REMOTE_SERVER_HOST):$(REMOTE_SERVER_UPSTREAM_PORT)
ACCESS_URL?=https://$(REMOTE_SERVER_HOST):$(REMOTE_SERVER_BUSINESS_PORT)

# Go 相关变量
GO=go
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
CGO_ENABLED?=1

# Android SDK 相关变量
ANDROID_SDK?=$(HOME)/sdk/Android/Sdk
ANDROID_NDK=$(ANDROID_SDK)/ndk/default

# 编译参数
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT} -s -w"

# GUI 编译参数 - 包含远程服务器配置
GUI_LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT} -s -w"

# 输出目录
DIST_DIR=dist
BUILD_DIR=dist

# 支持的平台
PLATFORMS=linux/amd64 windows/amd64 

# 默认目标
.PHONY: all
all: build build-gui

# 构建当前平台
.PHONY: build
build:
	@echo "构建 ${BINARY_NAME} for ${GOOS}/${GOARCH}..."
	@mkdir -p ${BUILD_DIR}
	CGO_ENABLED=${CGO_ENABLED} IMAGE_DEBUG=false ${GO} build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ./main.go
	@echo "构建完成: ${BUILD_DIR}/${BINARY_NAME}"

# 构建GUI客户端
.PHONY: build-gui
build-gui:
	@echo "构建 ${GUI_BINARY_NAME} for ${GOOS}/${GOARCH}..."
	@echo "远程服务器配置:"
	@echo "  上游端口: $(REMOTE_SERVER_UPSTREAM_PORT)"
	@echo "  业务端口: $(REMOTE_SERVER_BUSINESS_PORT)"
	@echo "  远程服务器URL: $(REMOTE_SERVER_URL)"
	@echo "  访问URL: $(ACCESS_URL)"
	@mkdir -p ${BUILD_DIR}
	CGO_ENABLED=${CGO_ENABLED} IMAGE_DEBUG=false \
	GOXRDP_REMOTE_SERVER=$(REMOTE_SERVER_URL) \
	GOXRDP_ACCESS_URL=$(ACCESS_URL) \
	${GO} build ${GUI_LDFLAGS} -o ${BUILD_DIR}/${GUI_BINARY_NAME} ./guiclient/main.go
	@echo "GUI构建完成: ${BUILD_DIR}/${GUI_BINARY_NAME}"

# 构建所有平台
.PHONY: build-all
build-all: 
	@echo "构建所有平台的 ${BINARY_NAME}..."
	@mkdir -p ${DIST_DIR}
	@for platform in ${PLATFORMS}; do \
		IFS='/' read -r os arch <<< "$$platform"; \
		echo "构建 $$os/$$arch..."; \
		if [ "$$os" = "windows" ]; then \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=1 IMAGE_DEBUG=false BUILD_ENV=prod \
			CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
			${GO} build ${LDFLAGS} -o ${DIST_DIR}/${BINARY_NAME}-$$os-$$arch.exe ./main.go; \
		else \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=1 IMAGE_DEBUG=false BUILD_ENV=prod \
			${GO} build ${LDFLAGS} -o ${DIST_DIR}/${BINARY_NAME}-$$os-$$arch ./main.go; \
		fi; \
	done
	@echo "所有平台构建完成，输出目录: ${DIST_DIR}"

# 构建所有平台的GUI客户端
.PHONY: build-gui-all
build-gui-all:
	@echo "构建所有平台的 ${GUI_BINARY_NAME}..."
	@mkdir -p ${DIST_DIR}
	@for platform in ${PLATFORMS}; do \
		IFS='/' read -r os arch <<< "$$platform"; \
		echo "构建GUI $$os/$$arch..."; \
		if [ "$$os" = "windows" ]; then \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=1 IMAGE_DEBUG=false BUILD_ENV=prod \
			CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
			GOXRDP_REMOTE_SERVER=$(REMOTE_SERVER_URL) \
			GOXRDP_ACCESS_URL=$(ACCESS_URL) \
			${GO} build ${GUI_LDFLAGS} -o ${DIST_DIR}/${GUI_BINARY_NAME}-$$os-$$arch.exe ./guiclient/main.go; \
		else \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=1 IMAGE_DEBUG=false BUILD_ENV=prod \
			GOXRDP_REMOTE_SERVER=$(REMOTE_SERVER_URL) \
			GOXRDP_ACCESS_URL=$(ACCESS_URL) \
			${GO} build ${GUI_LDFLAGS} -o ${DIST_DIR}/${GUI_BINARY_NAME}-$$os-$$arch ./guiclient/main.go; \
		fi; \
	done
	@echo "所有平台GUI构建完成，输出目录: ${DIST_DIR}"

# 构建特定平台
.PHONY: build-linux
build-linux:
	@echo "构建 Linux 版本..."
	@mkdir -p ${DIST_DIR}
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 IMAGE_DEBUG=false BUILD_ENV=prod ${GO} build ${LDFLAGS} -o ${DIST_DIR}/${BINARY_NAME}-linux-amd64 ./main.go
	@echo "Linux 版本构建完成"

.PHONY: build-windows
build-windows:
	@echo "构建 Windows 版本..."
	@mkdir -p ${DIST_DIR}
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 IMAGE_DEBUG=false BUILD_ENV=prod \
	CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
	${GO} build ${LDFLAGS} -o ${DIST_DIR}/${BINARY_NAME}-windows-amd64.exe ./main.go
	@echo "Windows 版本构建完成"

# 构建特定平台的GUI客户端
.PHONY: build-gui-linux
build-gui-linux:
	@echo "构建 Linux GUI 版本..."
	@mkdir -p ${DIST_DIR}
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 IMAGE_DEBUG=false BUILD_ENV=prod \
	GOXRDP_REMOTE_SERVER=$(REMOTE_SERVER_URL) \
	GOXRDP_ACCESS_URL=$(ACCESS_URL) \
	${GO} build ${GUI_LDFLAGS} -o ${DIST_DIR}/${GUI_BINARY_NAME}-linux-amd64 ./guiclient/main.go
	@echo "Linux GUI 版本构建完成"

.PHONY: build-gui-windows
build-gui-windows:
	@echo "构建 Windows GUI 版本..."
	@mkdir -p ${DIST_DIR}
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 IMAGE_DEBUG=false BUILD_ENV=prod \
	CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
	GOXRDP_REMOTE_SERVER=$(REMOTE_SERVER_URL) \
	GOXRDP_ACCESS_URL=$(ACCESS_URL) \
	${GO} build ${GUI_LDFLAGS} -o ${DIST_DIR}/${GUI_BINARY_NAME}-windows-amd64.exe ./guiclient/main.go
	@echo "Windows GUI 版本构建完成"

# Android 编译目标
.PHONY: build-android
build-android:
	@echo "构建所有 Android API 版本 (28-35)..."
	@mkdir -p ${DIST_DIR}
	@if [ ! -d "$(ANDROID_NDK)" ]; then \
		echo "错误: Android NDK 未找到，请检查 ANDROID_SDK 环境变量"; \
		echo "当前 ANDROID_SDK: $(ANDROID_SDK)"; \
		exit 1; \
	fi
	@for api in 28 29 30 31 32 33 34 35; do \
		echo "构建 Android API $$api 版本..."; \
		GOOS=android GOARCH=arm64 CGO_ENABLED=1 IMAGE_DEBUG=false \
		CC=$(ANDROID_NDK)/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android$$api-clang \
		CXX=$(ANDROID_NDK)/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android$$api-clang++ \
		${GO} build ${LDFLAGS} -o ${DIST_DIR}/${BINARY_NAME}-android-arm64-api$$api ./main.go; \
	done
	@echo "所有 Android API 版本构建完成，输出目录: ${DIST_DIR}"

# Debug 构建目标
.PHONY: build-debug
build-debug:
	@echo "构建 Debug 版本 (启用 DEBUG 输出)..."
	@mkdir -p ${BUILD_DIR}
	CGO_ENABLED=${CGO_ENABLED} IMAGE_DEBUG=true ${GO} build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-debug ./main.go
	@echo "Debug 版本构建完成: ${BUILD_DIR}/${BINARY_NAME}-debug"

# Debug GUI 构建目标
.PHONY: build-gui-debug
build-gui-debug:
	@echo "构建 Debug GUI 版本 (启用 DEBUG 输出)..."
	@mkdir -p ${BUILD_DIR}
	CGO_ENABLED=${CGO_ENABLED} IMAGE_DEBUG=true \
	GOXRDP_REMOTE_SERVER=$(REMOTE_SERVER_URL) \
	GOXRDP_ACCESS_URL=$(ACCESS_URL) \
	${GO} build ${GUI_LDFLAGS} -o ${BUILD_DIR}/${GUI_BINARY_NAME}-debug ./guiclient/main.go
	@echo "Debug GUI 版本构建完成: ${BUILD_DIR}/${GUI_BINARY_NAME}-debug"

# 清理构建文件
.PHONY: clean
clean:
	@echo "清理构建文件..."
	@rm -rf ${BUILD_DIR} ${DIST_DIR}
	@echo "清理完成"

# 显示配置信息
.PHONY: config
config:
	@echo "当前构建配置:"
	@echo "  远程服务器主机: $(REMOTE_SERVER_HOST)"
	@echo "  上游端口: $(REMOTE_SERVER_UPSTREAM_PORT)"
	@echo "  业务端口: $(REMOTE_SERVER_BUSINESS_PORT)"
	@echo "  远程服务器URL: $(REMOTE_SERVER_URL)"
	@echo "  访问URL: $(ACCESS_URL)"
	@echo "  目标平台: $(GOOS)/$(GOARCH)"
	@echo "  CGO启用: $(CGO_ENABLED)"

# 帮助
.PHONY: help
help:
	@echo "可用的 Make 目标:"
	@echo "  build        - 构建当前平台版本 (默认不启用 DEBUG)"
	@echo "  build-gui    - 构建GUI客户端版本"
	@echo "  build-debug  - 构建 Debug 版本 (启用 DEBUG 输出)"
	@echo "  build-gui-debug - 构建 Debug GUI 版本"
	@echo "  build-all    - 构建所有平台版本 (默认不启用 DEBUG)"
	@echo "  build-gui-all - 构建所有平台GUI版本"
	@echo "  build-linux  - 构建 Linux 版本"
	@echo "  build-gui-linux - 构建 Linux GUI 版本"
	@echo "  build-windows - 构建 Windows 版本"
	@echo "  build-gui-windows - 构建 Windows GUI 版本"
	@echo "  build-android - 构建 Android 版本"
	@echo "  clean        - 清理构建文件"
	@echo "  config       - 显示当前配置信息"
	@echo "  help         - 显示此帮助信息"
	@echo ""
	@echo "环境变量:"
	@echo "  REMOTE_SERVER_HOST - 远程服务器主机名 (默认: piko-upstream.friddle.me)"
	@echo "  REMOTE_SERVER_UPSTREAM_PORT - 上游端口 (默认: 8022)"
	@echo "  REMOTE_SERVER_BUSINESS_PORT - 业务端口 (默认: 8088)"
	@echo "  REMOTE_SERVER_URL - 完整的远程服务器URL (自动生成)"
	@echo "  ACCESS_URL - 访问URL (自动生成)"
	@echo ""
	@echo "使用示例:"
	@echo "  make build-gui REMOTE_SERVER_HOST=my-server.com REMOTE_SERVER_UPSTREAM_PORT=9000"
	@echo "  make build-gui-all REMOTE_SERVER_BUSINESS_PORT=9090"
	@echo ""
	@echo "Windows 交叉编译要求:"
	@echo "  需要安装 MinGW 交叉编译器: sudo pacman -S mingw-w64-gcc" 