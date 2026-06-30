FROM mcr.microsoft.com/devcontainers/base:alpine
RUN apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community --repository=https://dl-cdn.alpinelinux.org/alpine/edge/testing go
RUN GOBIN=/usr/local/bin go install -v github.com/go-delve/delve/cmd/dlv@latest
RUN GOBIN=/usr/local/bin go install -v	golang.org/x/tools/gopls@latest
RUN GOBIN=/usr/local/bin go install -v github.com/golang/vscode-go/vscgo@latest
RUN GOBIN=/usr/local/bin go install -v github.com/haya14busa/goplay/cmd/goplay@latest
RUN GOBIN=/usr/local/bin go install -v github.com/fatih/gomodifytags@latest
RUN GOBIN=/usr/local/bin go install -v github.com/josharian/impl@latest
RUN GOBIN=/usr/local/bin go install -v github.com/cweill/gotests/gotests@latest
RUN GOBIN=/usr/local/bin go install -v github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
