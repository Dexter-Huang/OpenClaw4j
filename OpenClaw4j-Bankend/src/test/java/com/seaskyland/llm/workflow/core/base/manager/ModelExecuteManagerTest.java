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

package com.seaskyland.llm.workflow.core.base.manager;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

import com.seaskyland.llm.workflow.core.model.llm.ModelFactory;
import com.seaskyland.llm.workflow.runtime.domain.agent.AgentResponse;
import java.util.List;
import java.util.Map;
import org.junit.jupiter.api.Test;
import org.springframework.ai.chat.messages.AssistantMessage;
import org.springframework.ai.chat.messages.UserMessage;
import org.springframework.ai.chat.metadata.ChatGenerationMetadata;
import org.springframework.ai.chat.metadata.ChatResponseMetadata;
import org.springframework.ai.chat.metadata.Usage;
import org.springframework.ai.chat.model.ChatModel;
import org.springframework.ai.chat.model.ChatResponse;
import org.springframework.ai.chat.model.Generation;
import org.springframework.ai.chat.prompt.ChatOptions;
import org.springframework.ai.chat.prompt.Prompt;
import org.springframework.test.util.ReflectionTestUtils;
import reactor.core.publisher.Flux;

class ModelExecuteManagerTest {

  @Test
  void streamPassesRequestedModelIdThroughPromptOptions() {
    CapturingChatModel chatModel = new CapturingChatModel();
    ModelFactory modelFactory = mock(ModelFactory.class);
    when(modelFactory.getChatModel("Tongyi")).thenReturn(chatModel);

    ModelExecuteManager manager = new ModelExecuteManager();
    ReflectionTestUtils.setField(manager, "modelFactory", modelFactory);

    List<AgentResponse> responses =
        manager.stream("Tongyi", "qwen-max", Map.of(), List.of(new UserMessage("hello")))
            .collectList()
            .block();

    assertThat(responses).hasSize(1);
    assertThat(chatModel.lastPrompt).isNotNull();
    assertThat(chatModel.lastPrompt.getOptions()).isInstanceOf(ChatOptions.class);
    assertThat(((ChatOptions) chatModel.lastPrompt.getOptions()).getModel()).isEqualTo("qwen-max");
  }

  @Test
  void streamKeepsTerminalUsageMetadataWhenProviderSendsUsageAfterContentChunks() {
    TerminalUsageChatModel chatModel = new TerminalUsageChatModel();
    ModelFactory modelFactory = mock(ModelFactory.class);
    when(modelFactory.getChatModel("Tongyi")).thenReturn(chatModel);

    ModelExecuteManager manager = new ModelExecuteManager();
    ReflectionTestUtils.setField(manager, "modelFactory", modelFactory);

    List<AgentResponse> responses =
        manager.stream("Tongyi", "qwen-max", Map.of(), List.of(new UserMessage("hello")))
            .collectList()
            .block();

    assertThat(responses).hasSize(2);
    assertThat(responses.getLast().getMessage().getContent()).isEqualTo("");
    assertThat(responses.getLast().getUsage().getPromptTokens()).isEqualTo(12);
    assertThat(responses.getLast().getUsage().getCompletionTokens()).isEqualTo(7);
    assertThat(responses.getLast().getUsage().getTotalTokens()).isEqualTo(19);
  }

  private static final class CapturingChatModel implements ChatModel {

    private Prompt lastPrompt;

    @Override
    public ChatResponse call(Prompt prompt) {
      throw new AssertionError("stream should be used");
    }

    @Override
    public Flux<ChatResponse> stream(Prompt prompt) {
      lastPrompt = prompt;
      return Flux.just(
          ChatResponse.builder()
              .metadata(
                  ChatResponseMetadata.builder()
                      .model("qwen-max")
                      .usage(new TokenUsage(1, 1, 2))
                      .build())
              .generations(
                  List.of(
                      new Generation(
                          new AssistantMessage("ok"),
                          ChatGenerationMetadata.builder().finishReason("stop").build())))
              .build());
    }

    @Override
    public ChatOptions getOptions() {
      return ChatOptions.builder().build();
    }
  }

  private static final class TerminalUsageChatModel implements ChatModel {

    @Override
    public ChatResponse call(Prompt prompt) {
      throw new AssertionError("stream should be used");
    }

    @Override
    public Flux<ChatResponse> stream(Prompt prompt) {
      return Flux.just(
          ChatResponse.builder()
              .metadata(
                  ChatResponseMetadata.builder()
                      .model("qwen-max")
                      .usage(new TokenUsage(0, 0, 0))
                      .build())
              .generations(
                  List.of(
                      new Generation(
                          new AssistantMessage("ok"),
                          ChatGenerationMetadata.builder().finishReason("stop").build())))
              .build(),
          ChatResponse.builder()
              .metadata(
                  ChatResponseMetadata.builder()
                      .model("qwen-max")
                      .usage(new TokenUsage(12, 7, 19))
                      .build())
              .generations(List.of())
              .build());
    }

    @Override
    public ChatOptions getOptions() {
      return ChatOptions.builder().build();
    }
  }

  private static final class TokenUsage implements Usage {

    private final int promptTokens;
    private final int completionTokens;
    private final int totalTokens;

    private TokenUsage(int promptTokens, int completionTokens, int totalTokens) {
      this.promptTokens = promptTokens;
      this.completionTokens = completionTokens;
      this.totalTokens = totalTokens;
    }

    @Override
    public Integer getPromptTokens() {
      return promptTokens;
    }

    @Override
    public Integer getCompletionTokens() {
      return completionTokens;
    }

    @Override
    public Integer getTotalTokens() {
      return totalTokens;
    }

    @Override
    public Object getNativeUsage() {
      return null;
    }
  }
}
