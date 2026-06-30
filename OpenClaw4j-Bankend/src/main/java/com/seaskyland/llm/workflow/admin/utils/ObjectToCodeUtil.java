package com.seaskyland.llm.workflow.admin.utils;

import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;
import java.util.stream.Stream;

/**
 * 定义一些常用对象的构造方法代码
 *
 * @author vlsmb
 * @since 2025/9/5
 */
public final class ObjectToCodeUtil {

  private ObjectToCodeUtil() {}

  private static String mapToCode(Map<?, ?> map) {
    String elements =
        map.entrySet().stream()
            .flatMap(e -> Stream.of(e.getKey(), e.getValue()))
            .map(ObjectToCodeUtil::toCode)
            .collect(Collectors.joining(", "));
    return "Map.of(" + elements + ")";
  }

  private static String listToCode(List<?> list) {
    String elements = list.stream().map(ObjectToCodeUtil::toCode).collect(Collectors.joining(", "));
    return "List.of(" + elements + ")";
  }

  public static String toCode(Object object) {
    if (object == null) {
      return "null";
    } else if (object instanceof String) {
      return "\"" + object + "\"";
    } else if (object instanceof List<?>) {
      return listToCode((List<?>) object);
    } else if (object instanceof Map<?, ?>) {
      return mapToCode((Map<?, ?>) object);
    } else {
      // 默认使用 toString() 方法
      return object.toString();
    }
  }
}
