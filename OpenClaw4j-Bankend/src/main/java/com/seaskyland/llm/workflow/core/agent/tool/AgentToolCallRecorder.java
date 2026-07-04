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

import com.seaskyland.llm.workflow.core.utils.common.IdGenerator;
import com.seaskyland.llm.workflow.runtime.domain.chat.ToolCall;
import com.seaskyland.llm.workflow.runtime.domain.chat.ToolCallType;
import java.util.ArrayList;
import java.util.List;

/** Records direct-return tool calls so agent responses can expose structured tool steps. */
public final class AgentToolCallRecorder {

  private static final ThreadLocal<List<ToolCall>> TOOL_CALLS =
      ThreadLocal.withInitial(ArrayList::new);

  private AgentToolCallRecorder() {}

  public static void clear() {
    TOOL_CALLS.remove();
  }

  public static void record(ToolCallType callType, String name, String arguments, String output) {
    record(IdGenerator.uuid(), callType, name, arguments, output);
  }

  public static void record(
      String id, ToolCallType callType, String name, String arguments, String output) {
    TOOL_CALLS
        .get()
        .add(
            ToolCall.builder()
                .id(id)
                .type(callType)
                .function(ToolCall.Function.builder().name(name).arguments(arguments).build())
                .build());

    TOOL_CALLS
        .get()
        .add(
            ToolCall.builder()
                .id(id)
                .type(toResultType(callType))
                .function(ToolCall.Function.builder().name(name).output(output).build())
                .build());
  }

  public static List<ToolCall> drain() {
    List<ToolCall> toolCalls = new ArrayList<>(TOOL_CALLS.get());
    clear();
    return toolCalls;
  }

  private static ToolCallType toResultType(ToolCallType callType) {
    return switch (callType) {
      case MCP_TOOL_CALL -> ToolCallType.MCP_TOOL_RESULT;
      case COMPONENT_TOOL_CALL -> ToolCallType.COMPONENT_TOOL_RESULT;
      default -> ToolCallType.TOOL_RESULT;
    };
  }
}
