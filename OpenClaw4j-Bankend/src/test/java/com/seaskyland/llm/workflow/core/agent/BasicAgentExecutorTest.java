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

package com.seaskyland.llm.workflow.core.agent;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

import com.seaskyland.llm.workflow.core.agent.tool.AgentToolCallRecorder;
import com.seaskyland.llm.workflow.core.base.manager.AppComponentManager;
import com.seaskyland.llm.workflow.core.base.manager.DocumentRetrieverManager;
import com.seaskyland.llm.workflow.core.base.manager.FileManager;
import com.seaskyland.llm.workflow.core.base.service.McpServerService;
import com.seaskyland.llm.workflow.core.base.service.PluginService;
import com.seaskyland.llm.workflow.core.base.service.ToolExecutionService;
import com.seaskyland.llm.workflow.core.config.CommonConfig;
import com.seaskyland.llm.workflow.core.model.llm.ModelFactory;
import com.seaskyland.llm.workflow.runtime.domain.agent.AgentRequest;
import com.seaskyland.llm.workflow.runtime.domain.agent.AgentResponse;
import com.seaskyland.llm.workflow.runtime.domain.agent.AgentStatus;
import com.seaskyland.llm.workflow.runtime.domain.app.AgentConfig;
import com.seaskyland.llm.workflow.runtime.domain.chat.ChatMessage;
import com.seaskyland.llm.workflow.runtime.domain.chat.ContentType;
import com.seaskyland.llm.workflow.runtime.domain.chat.MessageRole;
import com.seaskyland.llm.workflow.runtime.domain.chat.ToolCall;
import com.seaskyland.llm.workflow.runtime.domain.chat.ToolCallType;
import com.seaskyland.llm.workflow.runtime.domain.tool.InputSchema;
import com.seaskyland.llm.workflow.runtime.domain.tool.ToolCallSchema;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import org.junit.jupiter.api.Test;
import org.springframework.ai.chat.memory.ChatMemory;
import org.springframework.ai.chat.messages.AssistantMessage;
import org.springframework.ai.chat.metadata.ChatGenerationMetadata;
import org.springframework.ai.chat.metadata.ChatResponseMetadata;
import org.springframework.ai.chat.model.ChatModel;
import org.springframework.ai.chat.model.ChatResponse;
import org.springframework.ai.chat.model.Generation;
import org.springframework.ai.chat.prompt.ChatOptions;
import org.springframework.ai.chat.prompt.Prompt;
import reactor.core.publisher.Flux;

class BasicAgentExecutorTest {

  @Test
  void streamExecuteUsesNonStreamingExecutionWhenToolsAreConfigured() {
    SynchronousChatModel chatModel = new SynchronousChatModel();
    ModelFactory modelFactory = mock(ModelFactory.class);
    when(modelFactory.getChatModel("test-provider")).thenReturn(chatModel);

    AppComponentManager appComponentManager = mock(AppComponentManager.class);
    HashMap<String, ToolCallSchema> toolSchemas = new HashMap<>();
    toolSchemas.put("component-1", toolCallSchema());
    when(appComponentManager.getToolCallSchema(List.of("component-1"))).thenReturn(toolSchemas);

    BasicAgentExecutor executor =
        new BasicAgentExecutor(
            mock(ToolExecutionService.class),
            mock(PluginService.class),
            mock(McpServerService.class),
            appComponentManager,
            mock(DocumentRetrieverManager.class),
            mock(ChatMemory.class),
            mock(CommonConfig.class),
            modelFactory,
            mock(FileManager.class));

    List<AgentResponse> responses =
        executor.streamExecute(agentContext(), agentRequest()).collectList().block();

    assertThat(chatModel.callCount).isEqualTo(1);
    assertThat(chatModel.streamCount).isZero();
    assertThat(responses).hasSize(1);
    assertThat(responses.getFirst().getStatus()).isEqualTo(AgentStatus.COMPLETED);
    assertThat(responses.getFirst().getMessage().getContent()).isEqualTo("call response");
  }

