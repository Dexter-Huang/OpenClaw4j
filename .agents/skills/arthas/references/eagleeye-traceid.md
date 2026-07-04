# EagleEye TraceId 获取

适用场景：需要在不改代码的情况下，从线上请求线程中获取 EagleEye `traceId`，用于关联日志、链路系统或复现请求。

## 1. 前置检查

确认 EagleEye 类存在：

```text
sc -d com.taobao.eagleeye.EagleEye
```

如果找不到：

- 应用可能未集成 EagleEye。
- 类名可能被 relocate 或 shade。
- 需要用户提供实际依赖或类名。

## 2. 选择观察点

选择确定会被目标请求线程调用的方法：

- Controller 入口方法
- RPC Provider 方法
- Servlet Filter / Spring Interceptor
- 业务入口 service 方法

避免选择高频基础方法，否则输出量和开销都会失控。

## 3. 用 watch 打印 traceId

只打印 traceId：

```text
watch <类全名> <方法名> '@com.taobao.eagleeye.EagleEye@getTraceId()' -n 5
```

同时打印参数和 traceId：

```text
watch <类全名> <方法名> '{params, @com.taobao.eagleeye.EagleEye@getTraceId()}' -n 5 -x 2
```

说明：

- `@类名@静态方法()` 是 OGNL 静态方法调用语法。
- `-n 5` 必须保留或按需调小。
- `-x` 控制对象展开深度，默认不要过大。

## 4. 用 trace 关联耗时

```text
trace <类全名> <方法名> -n 5
```

如果环境集成了 EagleEye，`trace` 输出中可能直接带出 `trace_id=...`，同时可以看到调用链耗时。

## 5. 输出结论

建议包含：

```text
观察点：
使用命令：
traceId：
相关参数摘要：
下一步：
```

下一步通常是拿 traceId 去日志或链路系统查询，或把观察点下钻到 DAO/RPC 等更具体方法。
