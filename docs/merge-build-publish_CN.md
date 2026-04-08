# 合并、打包、推送操作手册

适用仓库：`E:\github\CLIProxyAPI`

本手册记录一次已经实际走通的流程，用于下次继续执行：

- 检查上游 `origin/main` 是否更新
- 合并到当前工作分支
- 处理已知冲突
- 用 Docker 本地打包
- 推送代码分支
- 推送 Docker 镜像

本次实际结果：

- merge commit：`e3a0bcee`
- 对齐源库版本：`v6.9.16`
- 已推送分支：`fork/fix-openai-compat-endpoint-routing`
- 本地镜像：`cliproxyapi:v6.9.16`
- 目标镜像仓库：`eceasy/cli-proxy-api`

## 1. 检查远端和当前分支

```powershell
git -C E:\github\CLIProxyAPI remote -v
git -C E:\github\CLIProxyAPI status --short --branch
git -C E:\github\CLIProxyAPI branch -vv
```

## 2. 拉取上游并判断是否要合并

```powershell
git -C E:\github\CLIProxyAPI fetch origin --prune --tags
git -C E:\github\CLIProxyAPI rev-list --left-right --count HEAD...origin/main
git -C E:\github\CLIProxyAPI log --oneline --decorate --left-right HEAD...origin/main -n 30
```

如果右侧有新增提交，就需要 merge。

## 3. 如果有 `.git/index.lock`，先清理

```powershell
Test-Path E:\github\CLIProxyAPI\.git\index.lock
Remove-Item -LiteralPath E:\github\CLIProxyAPI\.git\index.lock -Force
```

只在确认是残留锁文件时删除。

## 4. 合并上游

```powershell
git -C E:\github\CLIProxyAPI merge --no-edit origin/main
```

本次实际冲突文件：

- `internal/translator/openai/openai/responses/openai_openai-responses_response.go`
- `internal/translator/openai/openai/responses/openai_openai-responses_response_test.go`

## 5. 已知冲突处理方式

这次 `Responses` 冲突可直接采用 merge 对侧版本：

```powershell
git -C E:\github\CLIProxyAPI checkout --theirs -- internal/translator/openai/openai/responses/openai_openai-responses_response.go internal/translator/openai/openai/responses/openai_openai-responses_response_test.go
git -C E:\github\CLIProxyAPI add -- internal/translator/openai/openai/responses/openai_openai-responses_response.go internal/translator/openai/openai/responses/openai_openai-responses_response_test.go
git -C E:\github\CLIProxyAPI diff --check
```

## 6. 完成 merge commit 并推送代码

```powershell
git -C E:\github\CLIProxyAPI commit --no-edit
git -C E:\github\CLIProxyAPI push fork fix-openai-compat-endpoint-routing
```

## 7. 不要直接用本机 Go 打包

先检查：

```powershell
go version
Get-Content E:\github\CLIProxyAPI\go.mod -TotalCount 10
```

本次实际情况：

- 本机 Go：`go1.20.6`
- 仓库要求：`go 1.26.0`

所以要走 Docker 构建，不要直接 `go build`。

## 8. 启动 Docker Desktop

```powershell
$env:Path += ';C:\Program Files\Docker\Docker\resources\bin'
docker desktop start
docker info
```

## 9. 先检查代理环境变量

这是本次最关键的坑。先看当前代理：

```powershell
Get-ChildItem Env: | Where-Object { $_.Name -match 'GO|PROXY|HTTP|HTTPS|NO_PROXY' } | Sort-Object Name
```

本次实际发现以下错误代理：

```text
HTTP_PROXY=http://127.0.0.1:9
HTTPS_PROXY=http://127.0.0.1:9
ALL_PROXY=http://127.0.0.1:9
GIT_HTTP_PROXY=http://127.0.0.1:9
GIT_HTTPS_PROXY=http://127.0.0.1:9
```

这会导致：

- `go mod download` 失败
- Docker Hub OAuth 取 token 时报 `EOF`

在当前会话里先清掉这些坏代理：

```powershell
Remove-Item Env:HTTP_PROXY,Env:HTTPS_PROXY,Env:ALL_PROXY,Env:GIT_HTTP_PROXY,Env:GIT_HTTPS_PROXY -ErrorAction SilentlyContinue
```

## 10. 用 Docker 本地打包

推荐命令：

```powershell
Remove-Item Env:HTTP_PROXY,Env:HTTPS_PROXY,Env:ALL_PROXY,Env:GIT_HTTP_PROXY,Env:GIT_HTTPS_PROXY -ErrorAction SilentlyContinue
$env:Path += ';C:\Program Files\Docker\Docker\resources\bin'
docker build -t cliproxyapi:merge-e3a0bcee --build-arg VERSION=merge-e3a0bcee --build-arg COMMIT=e3a0bcee --build-arg BUILD_DATE=2026-04-07T00:00:00Z --build-arg GOPROXY=https://goproxy.cn,direct --build-arg GOSUMDB=sum.golang.google.cn .
```

如果版本号要和源库保持一致，建议直接把 `VERSION` 设为源库 tag：`v6.9.16`。

```powershell
docker build -t cliproxyapi:v6.9.16 --build-arg VERSION=v6.9.16 --build-arg COMMIT=e3a0bcee --build-arg BUILD_DATE=2026-04-07T00:00:00Z --build-arg GOPROXY=https://goproxy.cn,direct --build-arg GOSUMDB=sum.golang.google.cn .
```

本次实际已成功构建镜像；后续建议统一使用版本标签：`cliproxyapi:v6.9.16`。

## 11. 推送 Docker 镜像

仓库工作流默认目标：`eceasy/cli-proxy-api`

```powershell
docker tag cliproxyapi:v6.9.16 eceasy/cli-proxy-api:v6.9.16
Remove-Item Env:HTTP_PROXY,Env:HTTPS_PROXY,Env:ALL_PROXY,Env:GIT_HTTP_PROXY,Env:GIT_HTTPS_PROXY -ErrorAction SilentlyContinue
$env:Path += ';C:\Program Files\Docker\Docker\resources\bin'
docker push eceasy/cli-proxy-api:v6.9.16
```

## 12. 本次镜像推送失败原因

本次推送失败经历了两层问题：

1. 先是网络层错误：`failed to fetch oauth token: Post "https://auth.docker.io/token": EOF`
2. 清掉坏代理后，变成权限错误：`insufficient_scope: authorization failed`

这说明当前 Docker 登录身份没有 `eceasy/cli-proxy-api` 的推送权限。

## 13. 下次如果还要推镜像，必须先满足的条件

- Docker Desktop 当前登录账号拥有 `eceasy/cli-proxy-api` 的 push 权限
- 或者改推到自己有权限的仓库
- 或者改用 GitHub Actions tag 流程自动推送

## 14. 下次最小复用顺序

1. 检查远端和当前分支
2. `fetch origin --prune --tags`
3. 如有残留锁文件，删 `.git/index.lock`
4. `merge origin/main`
5. 如仍是那两个 `Responses` 文件冲突，直接 `checkout --theirs`
6. `add` 后 `commit --no-edit`
7. `push fork 当前分支`
8. 清理坏代理变量
9. 启动 Docker Desktop
10. 用 Docker 构建本地镜像
11. 确认镜像仓库权限后再 `docker push`
