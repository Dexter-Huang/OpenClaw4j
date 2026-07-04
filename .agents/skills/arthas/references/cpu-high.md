# CPU 飙高排查

适用场景：机器 CPU 飙高、Java 应用响应变慢、负载异常升高。

核心思路：先定位“哪个线程在忙”，再收敛到“哪个方法/代码路径在消耗 CPU”。

## 1. 查看 JVM 概况

```text
dashboard -n 1
```

关注：

- CPU 使用率
- 线程数量和状态
- GC 次数和耗时
- 堆内存变化

## 2. 定位热点线程

```text
thread -n 5
```

记录：

- 热点线程 ID
- 线程名
- CPU 占比
- 栈顶关键方法

判断方向：

- 栈在正则、JSON、加解密、压缩、日志格式化等位置：偏 CPU 密集计算。
- 大量 `BLOCKED`：偏锁竞争，继续看阻塞源。
- GC 线程占比高：结合 `dashboard`、`jvm` 和 GC 日志判断。

## 3. 收敛代码路径

当堆栈指向可疑类/方法后，再做有限观测：

```text
stack <class> <method> -n 5
trace <class> <method> -n 5
```

需要观察参数或返回值时：

```text
watch <class> <method> '{params, returnObj, throwExp}' -n 5 -x 2
```

原则：

- 类名和方法名尽量精确。
- 每次先限制 `-n`。
- 不要对高频基础方法做宽泛匹配。
- 先用 `stack` / `trace` 看路径，再决定是否 `watch` 参数。

## 4. 输出结论

至少包含：

```text
现象：
dashboard 摘要：
热点线程：
关键堆栈：
初步判断：
下一步建议：
```

如果还不能判断，给出下一条最小风险命令，例如更精确的 `trace` 目标，而不是扩大范围盲查。
