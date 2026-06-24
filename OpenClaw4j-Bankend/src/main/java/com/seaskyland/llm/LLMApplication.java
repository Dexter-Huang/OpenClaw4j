package com.seaskyland.llm;
// SAA include
import com.seaskyland.llm.workflow.admin.generator.GeneratorApplication;
import com.seaskyland.llm.workflow.admin.generator.config.GraphProjectGenerationConfiguration;
import com.seaskyland.llm.workflow.core.config.StudioProperties;
import org.mybatis.spring.annotation.MapperScan;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration;
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
@SpringBootApplication(scanBasePackages = "com.seaskyland")
@ComponentScan(basePackages = { "com.seaskyland" }
        // SAA include
        ,excludeFilters = {
                @ComponentScan.Filter(type = FilterType.ASSIGNABLE_TYPE,
                        classes = GraphProjectGenerationConfiguration.class),
                @ComponentScan.Filter(type = FilterType.ASSIGNABLE_TYPE, classes = GeneratorApplication.class),
                @ComponentScan.Filter(type = FilterType.ASSIGNABLE_TYPE,
                        classes = GeneratorApplication.MockLoginController.class) }
)
@EnableAspectJAutoProxy(proxyTargetClass = true)
@MapperScan(value = {"com.seaskyland.llm.workflow.core.base.mapper"})
// SAA include
@EnableConfigurationProperties(value = {StudioProperties.class})
public class LLMApplication {
    public static void main(String[] args) {
        SpringApplication.run(LLMApplication.class, args).registerShutdownHook();
    }
}