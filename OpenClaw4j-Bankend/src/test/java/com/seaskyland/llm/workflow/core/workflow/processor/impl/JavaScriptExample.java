package com.seaskyland.llm.workflow.core.workflow.processor.impl;

import java.util.HashMap;
import java.util.Map;

/** Java脚本执行示例 这个类展示了如何编写可以在工作流中执行的Java脚本 */
public class JavaScriptExample {

  /**
   * Java脚本的入口方法 注意：该方法必须是public static Map<String, Object> main(Map<String, Object> params)
   *
   * @param params 输入参数
   * @return 输出结果
   */
  public static Map<String, Object> main(Map<String, Object> params) {
    Map<String, Object> result = new HashMap<>();

    try {
      // 获取输入参数
      Map<String, Object> inputParams = (Map<String, Object>) params.get("params");

      // 示例：执行加法运算
      Object value1 = inputParams.get("value1");
      Object value2 = inputParams.get("value2");

      if (value1 instanceof Number && value2 instanceof Number) {
        double sum = ((Number) value1).doubleValue() + ((Number) value2).doubleValue();
        result.put("sum", sum);
        result.put("success", true);
        result.put("message", "计算完成");
      } else {
        result.put("success", false);
        result.put("message", "输入参数必须是数字类型");
      }

      // 可以添加更多业务逻辑
      result.put("processedAt", System.currentTimeMillis());

    } catch (Exception e) {
      result.put("success", false);
      result.put("message", "执行出错: " + e.getMessage());
    }

    return result;
  }
}
