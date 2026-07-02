/*
 * Copyright 2025 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.seaskyland.llm.workflow.core.config;

import com.baomidou.mybatisplus.annotation.DbType;
import com.baomidou.mybatisplus.extension.plugins.MybatisPlusInterceptor;
import com.baomidou.mybatisplus.extension.plugins.inner.PaginationInnerInterceptor;
import com.baomidou.mybatisplus.extension.spring.MybatisSqlSessionFactoryBean;
import org.apache.commons.lang3.StringUtils;
import org.apache.ibatis.session.SqlSessionFactory;
import org.springframework.beans.factory.annotation.Value;
import org.mybatis.spring.annotation.MapperScan;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.ImportRuntimeHints;

import javax.sql.DataSource;
import java.util.Locale;

/**
 * Configuration class for MyBatis-Plus integration. Provides pagination support and
 * mapper scanning configuration.
 *
 * @since 1.0.0.3
 */
@Configuration
@ImportRuntimeHints(MybatisPlusRuntimeHints.class)
@MapperScan(basePackages = "com.seaskyland.llm.workflow.core.base.mapper",
		sqlSessionFactoryRef = "sqlSessionFactory")
public class MybatisPlusConfig {

	/**
	 * Configures MyBatis-Plus interceptor with the active database dialect.
	 * @return MybatisPlusInterceptor instance
	 */
	@Bean
	public MybatisPlusInterceptor mybatisPlusInterceptor(
			@Value("${spring.datasource.url:}") String datasourceUrl,
			@Value("${mybatis-plus.global-config.db-config.db-type:sqlite}") String configuredDbType) {
		MybatisPlusInterceptor interceptor = new MybatisPlusInterceptor();
		interceptor.addInnerInterceptor(new PaginationInnerInterceptor(resolveDbType(datasourceUrl, configuredDbType)));
		return interceptor;
	}

	@Bean
	public SqlSessionFactory sqlSessionFactory(DataSource dataSource, MybatisPlusInterceptor interceptor,
			@Value("${spring.datasource.url:}") String datasourceUrl,
			@Value("${mybatis-plus.global-config.db-config.db-type:sqlite}") String configuredDbType)
			throws Exception {
		MybatisSqlSessionFactoryBean factoryBean = new MybatisSqlSessionFactoryBean();
		factoryBean.setDataSource(dataSource);
		factoryBean.setPlugins(interceptor);
		factoryBean.setTypeHandlersPackage("com.seaskyland.llm.workflow.core.config");
		if (resolveDbType(datasourceUrl, configuredDbType) == DbType.SQLITE) {
			factoryBean.setTypeHandlers(new SQLiteDateTypeHandler());
		}
		return factoryBean.getObject();
	}

	private static DbType resolveDbType(String datasourceUrl, String configuredDbType) {
		String url = StringUtils.defaultString(datasourceUrl).toLowerCase(Locale.ROOT);
		if (url.startsWith("jdbc:sqlite:")) {
			return DbType.SQLITE;
		}
		if (url.startsWith("jdbc:postgresql:")) {
			return DbType.POSTGRE_SQL;
		}

		String value = StringUtils.defaultIfBlank(configuredDbType, "sqlite")
			.trim()
			.toLowerCase(Locale.ROOT);
		if ("postgres".equals(value) || "postgresql".equals(value) || "pgvector".equals(value)) {
			return DbType.POSTGRE_SQL;
		}
		DbType dbType = DbType.getDbType(value);
		if (dbType != DbType.OTHER) {
			return dbType;
		}
		return DbType.valueOf(value.toUpperCase(Locale.ROOT).replace('-', '_'));
	}

}
