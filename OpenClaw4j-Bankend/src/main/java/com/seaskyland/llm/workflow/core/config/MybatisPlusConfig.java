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
import org.apache.ibatis.session.SqlSessionFactory;
import org.mybatis.spring.annotation.MapperScan;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.ImportRuntimeHints;
import org.springframework.core.env.Environment;

import javax.sql.DataSource;

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
	 * Configures MyBatis-Plus interceptor with MySQL pagination support.
	 * @return MybatisPlusInterceptor instance
	 */
	@Bean
	public MybatisPlusInterceptor mybatisPlusInterceptor(Environment environment) {
		MybatisPlusInterceptor interceptor = new MybatisPlusInterceptor();
		interceptor.addInnerInterceptor(new PaginationInnerInterceptor(resolveDbType(environment)));
		return interceptor;
	}

	@Bean
	public SqlSessionFactory sqlSessionFactory(DataSource dataSource, MybatisPlusInterceptor interceptor,
			Environment environment)
			throws Exception {
		MybatisSqlSessionFactoryBean factoryBean = new MybatisSqlSessionFactoryBean();
		factoryBean.setDataSource(dataSource);
		factoryBean.setPlugins(interceptor);
		return factoryBean.getObject();
	}

	private DbType resolveDbType(Environment environment) {
		return DbType.MYSQL;
	}

}
