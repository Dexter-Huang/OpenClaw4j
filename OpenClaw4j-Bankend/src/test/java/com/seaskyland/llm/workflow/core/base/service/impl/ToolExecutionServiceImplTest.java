/*
 * Copyright 2026 the original author or authors.
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

package com.seaskyland.llm.workflow.core.base.service.impl;

import static org.assertj.core.api.Assertions.assertThat;

import com.seaskyland.llm.workflow.runtime.domain.tool.ApiParameter;
import com.seaskyland.llm.workflow.runtime.enums.ParameterType;
import java.lang.reflect.Method;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import org.junit.jupiter.api.Test;

class ToolExecutionServiceImplTest {

  @Test
  void constructOutputsCopiesScalarArrayValuesFromSource() throws Exception {
    ToolExecutionServiceImpl service = new ToolExecutionServiceImpl(null, null);
    Map<String, Object> source = Map.of("tags", List.of("alpha", "beta"));
    Map<String, Object> target = new HashMap<>();
    ApiParameter outputParam =
        ApiParameter.builder().key("tags").type(ParameterType.ARRAY_STRING.getType()).build();

    Method constructOutputs =
        ToolExecutionServiceImpl.class.getDeclaredMethod(
            "constructOutputs", Map.class, Map.class, List.class);
    constructOutputs.setAccessible(true);
    constructOutputs.invoke(service, source, target, List.of(outputParam));

    assertThat(target).containsEntry("tags", List.of("alpha", "beta"));
  }

  @Test
  void constructOutputsCopiesFileValuesFromSource() throws Exception {
    ToolExecutionServiceImpl service = new ToolExecutionServiceImpl(null, null);
    Map<String, Object> source = Map.of("attachment", "file-001");
    Map<String, Object> target = new HashMap<>();
    ApiParameter outputParam =
        ApiParameter.builder().key("attachment").type(ParameterType.FILE.getType()).build();

    Method constructOutputs =
        ToolExecutionServiceImpl.class.getDeclaredMethod(
            "constructOutputs", Map.class, Map.class, List.class);
    constructOutputs.setAccessible(true);
    constructOutputs.invoke(service, source, target, List.of(outputParam));

    assertThat(target).containsEntry("attachment", "file-001");
  }
}
