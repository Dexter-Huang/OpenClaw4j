# 本机 Arthas 离线包

本机 Arthas 位于：

```text
D:\arthas-bin
```

已确认关键文件：

```text
arthas-boot.jar
arthas-core.jar
arthas-agent.jar
arthas-client.jar
as.bat
as-service.bat
arthas.properties
async-profiler/
lib/
```

`arthas-boot.jar --help` 显示本机 Arthas 版本为 `4.3.1`。Arthas 4 支持 JDK 8+；诊断 JDK 6/7 应改用 Arthas 3。

## 环境准备

本项目默认 JDK：

```powershell
$env:JAVA_HOME='D:\jdk-26'
$env:Path='D:\jdk-26\bin;' + $env:Path
```

列出 Java 进程：

```powershell
& 'D:\jdk-26\bin\jps.exe' -l
```

## Attach 方式

交互 attach 指定 PID：

```powershell
& 'D:\jdk-26\bin\java.exe' -jar 'D:\arthas-bin\arthas-boot.jar' <pid>
```

只 attach，不自动连接客户端：

```powershell
& 'D:\jdk-26\bin\java.exe' -jar 'D:\arthas-bin\arthas-boot.jar' --attach-only <pid>
```

执行一次性命令：

```powershell
& 'D:\jdk-26\bin\java.exe' -jar 'D:\arthas-bin\arthas-boot.jar' -c 'dashboard -n 1; thread -n 5' <pid>
```

按进程名选择：

```powershell
& 'D:\jdk-26\bin\java.exe' -jar 'D:\arthas-bin\arthas-boot.jar' --select LLMApplication
```

Windows 批处理入口：

```powershell
& 'D:\arthas-bin\as.bat' <pid> --ignore-tools
```

`as.bat` 要求第一个参数是 PID。它默认：

```text
telnet port: 3658
http port: 8563
target ip: 127.0.0.1
```

JDK 9+ 没有 `tools.jar`，使用 `as.bat` 时加 `--ignore-tools`。`arthas-boot.jar` 路径通常更直接，优先使用它。

## 端口与配置

`D:\arthas-bin\arthas.properties` 当前包含：

```properties
arthas.telnetPort=3658
arthas.httpPort=8563
arthas.ip=127.0.0.1
arthas.sessionTimeout=10800
arthas.localConnectionNonAuth=true
arthas.mcpEndpoint=/mcp
arthas.mcpProtocol=STREAMABLE
```

HTTP 控制台默认：

```text
http://127.0.0.1:8563
```

MCP endpoint 默认：

```text
http://127.0.0.1:8563/mcp
```

如果端口冲突：

```powershell
& 'D:\jdk-26\bin\java.exe' -jar 'D:\arthas-bin\arthas-boot.jar' --telnet-port 3668 --http-port 8573 <pid>
```

## 停止与清理

在 Arthas 会话内优先使用：

```text
stop
```

`stop` 会关闭 Arthas server 并从目标 JVM 卸载。只退出客户端时使用 `quit` 或 `exit`，不要误以为它会停止 Arthas server。

## Windows 调用注意

- PowerShell 中路径用单引号和 `&` 调用。
- 不要把整条 `java -jar ...` 拼成字符串再 `Invoke-Expression`。
- 执行批量命令时优先使用 `-c` 或 `-f`，不要套 `cmd.exe /c`。
- 如果要连接正在运行的 Arthas server，优先访问 HTTP 控制台；本机不一定安装 telnet。
