package com.seaskyland.llm.workflow.admin.generator.service.generator;

import java.io.IOException;
import java.util.Map;

import org.springframework.stereotype.Component;

@Component
public class SimpleTemplateRenderer {

	public String render(String templateName, Map<String, Object> model) throws IOException {
		throw new UnsupportedOperationException("Generator templates are disabled during the Spring Boot 4 migration.");
	}

}
