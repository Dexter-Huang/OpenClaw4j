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

package com.seaskyland.llm.workflow.core.workflow.runtime;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;

import com.google.common.collect.Lists;
import com.seaskyland.llm.workflow.core.config.CommonConfig;
import com.seaskyland.llm.workflow.core.workflow.WorkflowConfig;
import com.seaskyland.llm.workflow.core.workflow.WorkflowInnerService;
import com.seaskyland.llm.workflow.runtime.domain.workflow.Edge;
import com.seaskyland.llm.workflow.runtime.domain.workflow.Node;
import com.seaskyland.llm.workflow.runtime.domain.workflow.NodeTypeEnum;
import java.util.Map;
import org.jgrapht.graph.DirectedAcyclicGraph;
import org.junit.jupiter.api.Test;
import org.springframework.ai.chat.memory.ChatMemory;

class WorkflowExecuteManagerTest {

  @Test
  void constructExecutableGraphIgnoresCanvasNodesUnreachableFromStart() {
    WorkflowExecuteManager manager =
        new WorkflowExecuteManager(
            Map.of(),
            mock(WorkflowInnerService.class),
            mock(ChatMemory.class),
            mock(CommonConfig.class));
    WorkflowConfig config = new WorkflowConfig();
    config.setNodes(
        Lists.newArrayList(
            node("Start_1", NodeTypeEnum.START),
            node("Classifier_1", NodeTypeEnum.CLASSIFIER),
            node("End_1", NodeTypeEnum.END),
            node("Script_1", NodeTypeEnum.SCRIPT)));
    config.setEdges(
        Lists.newArrayList(edge("Start_1", "Classifier_1"), edge("Classifier_1", "End_1")));

    DirectedAcyclicGraph<String, Edge> graph = manager.constructExecutableGraph(config);

    assertThat(graph.vertexSet()).containsExactlyInAnyOrder("Start_1", "Classifier_1", "End_1");
    assertThat(graph.edgeSet()).hasSize(2);
  }

  private static Node node(String id, NodeTypeEnum type) {
    Node node = new Node();
    node.setId(id);
    node.setName(id);
    node.setType(type.getCode());
    return node;
  }

  private static Edge edge(String source, String target) {
    Edge edge = new Edge();
    edge.setId(source + "_to_" + target);
    edge.setSource(source);
    edge.setSourceHandle(source);
    edge.setTarget(target);
    edge.setTargetHandle(target);
    return edge;
  }
}
