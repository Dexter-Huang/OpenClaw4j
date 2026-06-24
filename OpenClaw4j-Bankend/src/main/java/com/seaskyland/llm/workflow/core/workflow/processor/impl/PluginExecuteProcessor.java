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

import com.seaskyland.llm.workflow.runtime.exception.BizException;
import com.seaskyland.llm.workflow.runtime.enums.ErrorCode;
import com.seaskyland.llm.workflow.runtime.domain.BizError;
import com.seaskyland.llm.workflow.runtime.domain.plugin.ToolExecutionRequest;
import com.seaskyland.llm.workflow.runtime.domain.plugin.ToolExecutionResult;
import com.seaskyland.llm.workflow.runtime.domain.workflow.Edge;
import com.seaskyland.llm.workflow.runtime.domain.workflow.Node;
import com.seaskyland.llm.workflow.runtime.domain.workflow.NodeResult;
import com.seaskyland.llm.workflow.runtime.domain.workflow.NodeStatusEnum;
import com.seaskyland.llm.workflow.runtime.domain.workflow.NodeTypeEnum;
import com.seaskyland.llm.workflow.runtime.domain.workflow.ValueFromEnum;
import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
import com.seaskyland.llm.workflow.core.base.service.ToolExecutionService;
import com.seaskyland.llm.workflow.core.config.CommonConfig;
import com.seaskyland.llm.workflow.core.base.manager.CacheManager;
import com.seaskyland.llm.workflow.core.workflow.WorkflowContext;
import com.seaskyland.llm.workflow.core.utils.common.VariableUtils;
import com.seaskyland.llm.workflow.core.workflow.WorkflowInnerService;
import com.seaskyland.llm.workflow.core.workflow.processor.AbstractExecuteProcessor;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.google.common.collect.Maps;
import lombok.Data;
import lombok.extern.slf4j.Slf4j;
import org.jgrapht.graph.DirectedAcyclicGraph;
import org.springframework.ai.chat.memory.ChatMemory;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.Map;

/**
 * plugin node for executing plugins. It uses the ToolExecutionService to execute the
 * plugin and handles errors listing functionality.
 *
 * @author huangtao
 * @since 1.0.0.3
 */
@Slf4j
@Component("PluginExecuteProcessor")
public class PluginExecuteProcessor extends AbstractExecuteProcessor {

	private final ToolExecutionService toolExecutionService;

	public PluginExecuteProcessor(ToolExecutionService toolExecutionService, CacheManager cacheManager,
			WorkflowInnerService workflowInnerService, ChatMemory conversationChatMemory, CommonConfig commonConfig) {
		super(cacheManager, workflowInnerService, conversationChatMemory, commonConfig);
		this.toolExecutionService = toolExecutionService;
	}

	@Override
	public NodeResult innerExecute(DirectedAcyclicGraph<String, Edge> graph, Node node, WorkflowContext context) {
		long start = System.currentTimeMillis();
		ToolExecutionResult result;
		NodeResult nodeResult = new NodeResult();
		nodeResult.setNodeId(node.getId());
		nodeResult.setNodeName(node.getName());
		nodeResult.setNodeType(node.getType());
		nodeResult.setInput(JsonUtils.toJson(constructInputParamsMap(node, context)));
		ToolExecutionRequest toolExecutionRequest = buildToolRequest(node, context);
		try {
			result = toolExecutionService.executeTool(toolExecutionRequest);
		}
		catch (Exception e) {
			throw new BizException(ErrorCode.WORKFLOW_EXECUTE_ERROR.toError("plugin execute failed"), e);
		}
		finally {
			log.info("plugin execute cost: {}ms, requestId:{}", System.currentTimeMillis() - start,
					context.getRequestId());
		}
		if (result == null || !result.isSuccess()) {
			nodeResult.setOutput(null);
			BizError error = result.getError();
			nodeResult.setErrorInfo(error == null ? null : error.getMessage());
			nodeResult.setError(error);
			nodeResult.setNodeStatus(NodeStatusEnum.FAIL.getCode());
			return nodeResult;
		}
		nodeResult.setOutput(result.getOutput());
		nodeResult.setNodeStatus(NodeStatusEnum.SUCCESS.getCode());
		nodeResult.setUsages(null);
		return nodeResult;
	}

	@Override
	public String getNodeType() {
		return NodeTypeEnum.PLUGIN.getCode();
	}

	@Override
	public String getNodeDescription() {
		return NodeTypeEnum.PLUGIN.getDesc();
	}

	@Override
	public CheckNodeParamResult checkNodeParam(DirectedAcyclicGraph<String, Edge> graph, Node node) {
		CheckNodeParamResult result = super.checkNodeParam(graph, node);
		CheckNodeParamResult inputParamsResult = checkInputParams(node.getConfig().getInputParams());
		if (!inputParamsResult.isSuccess()) {
			result.setSuccess(false);
			result.getErrorInfos().addAll(inputParamsResult.getErrorInfos());
		}
		return result;
	}

	@Data
	public static class NodeParam {

		@JsonProperty("tool_id")
		private String toolId;

		@JsonProperty("tool_name")
		private String toolName;

		private String desc; // 工具描述

		@JsonProperty("plugin_id")
		private String pluginId;

		@JsonProperty("plugin_desc")
		private String pluginDesc;

	}

	private ToolExecutionRequest buildToolRequest(Node node, WorkflowContext context) {
		Map<String, Object> nodeParamString = node.getConfig().getNodeParam();
		NodeParam nodeParam = JsonUtils.fromMap(nodeParamString, NodeParam.class);

		ToolExecutionRequest toolExecutionRequest = new ToolExecutionRequest();
		String toolId = nodeParam.getToolId();
		toolExecutionRequest.setToolId(toolId);
		String pluginId = nodeParam.getPluginId();
		toolExecutionRequest.setPluginId(pluginId);
		toolExecutionRequest.setRequestId(context.getRequestId());

		List<Node.InputParam> inputParams = node.getConfig().getInputParams();
		Map<String, Object> pluginInput = convertToPluginInput(inputParams, context);
		toolExecutionRequest.setArguments(pluginInput);
		return toolExecutionRequest;
	}

	private Map<String, Object> convertToPluginInput(List<Node.InputParam> inputParams, WorkflowContext context) {
		Map<String, Object> maps = Maps.newHashMap();
		inputParams.forEach(inputParam -> {
			if (inputParam.getKey() != null && inputParam.getValueFrom() != null) {
				if (ValueFromEnum.input.name().equals(inputParam.getValueFrom())) {
					maps.put(inputParam.getKey(), inputParam.getValue());
				}
				else {
					maps.put(inputParam.getKey(),
							VariableUtils.getValueFromPayload(
									VariableUtils.getExpressionFromBracket((String) inputParam.getValue()),
									context.getVariablesMap()));
				}
			}
		});
		return maps;
	}

}
