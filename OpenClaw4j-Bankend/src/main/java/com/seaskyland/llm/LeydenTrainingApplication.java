package com.seaskyland.llm;

import java.util.List;

public final class LeydenTrainingApplication {

  private LeydenTrainingApplication() {}

  public static void main(String[] args) throws ClassNotFoundException {
    for (String className : trainingClasses()) {
      Class.forName(className);
    }
  }

  private static List<String> trainingClasses() {
    return List.of(
        "com.seaskyland.llm.LLMApplication",
        "org.springframework.boot.SpringApplication",
        "org.springframework.context.annotation.AnnotationConfigApplicationContext",
        "org.springframework.core.io.support.SpringFactoriesLoader",
        "org.springframework.core.type.classreading.SimpleMetadataReaderFactory",
        "org.springframework.beans.factory.support.DefaultListableBeanFactory",
        "org.springframework.web.servlet.DispatcherServlet",
        "org.springframework.boot.web.server.servlet.context.AnnotationConfigServletWebServerApplicationContext",
        "com.baomidou.mybatisplus.core.MybatisConfiguration",
        "org.apache.ibatis.session.SqlSessionFactory",
        "com.mysql.cj.jdbc.Driver",
        "com.fasterxml.jackson.databind.ObjectMapper",
        "com.seaskyland.llm.workflow.runtime.domain.Result",
        "com.seaskyland.llm.workflow.core.base.mq.jvm.JvmMessageBus",
        "com.seaskyland.llm.workflow.core.config.MybatisPlusConfig");
  }
}
