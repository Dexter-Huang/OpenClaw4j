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

package com.seaskyland.llm.workflow.core.workflow.processor.impl;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyMap;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.seaskyland.llm.workflow.core.base.manager.CacheManager;
import com.seaskyland.llm.workflow.core.base.manager.FileManager;
import com.seaskyland.llm.workflow.core.base.manager.ModelExecuteManager;
import com.seaskyland.llm.workflow.core.config.CommonConfig;
import com.seaskyland.llm.workflow.core.config.StudioProperties;
import com.seaskyland.llm.workflow.core.workflow.WorkflowContext;
import com.seaskyland.llm.workflow.core.workflow.WorkflowInnerService;
import com.seaskyland.llm.workflow.runtime.domain.agent.AgentResponse;
import com.seaskyland.llm.workflow.runtime.domain.chat.ChatMessage;
import com.seaskyland.llm.workflow.runtime.domain.file.File;
import com.seaskyland.llm.workflow.runtime.domain.workflow.Edge;
import com.seaskyland.llm.workflow.runtime.domain.workflow.Node;
import com.seaskyland.llm.workflow.runtime.domain.workflow.NodeResult;
import com.seaskyland.llm.workflow.runtime.domain.workflow.NodeTypeEnum;
import java.util.List;
import java.util.Map;
import org.jgrapht.graph.DirectedAcyclicGraph;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;
import org.springframework.ai.chat.memory.ChatMemory;
import org.springframework.ai.chat.messages.Message;
import org.springframework.ai.chat.messages.UserMessage;
import org.springframework.http.MediaType;
import reactor.core.publisher.Flux;

class ClassifierExecuteProcessorTest {

  @Test
  void innerExecuteSendsVisionFileAsModelMedia() {
    ModelExecuteManager modelExecuteManager = mock(ModelExecuteManager.class);
    FileManager fileManager = mock(FileManager.class);
    when(modelExecuteManager.stream(eq("Tongyi"), eq("qwen-vl"), anyMap(), any()))
        .thenReturn(
            Flux.just(
                AgentResponse.builder()
                    .message(ChatMessage.builder().content("-1").build())
                    .build()));
    when(fileManager.getMediaTypeFromUrl("https://example.com/question.png"))
        .thenReturn(MediaType.IMAGE_PNG);

    ClassifierExecuteProcessor processor =
        new ClassifierExecuteProcessor(
            mock(CacheManager.class),
            mock(WorkflowInnerService.class),
            mock(ChatMemory.class),
            mock(CommonConfig.class),
            mock(StudioProperties.class),
            modelExecuteManager,
            fileManager);

    WorkflowContext context = new WorkflowContext();
    context.setRequestId("req-vision-classifier");
    context.setTaskStatus("RUNNING");
    context
        .getVariablesMap()
        .put(
            "Start_1",
            Map.of(
                "query",
                "请判断图片里的问题类型",
                "img",
                File.builder()
                    .name("question.png")
                    .mimeType("image/png")
                    .source(File.SourceEnum.remoteUrl.name())
                    .url("https://example.com/question.png")
                    .build()));

    NodeResult result = processor.innerExecute(buildGraph(), buildClassifierNode(), context);

    assertThat(result.getOutput()).contains("\"subject\":\"Other\"");

    ArgumentCaptor<List<Message>> messagesCaptor = ArgumentCaptor.forClass(List.class);
    verify(modelExecuteManager)
        .stream(eq("Tongyi"), eq("qwen-vl"), anyMap(), messagesCaptor.capture());
    assertThat(messagesCaptor.getValue())
        .filteredOn(UserMessage.class::isInstance)
        .map(UserMessage.class::cast)
        .anySatisfy(
            message -> {
              assertThat(message.getText()).contains("请判断图片里的问题类型");
              assertThat(message.getMedia()).hasSize(1);
              assertThat(message.getMedia().getFirst().getMimeType().toString())
                  .isEqualTo("image/png");
            });
  }

  private static DirectedAcyclicGraph<String, Edge> buildGraph() {
    DirectedAcyclicGraph<String, Edge> graph = new DirectedAcyclicGraph<>(Edge.class);
    graph.addVertex("Classifier_1");
    graph.addVertex("End_1");

    Edge defaultEdge = new Edge();
    defaultEdge.setId("edge-default");
    defaultEdge.setSource("Classifier_1");
    defaultEdge.setSourceHandle("Classifier_1_default");
    defaultEdge.setTarget("End_1");
    defaultEdge.setTargetHandle("End_1");
    graph.addEdge("Classifier_1", "End_1", defaultEdge);
    return graph;
  }

  private static Node buildClassifierNode() {
    Node node = new Node();
    node.setId("Classifier_1");
    node.setName("意图分类1");
    node.setType(NodeTypeEnum.CLASSIFIER.getCode());

    Node.InputParam textInput = new Node.InputParam();
    textInput.setKey("input");
    textInput.setType("String");
    textInput.setValueFrom("refer");
    textInput.setValue("${Start_1.query}");

    Node.NodeCustomConfig config = new Node.NodeCustomConfig();
    config.setInputParams(Lists.newArrayList(textInput));
    config.setNodeParam(buildNodeParam());
    node.setConfig(config);
    return node;
  }

  private static Map<String, Object> buildNodeParam() {
    Map<String, Object> nodeParam = Maps.newHashMap();
    nodeParam.put("mode_switch", "efficient");
    nodeParam.put("instruction", "");
    nodeParam.put(
        "conditions",
        Lists.newArrayList(Map.of("id", "default", "subject", "")));

    Map<String, Object> visionParam = Maps.newHashMap();
    visionParam.put("key", "imageContent");
    visionParam.put("type", "File");
    visionParam.put("value_from", "refer");
    visionParam.put("value", "${Start_1.img}");

    Map<String, Object> visionConfig = Maps.newHashMap();
    visionConfig.put("enable", true);
    visionConfig.put("params", Lists.newArrayList(visionParam));

    Map<String, Object> modelConfig = Maps.newHashMap();
    modelConfig.put("provider", "Tongyi");
    modelConfig.put("model_id", "qwen-vl");
    modelConfig.put("params", Lists.newArrayList());
    modelConfig.put("vision_config", visionConfig);
    nodeParam.put("model_config", modelConfig);
    return nodeParam;
  }
}