  @Test
  void executeKeepsLocalToolLoopWhenToolAdvisorCouldReturnDirect() {
    ToolCallingChatModel chatModel = new ToolCallingChatModel();
    ModelFactory modelFactory = mock(ModelFactory.class);
    when(modelFactory.getChatModel("test-provider")).thenReturn(chatModel);

    AppComponentManager appComponentManager = mock(AppComponentManager.class);
    HashMap<String, ToolCallSchema> toolSchemas = new HashMap<>();
    toolSchemas.put("component-1", toolCallSchema());
    when(appComponentManager.getToolCallSchema(List.of("component-1"))).thenReturn(toolSchemas);
    when(appComponentManager.executeAgentComponent(org.mockito.ArgumentMatchers.any()))
        .thenReturn(
            AgentResponse.builder()
                .message(ChatMessage.builder().content("component output").build())
                .build());

    BasicAgentExecutor executor =
        new BasicAgentExecutor(
            mock(ToolExecutionService.class),
            mock(PluginService.class),
            mock(McpServerService.class),
            appComponentManager,
            mock(DocumentRetrieverManager.class),
            mock(ChatMemory.class),
            mock(CommonConfig.class),
            modelFactory,
            mock(FileManager.class));

    AgentResponse response = executor.execute(agentContext(), agentRequest());

    assertThat(chatModel.callCount).isEqualTo(2);
    assertThat(chatModel.streamCount).isZero();
    assertThat(response.getMessage().getContent()).isEqualTo("final response");
    assertThat(response.getMessage().getToolCalls())
        .extracting(ToolCall::getType)
        .containsExactly(ToolCallType.COMPONENT_TOOL_CALL, ToolCallType.COMPONENT_TOOL_RESULT);
  }

  @Test
  void executeProcessesMoreThanTenToolCallRoundsBeforeReturningFinalResponse() {
    MultiRoundToolCallingChatModel chatModel = new MultiRoundToolCallingChatModel();
    ModelFactory modelFactory = mock(ModelFactory.class);
    when(modelFactory.getChatModel("test-provider")).thenReturn(chatModel);

    AppComponentManager appComponentManager = mock(AppComponentManager.class);
    HashMap<String, ToolCallSchema> toolSchemas = new HashMap<>();
    toolSchemas.put("component-1", toolCallSchema());
    when(appComponentManager.getToolCallSchema(List.of("component-1"))).thenReturn(toolSchemas);
    when(appComponentManager.executeAgentComponent(org.mockito.ArgumentMatchers.any()))
        .thenReturn(
            AgentResponse.builder()
                .message(ChatMessage.builder().content("component output").build())
                .build());

    BasicAgentExecutor executor =
        new BasicAgentExecutor(
            mock(ToolExecutionService.class),
            mock(PluginService.class),
            mock(McpServerService.class),
            appComponentManager,
            mock(DocumentRetrieverManager.class),
            mock(ChatMemory.class),
            mock(CommonConfig.class),
            modelFactory,
            mock(FileManager.class));

    AgentResponse response = executor.execute(agentContext(), agentRequest());

    assertThat(chatModel.callCount).isEqualTo(12);
    assertThat(response.getStatus()).isEqualTo(AgentStatus.COMPLETED);
    assertThat(response.getMessage().getContent()).isEqualTo("final response");
    assertThat(response.getMessage().getToolCalls()).hasSize(22);
  }

  @Test
  void executeIncludesDirectReturnToolCallsInResponse() {
    DirectReturnToolChatModel chatModel = new DirectReturnToolChatModel();
    ModelFactory modelFactory = mock(ModelFactory.class);
    when(modelFactory.getChatModel("test-provider")).thenReturn(chatModel);

    AppComponentManager appComponentManager = mock(AppComponentManager.class);
    HashMap<String, ToolCallSchema> toolSchemas = new HashMap<>();
    toolSchemas.put("component-1", toolCallSchema());
    when(appComponentManager.getToolCallSchema(List.of("component-1"))).thenReturn(toolSchemas);

    BasicAgentExecutor executor =
        new BasicAgentExecutor(
            mock(ToolExecutionService.class),
            mock(PluginService.class),
            mock(McpServerService.class),
            appComponentManager,
            mock(DocumentRetrieverManager.class),
            mock(ChatMemory.class),
            mock(CommonConfig.class),
            modelFactory,
            mock(FileManager.class));

    AgentResponse response = executor.execute(agentContext(), agentRequest());

    assertThat(response.getMessage().getContent()).isEqualTo("component output");
    assertThat(response.getMessage().getToolCalls())
        .extracting(ToolCall::getType)
        .containsExactly(ToolCallType.MCP_TOOL_CALL, ToolCallType.MCP_TOOL_RESULT);
    assertThat(response.getMessage().getToolCalls())
        .extracting(toolCall -> toolCall.getFunction().getName())
        .containsExactly("python_execute", "python_execute");
    assertThat(response.getMessage().getToolCalls().get(1).getFunction().getOutput())
        .isEqualTo("component output");
  }

  private static AgentContext agentContext() {
    AgentConfig config = new AgentConfig();
    config.setModelProvider("test-provider");
    config.setModel("test-model");
    config.setAgentComponents(List.of("component-1"));

    AgentContext context = new AgentContext();
    context.setConfig(config);
    context.setRequest(agentRequest());
    return context;
  }

