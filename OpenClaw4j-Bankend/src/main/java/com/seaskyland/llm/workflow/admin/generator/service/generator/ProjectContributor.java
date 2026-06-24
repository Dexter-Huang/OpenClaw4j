package com.seaskyland.llm.workflow.admin.generator.service.generator;

import java.io.IOException;
import java.nio.file.Path;

@FunctionalInterface
public interface ProjectContributor {

	void contribute(Path projectRoot) throws IOException;

}
