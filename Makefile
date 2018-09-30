# Build

.PHONY: compile-vgpu

clean:
	rm -rf bin

compile-asaka-vgpu:
	GOARCH=amd64 GOOS=linux go build -o bin/asaka-vgpu src/asaka-vgpu/*.go

compile: compile-asaka-vgpu


