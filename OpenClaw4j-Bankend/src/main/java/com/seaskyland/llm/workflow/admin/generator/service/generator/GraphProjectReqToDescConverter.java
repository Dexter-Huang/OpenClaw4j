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

import java.text.Normalizer;
import java.util.Optional;

import com.seaskyland.llm.workflow.admin.generator.exception.NotImplementedException;
import com.seaskyland.llm.workflow.admin.generator.model.AppModeEnum;
import com.seaskyland.llm.workflow.admin.generator.service.dsl.DSLDialectType;

import org.springframework.util.StringUtils;

public class GraphProjectReqToDescConverter {

	public GraphProjectDescription convert(GraphProjectRequest request) {
		GraphProjectDescription description = new GraphProjectDescription();
		description.setPackageName(cleanInputValue(request.getPackageName()));
		description.setLanguage(StringUtils.hasText(request.getLanguage()) ? request.getLanguage() : "java");
		description.setDsl(request.getDsl());

		AppModeEnum appModeEnum = Optional.ofNullable(AppModeEnum.of(request.getAppMode()))
			.orElseThrow(() -> new NotImplementedException("Unsupported appMode: " + request.getAppMode()));
		description.setAppMode(appModeEnum);

		DSLDialectType dslDialectType = DSLDialectType.fromValue(request.getDslDialectType())
			.orElseThrow(
					() -> new NotImplementedException("Unsupported dslDialectType: " + request.getDslDialectType()));
		description.setDslDialectType(dslDialectType);
		return description;
	}

	protected String cleanInputValue(String value) {
		return StringUtils.hasText(value) ? Normalizer.normalize(value, Normalizer.Form.NFKD).replaceAll("\\p{M}", "")
				: value;
	}

}
