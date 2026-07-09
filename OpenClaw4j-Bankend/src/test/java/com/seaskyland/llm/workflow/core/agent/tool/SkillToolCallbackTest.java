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

package com.seaskyland.llm.workflow.core.agent.tool;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verifyNoInteractions;
import static org.mockito.Mockito.when;

import com.seaskyland.llm.workflow.core.base.service.SkillService;
import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
import java.util.List;
import java.util.Map;
import org.junit.jupiter.api.Test;

class SkillToolCallbackTest {

  @Test
  void exposesReadSkillFileToolDefinition() {
    SkillToolCallback callback = new SkillToolCallback(List.of("skill-1"), mock(SkillService.class));

    assertThat(callback.getToolDefinition().name()).isEqualTo("read_skill_file");
    assertThat(callback.getToolDefinition().description()).contains("Skill");
    assertThat(callback.getToolDefinition().inputSchema())
        .contains("skill_code")
        .contains("path")
        .contains("SKILL.md");
  }

  @Test
  void readsEnabledSkillFile() {
    SkillService skillService = mock(SkillService.class);
    when(skillService.readSkillFile("skill-1", "SKILL.md"))
        .thenReturn("## 基础信息\n姓名：黄明朗\n籍贯：阳江");
    SkillToolCallback callback = new SkillToolCallback(List.of("skill-1"), skillService);

    String output = callback.call("{\"skill_code\":\"skill-1\",\"path\":\"SKILL.md\"}");

    Map<String, Object> result = JsonUtils.fromJsonToMap(output);
    assertThat(result)
        .containsEntry("skill_code", "skill-1")
        .containsEntry("path", "SKILL.md");
    assertThat(result.get("content")).asString().contains("籍贯：阳江");
  }

  @Test
  void rejectsSkillFileReadWhenSkillIsNotEnabled() {
    SkillService skillService = mock(SkillService.class);
    SkillToolCallback callback = new SkillToolCallback(List.of("skill-1"), skillService);

    String output = callback.call("{\"skill_code\":\"skill-2\",\"path\":\"SKILL.md\"}");

    Map<String, Object> result = JsonUtils.fromJsonToMap(output);
    assertThat(result).containsEntry("success", false);
    assertThat(result.get("error")).asString().contains("not enabled");
    verifyNoInteractions(skillService);
  }
}
