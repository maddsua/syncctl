VERSION ?= 0.0.0
GOOS ?= linux
GOARCH ?= amd64
CGO_ENABLED ?= 0
ANDROID_API ?= 21
ANDROID_NDK_HOME ?= ''

ifeq ($(GOOS),android)
	ifeq ($(ANDROID_NDK_HOME),)
		$(error ANDROID_NDK_HOME is not set)
	endif

	TOOLCHAIN := $(ANDROID_NDK_HOME)/toolchains/llvm/prebuilt/linux-x86_64/bin

	ifeq ($(GOARCH),arm64)
		CC_PATH := $(TOOLCHAIN)/aarch64-linux-android$(ANDROID_API)-clang
	else ifeq ($(GOARCH),arm)
		CC_PATH := $(TOOLCHAIN)/armv7a-linux-androideabi$(ANDROID_API)-clang
	else
		$(error Unsupported Android architecture: $(GOARCH))
	endif
endif

BINARY_NAME=cli-$(GOOS)-$(GOARCH)-$(VERSION)

dist/$(BINARY_NAME):
	@echo "Building for $(GOOS)/$(GOARCH) (CGO_ENABLED=$(CGO_ENABLED))"
	mkdir -p dist
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) CC=$(CC_PATH) go build -v -ldflags "-s -w" -o dist/$(BINARY_NAME) ./cmd

cli: dist/$(BINARY_NAME)
