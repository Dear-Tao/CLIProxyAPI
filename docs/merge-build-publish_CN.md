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
- 最终推送镜像：`liming233/cliproxyapi:latest`
- 最终镜像 digest：`sha256:153e5bfa7af3b4183ad5c205d39da18bfe0fb00061a6f497746f82cb2737fad3`

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

## 11. 先确认当前 latest 是否还是旧镜像

在覆盖推送前，先直接运行当前 `latest` 镜像检查版本：

```powershell
$env:Path += ';C:\Program Files\Docker\Docker\resources\bin'
docker run --rm --entrypoint /bin/sh liming233/cliproxyapi:latest -c "./CLIProxyAPI"
```

本次实际检查结果是旧镜像：

```text
CLIProxyAPI Version: v6.9.7, Commit: ddbeb2a4, BuiltAt: 2026-03-31T03:52:28Z
```

这一步非常重要，可以避免把旧镜像误认成最新版本。

## 12. 如果基础镜像丢失，先补拉 builder 基础镜像

本次在重新构建时发现本地缺少：

- `golang:1.26-alpine`
- `alpine:3.22.0`

先检查：

```powershell
$env:Path += ';C:\Program Files\Docker\Docker\resources\bin'
docker image inspect golang:1.26-alpine
docker image inspect alpine:3.22.0
```

如果不存在，先补拉：

```powershell
Remove-Item Env:HTTP_PROXY,Env:HTTPS_PROXY,Env:ALL_PROXY,Env:GIT_HTTP_PROXY,Env:GIT_HTTPS_PROXY -ErrorAction SilentlyContinue
$env:Path += ';C:\Program Files\Docker\Docker\resources\bin'
docker pull golang:1.26-alpine
docker pull alpine:3.22.0
```

## 13. 重新构建最新版镜像

这次最终成功的重建命令是：

```powershell
Remove-Item Env:HTTP_PROXY,Env:HTTPS_PROXY,Env:ALL_PROXY,Env:GIT_HTTP_PROXY,Env:GIT_HTTPS_PROXY -ErrorAction SilentlyContinue
$env:Path += ';C:\Program Files\Docker\Docker\resources\bin'
docker build -t cliproxyapi:v6.9.16 --build-arg VERSION=v6.9.16 --build-arg COMMIT=f3d3e8f1 --build-arg BUILD_DATE=2026-04-08T00:00:00Z --build-arg GOPROXY=https://goproxy.cn,direct --build-arg GOSUMDB=sum.golang.google.cn .
```

## 14. 推送前验证新镜像版本号

推送前必须直接跑一次新镜像，确认它确实是最新版本：

```powershell
$env:Path += ';C:\Program Files\Docker\Docker\resources\bin'
docker run --rm --entrypoint /bin/sh cliproxyapi:v6.9.16 -c "./CLIProxyAPI"
```

本次实际输出：

```text
CLIProxyAPI Version: v6.9.16, Commit: f3d3e8f1, BuiltAt: 2026-04-08T00:00:00Z
```

虽然随后会因为没有 `config.yaml` 报错退出，但只要版本横幅正确，这个镜像就是最新包。

## 15. 推送 Docker 镜像

如果只推自己的仓库，并统一覆盖 `latest`，用这组命令：

```powershell
Remove-Item Env:HTTP_PROXY,Env:HTTPS_PROXY,Env:ALL_PROXY,Env:GIT_HTTP_PROXY,Env:GIT_HTTPS_PROXY -ErrorAction SilentlyContinue
$env:Path += ';C:\Program Files\Docker\Docker\resources\bin'
docker tag cliproxyapi:v6.9.16 liming233/cliproxyapi:latest
docker push liming233/cliproxyapi:latest
```

本次实际推送成功结果：

```text
latest: digest: sha256:153e5bfa7af3b4183ad5c205d39da18bfe0fb00061a6f497746f82cb2737fad3 size: 856
```

## 16. 为什么之前会误以为已经推上最新版

本次踩过一个真实坑：

- 旧的 `liming233/cliproxyapi:latest` 本地就已经存在
- 第一次判断时只看到了 push 输出和本地 tag，没有先跑容器看版本
- 后来实际运行 `latest` 才发现它还是 `v6.9.7`

下次必须遵守一个顺序：

1. 先运行当前 `latest` 看是否还是旧版本
2. 重新构建
3. 再运行新镜像确认版本号
4. 最后再 push

## 17. 下次最小复用顺序

1. 检查远端和当前分支
2. `fetch origin --prune --tags`
3. 如有残留锁文件，删 `.git/index.lock`
4. `merge origin/main`
5. 如仍是那两个 `Responses` 文件冲突，直接 `checkout --theirs`
6. `add` 后 `commit --no-edit`
7. `push fork 当前分支`
8. 清理坏代理变量
9. 启动 Docker Desktop
10. 先运行当前 `liming233/cliproxyapi:latest` 检查是不是旧镜像
11. 如本地缺少 builder 基础镜像，先 `docker pull golang:1.26-alpine` 和 `docker pull alpine:3.22.0`
12. 重新构建 `cliproxyapi:v6.9.16`
13. 运行新镜像确认版本号是 `v6.9.16` 和最新 commit
14. `docker tag cliproxyapi:v6.9.16 liming233/cliproxyapi:latest`
15. `docker push liming233/cliproxyapi:latest`

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