  private static AgentRequest agentRequest() {
    AgentRequest request = new AgentRequest();
    request.setMessages(
        List.of(
            ChatMessage.builder()
                .role(MessageRole.USER)
                .contentType(ContentType.TEXT)
                .content("call a tool")
                .build()));
    request.setExtraPrams(Map.of());
    return request;
  }

  private static ToolCallSchema toolCallSchema() {
    InputSchema inputSchema = new InputSchema();
    inputSchema.setType("object");
    inputSchema.setProperties(Map.of());

    ToolCallSchema schema = new ToolCallSchema();
    schema.setName("component_tool");
    schema.setDescription("component tool");
    schema.setInputSchema(inputSchema);
    return schema;
  }

  private static AssistantMessage.ToolCall componentToolCall() {
    return new AssistantMessage.ToolCall("tool-call-1", "function", "component_tool", "{}");
  }

  private static final class SynchronousChatModel implements ChatModel {

    private int callCount;

    private int streamCount;

    @Override
    public ChatResponse call(Prompt prompt) {
      callCount++;
      return ChatResponse.builder()
          .metadata(ChatResponseMetadata.builder().model("test-model").build())
          .generations(
              List.of(
                  new Generation(
                      new AssistantMessage("call response"),
                      ChatGenerationMetadata.builder().finishReason("stop").build())))
          .build();
    }

    @Override
    public Flux<ChatResponse> stream(Prompt prompt) {
      streamCount++;
      return Flux.error(new AssertionError("stream should not be used when tools are configured"));
    }

    @Override
    public ChatOptions getOptions() {
      return ChatOptions.builder().build();
    }
  }

  private static final class ToolCallingChatModel implements ChatModel {

    private int callCount;

    private int streamCount;

    @Override
    public ChatResponse call(Prompt prompt) {
      callCount++;
      if (callCount == 1) {
        return ChatResponse.builder()
            .metadata(ChatResponseMetadata.builder().model("test-model").build())
            .generations(
                List.of(
                    new Generation(
                        AssistantMessage.builder()
                            .content("")
                            .toolCalls(List.of(componentToolCall()))
                            .build(),
                        ChatGenerationMetadata.builder().finishReason("tool_calls").build())))
            .build();
      }

      return ChatResponse.builder()
          .metadata(ChatResponseMetadata.builder().model("test-model").build())
          .generations(
              List.of(
                  new Generation(
                      new AssistantMessage("final response"),
                      ChatGenerationMetadata.builder().finishReason("stop").build())))
          .build();
    }

    @Override
    public Flux<ChatResponse> stream(Prompt prompt) {
      streamCount++;
      return Flux.error(new AssertionError("stream should not be called"));
    }

    @Override
    public ChatOptions getOptions() {
      return ChatOptions.builder().build();
    }
  }

  private static final class MultiRoundToolCallingChatModel implements ChatModel {

    private int callCount;

    @Override
    public ChatResponse call(Prompt prompt) {
      callCount++;
      if (callCount < 12) {
        return ChatResponse.builder()
            .metadata(ChatResponseMetadata.builder().model("test-model").build())
            .generations(
                List.of(
                    new Generation(
                        AssistantMessage.builder()
                            .content("")
                            .toolCalls(List.of(componentToolCall()))
                            .build(),
                        ChatGenerationMetadata.builder().finishReason("tool_calls").build())))
            .build();
      }

      return ChatResponse.builder()
          .metadata(ChatResponseMetadata.builder().model("test-model").build())
          .generations(
              List.of(
                  new Generation(
                      new AssistantMessage("final response"),
                      ChatGenerationMetadata.builder().finishReason("stop").build())))
          .build();
    }

    @Override
    public Flux<ChatResponse> stream(Prompt prompt) {
      return Flux.error(new AssertionError("stream should not be called"));
    }

    @Override
    public ChatOptions getOptions() {
      return ChatOptions.builder().build();
    }
  }

  private static final class DirectReturnToolChatModel implements ChatModel {

    @Override
    public ChatResponse call(Prompt prompt) {
      String output = "component output";
      AgentToolCallRecorder.record(
          "tool-call-1",
          ToolCallType.MCP_TOOL_CALL,
          "python_execute",
          "{\"query\":\"hello\"}",
          output);

      return ChatResponse.builder()
          .metadata(ChatResponseMetadata.builder().model("test-model").build())
          .generations(
              List.of(
                  new Generation(
                      new AssistantMessage(output),
                      ChatGenerationMetadata.builder().finishReason("stop").build())))
          .build();
    }

    @Override
    public Flux<ChatResponse> stream(Prompt prompt) {
      return Flux.error(new AssertionError("stream should not be called"));
    }

    @Override
    public ChatOptions getOptions() {
      return ChatOptions.builder().build();
    }
  }
}
