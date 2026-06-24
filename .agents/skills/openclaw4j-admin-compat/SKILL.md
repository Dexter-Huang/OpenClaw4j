---
name: openclaw4j-admin-compat
description: Use when changing OpenClaw4j legacy admin compatibility APIs, frontend legacy request handling, admin evaluation/prompt/tracing pages, or Result/AdminPage response contracts.
---

# OpenClaw4j Admin Compat

## 返回体契约

后端兼容 controller 应复用标准返回体：

```java
Result<T>
```

不要再创建 `AdminApiResponse` 这类第二套返回体。

legacy admin 分页可以保留一个小 DTO 来适配旧字段名：

```text
totalCount, totalPage, pageNumber, pageSize, pageItems
```

## 前端兼容点

旧前端页面经常判断 `response.code === 200`。兼容点放在 `packages/main/src/legacy/utils/request.ts`：当标准后端成功响应有 `data` 但没有 `code` 时，在这里规范化为 `code: 200`，不要逐个页面打补丁。

## 测试

后端契约覆盖在：

```text
OpenClaw4j-Bankend/src/test/java/com/seaskyland/llm/workflow/admin/compat/AdminCompatApiTest.java
```

契约变化使用 RED/GREEN：

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=AdminCompatApiTest' test
```

如果改了后端代码，最终完成前跑完整后端测试。

## 常见错误

- 把 `code/message/data` 重新做成后端第二套 wrapper。
- 直接返回 `PagingList` 给仍期待 `pageItems` 的旧页面。
- 在每个 React 页面里修兼容逻辑，而不是集中在 request 层规范化。
- 迭代阶段反复跑完整前后端构建，拖慢反馈。
