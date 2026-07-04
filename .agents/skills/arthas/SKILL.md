---
name: arthas
description: Use when diagnosing Java, JVM, Spring, or production runtime issues with Alibaba Arthas, especially attaching from D:\arthas-bin on Windows, high CPU, hot threads, stack, trace, watch, tt, vmtool, EagleEye traceId, Spring ApplicationContext, Bean, configuration, or Arthas MCP endpoint tasks.
---

# Arthas

使用 Alibaba Arthas 诊断 Java 应用时使用本 skill。本项目本机离线 Arthas 位于 `D:\arthas-bin`，版本经 `arthas-boot.jar --help` 确认为 `4.3.1`。

## 基本原则

- 先用只读、低风险命令收集证据，再考虑 `watch`、`trace`、`tt`、`jad`、`mc`、`redefine` 等更有影响的命令。
- `watch`、`trace`、`stack`、`tt` 等观测命令必须设置 `-n` 或明确的限制条件。
- 优先精确到类名和方法名，避免对高频大范围表达式做无界观测。
- 输出结论时包含关键证据摘要、初步判断和下一步建议。
- 不要在用户未确认时对生产进程执行会改变运行时行为的命令。

## 本机启动

先列出 Java 进程：

```powershell
$env:JAVA_HOME='D:\jdk-26'
$env:Path='D:\jdk-26\bin;' + $env:Path
& 'D:\jdk-26\bin\jps.exe' -l
```

推荐用 `arthas-boot.jar` attach 指定 PID：

```powershell
$env:JAVA_HOME='D:\jdk-26'
$env:Path='D:\jdk-26\bin;' + $env:Path
& 'D:\jdk-26\bin\java.exe' -jar 'D:\arthas-bin\arthas-boot.jar' <pid>
```

Windows 批处理入口也可用：

```powershell
& 'D:\arthas-bin\as.bat' <pid> --ignore-tools
```

`as.bat` 默认 telnet 端口 `3658`、HTTP 端口 `8563`，并从 `JAVA_HOME` 查找 `java.exe`。JDK 9+ 没有 `tools.jar` 时加 `--ignore-tools`。

详细本机路径、端口、MCP 和批处理方式见 `references/local-arthas-bin.md`。

## 场景索引

读取对应 reference 后再给出具体命令：

- CPU 飙高、线程热点、堆栈定位：`references/cpu-high.md`
- EagleEye traceId 获取：`references/eagleeye-traceid.md`
- Spring `ApplicationContext`、Bean、配置注入排查：`references/spring-context.md`
- 本机 `D:\arthas-bin`、端口、MCP、本地 attach：`references/local-arthas-bin.md`

## 常用只读命令

```text
dashboard -n 1
thread -n 5
jvm
sysprop
sysenv
sc -d <class-pattern>
sm -d <class-pattern> <method-pattern>
classloader -l
```

## 常用有限观测命令

```text
stack <class> <method> -n 5
trace <class> <method> -n 5
watch <class> <method> '{params, returnObj, throwExp}' -n 5 -x 2
tt -t <class> <method> -n 5
```

## 输出格式

诊断回复按这个结构组织：

```text
现象：
证据：
初步判断：
下一步：
风险/注意事项：
```

如果证据不足，先说明还需要哪个 Arthas 命令输出，而不是猜结论。
