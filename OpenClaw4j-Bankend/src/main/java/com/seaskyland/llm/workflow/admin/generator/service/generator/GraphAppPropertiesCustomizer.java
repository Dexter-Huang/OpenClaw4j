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
package com.seaskyland.llm.workflow.admin.generator.service.generator;

import java.io.InputStream;
import java.util.List;
import java.util.Map;

import org.yaml.snakeyaml.Yaml;

public class GraphAppPropertiesCustomizer {

	public GraphAppPropertiesCustomizer() {
	}

	public void customize(Map<String, Object> properties) {
		try (InputStream in = getClass().getClassLoader().getResourceAsStream("templates/default-application.yml")) {
			if (in == null) {
				throw new IllegalStateException("not found default-default-application.yml");
			}
			Yaml yaml = new Yaml();
			Object loaded = yaml.load(in);
			if (loaded instanceof Map<?, ?>) {
				flattenAndAdd(properties, "", (Map<String, Object>) loaded);
			}
		}
		catch (Exception ex) {
			throw new IllegalStateException("parse default-default-application.yml failed", ex);
		}
	}

	@SuppressWarnings("unchecked")
	private void flattenAndAdd(Map<String, Object> props, String prefix, Map<String, Object> map) {
		for (Map.Entry<String, Object> entry : map.entrySet()) {
			String key = prefix.isEmpty() ? entry.getKey() : prefix + "." + entry.getKey();
			Object value = entry.getValue();
			if (value instanceof Map) {
				flattenAndAdd(props, key, (Map<String, Object>) value);
			}
			else if (value instanceof List) {
				props.put(key, ((List<?>) value).toString());
			}
			else {
				props.put(key, value == null ? "" : value);
			}
		}
	}

}
