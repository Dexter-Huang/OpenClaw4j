package com.seaskyland.llm.workflow.core.utils;

import static org.assertj.core.api.Assertions.assertThat;

import java.lang.reflect.Method;
import org.junit.jupiter.api.Test;

class LogUtilsTest {

  @Test
  void buildFormatsSlf4jStylePlaceholdersAndKeepsThrowable() throws Exception {
    RuntimeException error = new RuntimeException("send failed");

    LogUtils.LogContent logContent = build("send document mq message, messageId: {}", "1", error);

    assertThat(logContent.getLogData()).contains("send document mq message, messageId: 1");
    assertThat(logContent.getLogData()).doesNotContain("messageId: {}");
    assertThat(logContent.getThrowable()).isSameAs(error);
  }

  private LogUtils.LogContent build(Object... objects) throws Exception {
    Method build = LogUtils.class.getDeclaredMethod("build", Object[].class);
    build.setAccessible(true);
    return (LogUtils.LogContent) build.invoke(null, new Object[] {objects});
  }
}
