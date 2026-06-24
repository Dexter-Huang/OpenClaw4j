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

package com.seaskyland.llm.workflow.admin.generator.model.workflow.nodedata;

import java.util.Collections;
import java.util.List;
import java.util.Map;

import com.seaskyland.llm.workflow.admin.generator.model.Variable;
import com.seaskyland.llm.workflow.admin.generator.model.VariableSelector;
import com.seaskyland.llm.workflow.admin.generator.model.VariableType;
import com.seaskyland.llm.workflow.admin.generator.model.workflow.NodeData;

import com.seaskyland.llm.workflow.admin.generator.service.dsl.DSLDialectType;
import org.springframework.http.HttpMethod;

/**
 * The data model of the HTTP node, which contains all the configurable items of the
 * Builder.。
 */
public class HttpNodeData extends NodeData {

	public static class HttpRequestNodeBody {

		private List<BodyData> data = Collections.emptyList();

		public static HttpRequestNodeBody from(Object rawBody) {
			HttpRequestNodeBody body = new HttpRequestNodeBody();
			Object data = rawBody instanceof Map<?, ?> map ? map.get("data") : rawBody;
			if (data instanceof List<?> list) {
				body.setData(list.stream().filter(Map.class::isInstance).map(Map.class::cast).map(BodyData::from).toList());
			}
			return body;
		}

		public List<BodyData> getData() {
			return data;
		}

		public void setData(List<BodyData> data) {
			this.data = data != null ? data : Collections.emptyList();
		}

	}

	public static class BodyData {

		private String key;

		private String value;

		static BodyData from(Map<?, ?> map) {
			BodyData data = new BodyData();
			data.setKey(stringValue(map.get("key")));
			data.setValue(stringValue(map.get("value")));
			return data;
		}

		public String getKey() {
			return key;
		}

		public void setKey(String key) {
			this.key = key;
		}

		public String getValue() {
			return value;
		}

		public void setValue(String value) {
			this.value = value;
		}

	}

	public static class AuthConfig {

		private String type;

		private String username;

		private String password;

		private String token;

		public static AuthConfig basic(String username, String password) {
			AuthConfig config = new AuthConfig();
			config.type = "basic";
			config.username = username;
			config.password = password;
			return config;
		}

		public static AuthConfig bearer(String token) {
			AuthConfig config = new AuthConfig();
			config.type = "bearer";
			config.token = token;
			return config;
		}

		public boolean isBasic() {
			return "basic".equals(type);
		}

		public boolean isBearer() {
			return "bearer".equals(type);
		}

		public String getUsername() {
			return username;
		}

		public String getPassword() {
			return password;
		}

		public String getToken() {
			return token;
		}

	}

	public static class RetryConfig {

		private final int maxRetries;

		private final long maxRetryInterval;

		private final boolean enable;

		public RetryConfig(int maxRetries, long maxRetryInterval, boolean enable) {
			this.maxRetries = maxRetries;
			this.maxRetryInterval = maxRetryInterval;
			this.enable = enable;
		}

		public int getMaxRetries() {
			return maxRetries;
		}

		public long getMaxRetryInterval() {
			return maxRetryInterval;
		}

		public boolean isEnable() {
			return enable;
		}

	}

	public record TimeoutConfig(int connect, int read, int write, int maxConnect, int maxRead, int maxWrite) {
	}

	private static String stringValue(Object value) {
		return value != null ? value.toString() : null;
	}

	public static List<Variable> getDefaultOutputSchemas(DSLDialectType dialectType) {
		return switch (dialectType) {
			case DIFY ->
				List.of(new Variable("body", VariableType.STRING), new Variable("status_code", VariableType.NUMBER),
						new Variable("headers", VariableType.OBJECT), new Variable("files", VariableType.ARRAY_FILE));
			case STUDIO -> List.of(new Variable("output", VariableType.STRING));
			default -> List.of();
		};
	}

	/** HTTP method, default GET */
	private HttpMethod method;

	/** Request URL */
	private String url;

	/** Request header */
	private Map<String, String> headers;

	/** queryParams */
	private Map<String, String> queryParams;

	/** body */
	private HttpRequestNodeBody body;

	/**
	 * rawBodyMap
	 */
	private Map<String, Object> rawBodyMap;

	/** authConfig */
	private AuthConfig authConfig;

	/** retryConfig */
	private RetryConfig retryConfig;

	/** TimeoutConfig */
	private TimeoutConfig timeoutConfig;

	/** outputKey */
	private String outputKey;

	public HttpNodeData(List<VariableSelector> inputs, List<Variable> outputs, HttpMethod method, String url,
			Map<String, String> headers, Map<String, String> queryParams, HttpRequestNodeBody body,
			AuthConfig authConfig, RetryConfig retryConfig, TimeoutConfig timeoutConfig, String outputKey) {
		super(inputs, outputs);
		this.method = method;
		this.url = url;
		this.headers = headers != null ? headers : Collections.emptyMap();
		this.queryParams = queryParams != null ? queryParams : Collections.emptyMap();
		this.body = body != null ? body : new HttpRequestNodeBody();
		this.authConfig = authConfig;
		this.retryConfig = retryConfig != null ? retryConfig : new RetryConfig(3, 1000, true);
		this.timeoutConfig = timeoutConfig;
		this.outputKey = outputKey;
		this.rawBodyMap = null;
	}

	public HttpMethod getMethod() {
		return method;
	}

	public void setMethod(HttpMethod method) {
		this.method = method;
	}

	public String getUrl() {
		return url;
	}

	public void setUrl(String url) {
		this.url = url;
	}

	public Map<String, String> getHeaders() {
		return headers;
	}

	public void setHeaders(Map<String, String> headers) {
		this.headers = headers;
	}

	public Map<String, String> getQueryParams() {
		return queryParams;
	}

	public void setQueryParams(Map<String, String> queryParams) {
		this.queryParams = queryParams;
	}

	public HttpRequestNodeBody getBody() {
		return body;
	}

	public void setBody(HttpRequestNodeBody body) {
		this.body = body;
	}

	public Map<String, Object> getRawBodyMap() {
		return rawBodyMap;
	}

	public void setRawBodyMap(Map<String, Object> rawBodyMap) {
		this.rawBodyMap = rawBodyMap;
	}

	public AuthConfig getAuthConfig() {
		return authConfig;
	}

	public void setAuthConfig(AuthConfig authConfig) {
		this.authConfig = authConfig;
	}

	public RetryConfig getRetryConfig() {
		return retryConfig;
	}

	public void setRetryConfig(RetryConfig retryConfig) {
		this.retryConfig = retryConfig;
	}

	public TimeoutConfig getTimeoutConfig() {
		return timeoutConfig;
	}

	public void setTimeoutConfig(TimeoutConfig timeoutConfig) {
		this.timeoutConfig = timeoutConfig;
	}

	public String getOutputKey() {
		return outputKey;
	}

	public void setOutputKey(String outputKey) {
		this.outputKey = outputKey;
	}

}
