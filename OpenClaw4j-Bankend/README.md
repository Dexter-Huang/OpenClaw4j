# cbes-llm

langfuse 配置
```python
import os
import base64
import urllib.parse

# 从环境变量获取凭证
LANGFUSE_PUBLIC_KEY = 'pk-lf-b8ddc404-72e0-4734-a8aa-aa172cda43d9'
LANGFUSE_SECRET_KEY = 'sk-lf-a058cb10-a3ea-42a8-9eee-276e9cca6956'

# 对凭证进行Base64编码
LANGFUSE_AUTH = base64.b64encode(
    f'{LANGFUSE_PUBLIC_KEY}:{LANGFUSE_SECRET_KEY}'.encode()
).decode()

# 构造认证头
auth_header = f'Basic {LANGFUSE_AUTH}'
print(auth_header)
```

如何关闭otel
```yaml
# 如果要关闭检测功能
management:
  tracing:
    enabled: false
  observations:
    annotations:
      enabled: false

otel:
  traces:
    exporter: none
  metrics:
    exporter: none
  logs:
    exporter: none
```

spel血泪教训：
```java
// 下面是正确的
List<String> docIds = List.of("1");
var b = new FilterExpressionBuilder();
var exp = b
        .and(b.eq(RagConstants.KEY_WORKSPACE_ID, "1"),  // 确保使用字符串类型
                b.in(RagConstants.KEY_DOC_ID, docIds.toArray()))
        .build();
// 下面会被错误解析成二位数组
List<String> docIds = List.of("1");
var b = new FilterExpressionBuilder();
var exp = b
        .and(b.eq(RagConstants.KEY_WORKSPACE_ID, "1"),  // 确保使用字符串类型
                b.in(RagConstants.KEY_DOC_ID, docIds))
        .build();
```
