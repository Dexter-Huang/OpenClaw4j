# ScopedValue Auth Context Migration

## Why Change

The backend used `ThreadLocal<RequestContext>` in `RequestContextHolder`. Authentication
interceptors populated that holder before controller execution, while async helpers copied
and cleared the same value manually.

That model has two drawbacks:

- Servlet container threads are reused, so every successful request needs a guaranteed
  cleanup path.
- Async code must remember to copy and remove the context around every task.

`ScopedValue` gives the request context a lexical lifetime. The value exists only while
the bound operation is running, and it disappears automatically when that operation
returns or throws.

## Why Filters Instead Of Interceptors

`ScopedValue` must wrap downstream execution:

```java
ScopedValue.where(REQUEST_CONTEXT, context).run(() -> chain.doFilter(request, response));
```

`HandlerInterceptor.preHandle()` cannot wrap the controller invocation. It only runs
before the handler and returns `true` or `false`. A servlet filter can surround
`FilterChain.doFilter`, so it is the correct boundary for binding authentication state.

## Runtime Shape

- `RequestContextHolder` owns one `ScopedValue<RequestContext>`.
- `getRequestContext()` remains the compatibility read API for controllers and services.
- Authentication filters build `RequestContext` after token/API-key validation.
- The filters call `RequestContextHolder.runWithRequestContext(...)` around
  `filterChain.doFilter(...)`.
- `RequestContextThreadPoolWrapper` captures the current context at submission time and
  rebinds it while the task runs.

## Build Requirement

The backend now targets JDK 26:

- `pom.xml` keeps `<java.version>26</java.version>`.
- Maven compiler plugin entries use `<release>${java.version}</release>`.
- Local verification should set `JAVA_HOME=D:\jdk-26` and put `D:\jdk-26\bin` first in
  `PATH`.

## Compatibility Notes

Call sites that only read the context through `RequestContextHolder.getRequestContext()`
do not need changes. Code that previously called `setRequestContext` or
`clearRequestContext` must use `runWithRequestContext` or `callWithRequestContext`.
