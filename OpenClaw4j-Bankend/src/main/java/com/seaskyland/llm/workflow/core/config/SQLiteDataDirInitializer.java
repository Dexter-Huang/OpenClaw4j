package com.seaskyland.llm.workflow.core.config;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.BeansException;
import org.springframework.beans.factory.config.BeanFactoryPostProcessor;
import org.springframework.beans.factory.config.ConfigurableListableBeanFactory;
import org.springframework.context.EnvironmentAware;
import org.springframework.core.env.Environment;
import org.springframework.stereotype.Component;

import java.io.File;

/**
 * 在 DataSource Bean 初始化之前，自动创建 SQLite 数据库文件所在的父目录。
 * BeanFactoryPostProcessor 是 Spring 最早的扩展点之一，早于所有普通 Bean 的实例化。
 */
@Component
public class SQLiteDataDirInitializer implements BeanFactoryPostProcessor, EnvironmentAware {

    private static final Logger log = LoggerFactory.getLogger(SQLiteDataDirInitializer.class);
    private static final String SQLITE_PREFIX = "jdbc:sqlite:";

    private Environment environment;

    @Override
    public void setEnvironment(Environment environment) {
        this.environment = environment;
    }

    @Override
    public void postProcessBeanFactory(ConfigurableListableBeanFactory beanFactory) throws BeansException {
        String url = environment.getProperty("spring.datasource.url", "");
        if (!url.startsWith(SQLITE_PREFIX)) {
            return;
        }
        String filePath = url.substring(SQLITE_PREFIX.length());
        File dbFile = new File(filePath);
        File parentDir = dbFile.getParentFile();
        if (parentDir != null && !parentDir.exists()) {
            boolean created = parentDir.mkdirs();
            if (created) {
                log.info("[SQLite] Created database directory: {}", parentDir.getAbsolutePath());
            } else {
                log.warn("[SQLite] Failed to create database directory: {}", parentDir.getAbsolutePath());
            }
        }
    }
}
