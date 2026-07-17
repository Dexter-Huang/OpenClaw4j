# OpenClaw4j Sandbox

`OpenClaw4j-Sandbox` 是工作流 `Script` 节点的独立代码执行服务。服务使用 Go + Hertz 暴露 HTTP API，在 Linux 容器中通过 sandlock Go SDK 执行 Python 和 JavaScript 脚本。

## 构建镜像

```powershell
docker build -t openclaw4j-sandbox:local OpenClaw4j-Sandbox
```

镜像构建使用 Docker BuildKit cache mount 缓存 Cargo、Go、pip、pnpm 和 sandlock 编译目录。后续重复构建时会复用本机 Docker builder 的缓存，不需要把这些构建产物挂成运行时 volume。

## 启动服务

```powershell
docker run --rm -p 127.0.0.1:9010:9010 openclaw4j-sandbox:local
```

或使用中间件 compose profile：

```powershell
docker compose -f deploy/docker-compose.middleware.yml --profile sandbox up -d openclaw4j-sandbox
```

## 调用示例

```powershell
Invoke-RestMethod -Uri 'http://127.0.0.1:9010/health'
```

```powershell
$body = @{
  language = 'python'
  code = "def main(params):`n    return {'sum': params['a'] + params['b']}"
  params = @{ a = 1; b = 2 }
} | ConvertTo-Json -Depth 10

Invoke-RestMethod -Uri 'http://127.0.0.1:9010/v1/execute' -Method Post -ContentType 'application/json' -Body $body
```

## 内置依赖

沙箱镜像默认内置一组白名单依赖：

- Python: `numpy`, `pandas`, `scipy`
- JavaScript: `lodash`, `dayjs`, `decimal.js`, `uuid`

Python 依赖安装在 `/opt/openclaw4j-sandbox/deps/python` venv 中，JavaScript 依赖安装在 `/opt/openclaw4j-sandbox/deps/node/node_modules`。运行脚本时 sandlock SDK 只给 `/opt/openclaw4j-sandbox/deps` 只读访问权限，并通过下面环境变量暴露运行时：

```text
OPENCLAW_SANDBOX_DEPS_DIR=/opt/openclaw4j-sandbox/deps
OPENCLAW_SANDBOX_PYTHON_RUNTIME=/opt/openclaw4j-sandbox/deps/python/bin/python
OPENCLAW_SANDBOX_NODE_PATH=/opt/openclaw4j-sandbox/deps/node/node_modules
```

新增依赖时优先修改：

- `deps/python/requirements.txt`
- `deps/node/package.json`，并重新生成 `deps/node/pnpm-lock.yaml`

不建议默认把依赖目录挂成 volume。空 volume 会遮蔽镜像内置依赖，导致线上和本地行为不一致。

## 本地测试

```powershell
cd OpenClaw4j-Sandbox
go test ./...
```

Windows 和其他非 Linux 环境会使用直接执行 fallback，仅用于本地单元测试和基础调试，不提供安全沙箱边界。安全行为以 Linux 容器中的 sandlock SDK 验证为准。

## 日志

服务默认输出 `info` 级别日志到 stdout，Docker 下可以直接查看：

```powershell
docker logs -f openclaw4j-sandbox
```

执行日志会记录 `request_id`、脚本语言、超时时间、资源限制、工作目录、执行耗时、退出码以及 stdout/stderr 字节数。脚本输出内容不会默认写入日志，避免大输出或敏感数据污染容器日志。
