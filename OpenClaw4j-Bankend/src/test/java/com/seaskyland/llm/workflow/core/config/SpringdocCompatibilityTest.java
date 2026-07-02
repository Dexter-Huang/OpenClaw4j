package com.seaskyland.llm.workflow.core.config;

import org.junit.jupiter.api.Test;
import org.springdoc.core.customizers.QuerydslPredicateOperationCustomizer;
import org.springframework.util.ReflectionUtils;

import static org.assertj.core.api.Assertions.assertThatCode;

class SpringdocCompatibilityTest {

	@Test
	void querydslCustomizerCanBeIntrospectedWithSpringData41() {
		assertThatCode(() -> ReflectionUtils.getDeclaredMethods(QuerydslPredicateOperationCustomizer.class))
			.doesNotThrowAnyException();
	}

}
