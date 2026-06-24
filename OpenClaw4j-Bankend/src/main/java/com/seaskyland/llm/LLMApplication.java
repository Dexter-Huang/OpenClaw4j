package com.seaskyland.llm;

import com.seaskyland.llm.workflow.core.config.StudioProperties;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.properties.EnableConfigurationProperties;

import org.springframework.context.annotation.ComponentScan;
import org.springframework.context.annotation.EnableAspectJAutoProxy;
import org.springframework.context.annotation.FilterType;


/**
 * @author chuyaliang
 * @version 1.0
 * @description: 主程序
 * @date 2024/4/1 10:16
 */
//@SpringBootApplication(scanBasePackages = "com.seaskyland", exclude= {DataSourceAutoConfiguration.class})
@SpringBootApplication
@ComponentScan(basePackages = "com.seaskyland",
        excludeFilters = @ComponentScan.Filter(type = FilterType.REGEX,
                pattern = "com\\.seaskyland\\.llm\\.workflow\\.admin\\.generator\\..*"))
@EnableAspectJAutoProxy(proxyTargetClass = true)
@EnableConfigurationProperties(value = {StudioProperties.class})
public class LLMApplication {
    public static void main(String[] args) {
        SpringApplication.run(LLMApplication.class, args).registerShutdownHook();
    }
}
