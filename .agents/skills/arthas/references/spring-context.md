# Spring Context / Bean 排查

适用场景：排查 Spring `ApplicationContext`、Bean 注册、条件装配、配置注入、类型冲突等问题。

原则：

- 先做只读查询，不要直接 `getBean()` 触发初始化。
- 使用 `-l` 限制 `vmtool` 实例数量。
- 输出 Bean 列表时先按关键词过滤，再限制结果数量。
- 类找不到时优先怀疑 ClassLoader 不对。

## 1. 获取 ApplicationContext

优先查 `AbstractApplicationContext`：

```text
vmtool --action getInstances --className org.springframework.context.support.AbstractApplicationContext -l 5
```

如果没有结果，再尝试接口：

```text
vmtool --action getInstances --className org.springframework.context.ApplicationContext -l 5
```

多个 context 时，根据 ClassLoader 和应用特征挑选：

- Spring Boot 应用常见 `LaunchedURLClassLoader`。
- 排除明显不是业务应用的模块 ClassLoader。

## 2. 查询配置值和来源

只看配置值：

```text
vmtool --action getInstances --className org.springframework.context.support.AbstractApplicationContext -l 1 --express 'instances[0].getEnvironment().getProperty("server.port")'
```

查配置来源：

```text
vmtool --action getInstances --className org.springframework.context.support.AbstractApplicationContext -l 1 --express '#env=instances[0].getEnvironment(), #ps=#env.getPropertySources().get("configurationProperties"), #ps.findConfigurationProperty("server.port")'
```

如果应用集成 Actuator，可以尝试 `EnvironmentEndpoint`：

```text
vmtool --action getInstances --className org.springframework.boot.actuate.env.EnvironmentEndpoint -l 1 --express 'instances[0].environmentEntry("server.port")'
```

## 3. 按 Bean Name 验证存在性

以 `fooService` 为例：

```text
vmtool --action getInstances --className org.springframework.context.support.AbstractApplicationContext -l 1 --express 'instances[0].containsBean("fooService")'
vmtool --action getInstances --className org.springframework.context.support.AbstractApplicationContext -l 1 --express 'instances[0].containsLocalBean("fooService")'
vmtool --action getInstances --className org.springframework.context.support.AbstractApplicationContext -l 1 --express 'instances[0].containsBeanDefinition("fooService")'
vmtool --action getInstances --className org.springframework.context.support.AbstractApplicationContext -l 1 --express 'instances[0].getAliases("fooService")'
```

判读：

- `containsBean=true` 但 `containsLocalBean=false`：可能来自父 context。
- `containsBean=false` 但按预期应存在：优先检查 context 是否选错、`@Profile`、`@Conditional`、配置是否生效。

## 4. 按关键词搜索 Bean

只知道关键词时，例如 `order`：

```text
vmtool --action getInstances --className org.springframework.context.support.AbstractApplicationContext -l 1 --express '#ctx=instances[0], #names=@java.util.Arrays@asList(#ctx.getBeanDefinitionNames()), #m=#names.{? #this.toLowerCase().contains("order")}, #m.subList(0, @java.lang.Math@min(#m.size(), 50))'
```

拿到候选后，再按 Bean Name 验证。

## 5. 按类型查找 Bean

已知接口或父类时：

```text
vmtool --action getInstances --className org.springframework.context.support.AbstractApplicationContext -l 1 --express 'instances[0].getBeanNamesForType(@com.foo.OrderService@class)'
```

返回多个候选时：

```text
vmtool --action getInstances --className org.springframework.context.support.AbstractApplicationContext -l 1 --express 'instances[0].getBeansOfType(@com.foo.OrderService@class).keySet()'
```

提示：

- JDK Proxy / CGLIB 场景优先按接口类型查。
- `@com.foo.OrderService@class` 报 `ClassNotFound` 时，先用 `classloader` 找应用 ClassLoader，再给 `vmtool` / `ognl` 增加 `--classLoader <hash>` 或 `--classLoaderClass <className>`。

## 6. 查看 BeanDefinition

确认 Bean 注册来源、scope、工厂方法：

```text
vmtool --action getInstances --className org.springframework.context.support.AbstractApplicationContext -l 1 --express '#ctx=instances[0], #bf=#ctx.getBeanFactory(), #bd=#bf.getBeanDefinition("fooService")'
```

不要为了看定义直接 `getBean()`，除非用户已接受可能触发初始化或副作用。
