/*
 * Copyright 2025 the original author or authors.
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

import com.seaskyland.llm.workflow.core.base.service.ToolExecutionService;
import com.seaskyland.llm.workflow.core.utils.api.OpenApiUtils;
import com.seaskyland.llm.workflow.runtime.domain.chat.ToolCallType;
import com.seaskyland.llm.workflow.runtime.domain.plugin.Tool;
import com.seaskyland.llm.workflow.runtime.domain.plugin.ToolExecutionRequest;
import com.seaskyland.llm.workflow.runtime.domain.plugin.ToolExecutionResult;
import com.seaskyland.llm.workflow.runtime.domain.tool.ToolCallSchema;
import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
import java.util.Map;
import org.jetbrains.annotations.NotNull;
import org.springframework.ai.tool.definition.ToolDefinition;
import org.springframework.ai.tool.metadata.ToolMetadata;

/**
 * Plugin tool callback implementation for handling tool execution requests.
 *
 * @since 1.0.0.3
 */
public class PluginToolCallback implements AgentToolCallback {

  /** Service for executing tool operations */
  private final ToolExecutionService toolExecutionService;

  /** Tool configuration and metadata */
  private final Tool tool;

  /** Additional parameters for tool execution */
  private final Map<String, Object> extraParams;

  /**
   * Creates a new PluginToolCallback instance.
   *
   * @param toolExecutionService Service for executing tool operations
   * @param tool Tool configuration
   * @param extraParams Additional execution parameters
   */
  public PluginToolCallback(
      ToolExecutionService toolExecutionService, Tool tool, Map<String, Object> extraParams) {
    this.toolExecutionService = toolExecutionService;
    this.tool = tool;
    this.extraParams = extraParams;
  }

  /** Gets the tool definition including name, description and input schema. */
  @NotNull
  @Override
  public ToolDefinition getToolDefinition() {
    ToolCallSchema schema = OpenApiUtils.buildToolCallSchema(this.tool);
    return ToolDefinition.builder()
        .name(schema.getName())
        .description(schema.getDescription())
        .inputSchema(JsonUtils.toJson(schema.getInputSchema()))
        .build();
  }

  /**
   * Executes the tool with the given input parameters. Merges any extra parameters if available.
   *
   * @param functionInput JSON string containing function input parameters
   * @return JSON string containing the execution result
   */
  @NotNull
  @Override
  public String call(@NotNull String functionInput) {
    String toolId = tool.getToolId();
    Map<String, Object> arguments =
        ToolArgumentsHelper.mergeToolArguments(functionInput, extraParams, toolId);

    ToolExecutionResult response =
        this.toolExecutionService.callOpenApi(
            ToolExecutionRequest.builder().tool(this.tool).arguments(arguments).build());

    if (!response.isSuccess()) {
      return JsonUtils.toJson(response.getError());
    }

    return response.getOutput();
  }

  /**
   * Gets the tool metadata configuration.
   *
   * @return ToolMetadata with returnDirect set to true
   */
  @NotNull
  @Override
  public ToolMetadata getToolMetadata() {
    return ToolMetadata.builder().returnDirect(true).build();
  }

  @Override
  public String getId() {
    return tool.getToolId();
  }

  @Override
  public ToolCallType getToolCallType() {
    return ToolCallType.TOOL_CALL;
  }
}
