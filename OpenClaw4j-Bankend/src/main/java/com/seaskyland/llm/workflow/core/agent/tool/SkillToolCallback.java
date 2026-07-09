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

import com.seaskyland.llm.workflow.core.base.service.SkillService;
import com.seaskyland.llm.workflow.runtime.domain.chat.ToolCallType;
import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import org.apache.commons.lang3.StringUtils;
import org.jetbrains.annotations.NotNull;
import org.springframework.ai.tool.definition.ToolDefinition;
import org.springframework.ai.tool.metadata.ToolMetadata;
import org.springframework.util.CollectionUtils;

/** Agent 内置 Skill 文件读取工具，用于让模型按需加载 Skill 包内文件。 */
public class SkillToolCallback implements AgentToolCallback {

  public static final String TOOL_NAME = "read_skill_file";

  private final Set<String> enabledSkillCodes;

  private final SkillService skillService;

  public SkillToolCallback(List<String> enabledSkillCodes, SkillService skillService) {
    this.enabledSkillCodes =
        CollectionUtils.isEmpty(enabledSkillCodes)
            ? Set.of()
            : new LinkedHashSet<>(
                enabledSkillCodes.stream().filter(StringUtils::isNotBlank).distinct().toList());
    this.skillService = skillService;
  }

  @NotNull
  @Override
  public ToolDefinition getToolDefinition() {
    return ToolDefinition.builder()
        .name(TOOL_NAME)
        .description(
            "读取当前 Agent 已启用 Skill 包内的文件。需要 Skill 内容时先读取主文件 SKILL.md；"
                + "如果主文件引用 references 或其他相对路径文件，再继续按需读取。")
        .inputSchema(
            JsonUtils.toJson(
                Map.of(
                    "type",
                    "object",
                    "properties",
                    Map.of(
                        "skill_code",
                        Map.of("type", "string", "description", "要读取的 Skill code"),
                        "path",
                        Map.of(
                            "type",
                            "string",
                            "description",
                            "Skill 包内相对路径，默认读取 SKILL.md，例如 SKILL.md 或 references/foo.md")),
                    "required",
                    List.of("skill_code", "path"),
                    "additionalProperties",
                    false)))
        .build();
  }

  @NotNull
  @Override
  public String call(@NotNull String functionInput) {
    Map<String, Object> arguments = JsonUtils.fromJsonToMap(functionInput);
    String skillCode = StringUtils.trimToEmpty(String.valueOf(arguments.get("skill_code")));
    String path = StringUtils.trimToEmpty(String.valueOf(arguments.get("path")));
    if (!enabledSkillCodes.contains(skillCode)) {
      String output =
          JsonUtils.toJson(
              Map.of(
                  "success",
                  false,
                  "error",
                  "skill is not enabled for current Agent: " + skillCode));
      AgentToolCallRecorder.record(getToolCallType(), TOOL_NAME, JsonUtils.toJson(arguments), output);
      return output;
    }

    String content = skillService.readSkillFile(skillCode, path);
    String output =
        JsonUtils.toJson(
            Map.of("success", true, "skill_code", skillCode, "path", path, "content", content));
    AgentToolCallRecorder.record(getToolCallType(), TOOL_NAME, JsonUtils.toJson(arguments), output);
    return output;
  }

  @NotNull
  @Override
  public ToolMetadata getToolMetadata() {
    return ToolMetadata.builder().returnDirect(false).build();
  }

  @Override
  public String getId() {
    return TOOL_NAME;
  }

  @Override
  public ToolCallType getToolCallType() {
    return ToolCallType.TOOL_CALL;
  }
}
