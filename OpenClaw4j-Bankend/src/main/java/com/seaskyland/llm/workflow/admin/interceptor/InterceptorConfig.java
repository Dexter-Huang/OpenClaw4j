/*
 * Copyright 2024-2025 the original author or authors.
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
package com.seaskyland.llm.workflow.admin.interceptor;

import com.seaskyland.llm.workflow.openapi.interceptor.ApiKeyAuthInterceptor;
import lombok.RequiredArgsConstructor;
import org.springframework.boot.web.servlet.FilterRegistrationBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.web.servlet.config.annotation.WebMvcConfigurer;

/**
 * Configuration class for setting up authenticated web request filters.
 */
@Configuration
@RequiredArgsConstructor
public class InterceptorConfig implements WebMvcConfigurer {

	/** Filter for token-based authentication */
	private final TokenAuthInterceptor tokenAuthInterceptor;

	/** Filter for API key authentication */
	private final ApiKeyAuthInterceptor apiKeyAuthInterceptor;

	@Bean
	public FilterRegistrationBean<TokenAuthInterceptor> tokenAuthFilterRegistration() {
		FilterRegistrationBean<TokenAuthInterceptor> registration = new FilterRegistrationBean<>(tokenAuthInterceptor);
		registration.addUrlPatterns("/console/v1/*", "/starter.zip");
		registration.setOrder(10);
		return registration;
	}

	@Bean
	public FilterRegistrationBean<ApiKeyAuthInterceptor> apiKeyAuthFilterRegistration() {
		FilterRegistrationBean<ApiKeyAuthInterceptor> registration = new FilterRegistrationBean<>(apiKeyAuthInterceptor);
		registration.addUrlPatterns("/api/v1/*");
		registration.setOrder(20);
		return registration;
	}

}
