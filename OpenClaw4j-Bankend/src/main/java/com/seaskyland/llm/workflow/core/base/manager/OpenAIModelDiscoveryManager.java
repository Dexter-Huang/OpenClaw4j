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
package com.seaskyland.llm.workflow.core.base.manager;

import com.fasterxml.jackson.databind.JsonNode;
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.seaskyland.llm.workflow.core.base.domain.RpcResult;
import com.seaskyland.llm.workflow.core.model.llm.domain.ModelConfigInfo;
import com.seaskyland.llm.workflow.core.model.llm.domain.ModelCredential;
import com.seaskyland.llm.workflow.runtime.enums.DataSourceEnum;
import com.seaskyland.llm.workflow.runtime.enums.ErrorCode;
import com.seaskyland.llm.workflow.runtime.exception.BizException;
import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
import java.util.List;
import java.util.Map;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.http.HttpHeaders;
import org.springframework.stereotype.Component;

/** OpenAI-compatible 远程模型发现服务。 */
@Slf4j
@Component
@RequiredArgsConstructor
public class OpenAIModelDiscoveryManager {

  private final HttpClientManager httpClientManager;

  public List<ModelConfigInfo> fetchRemoteModels(String provider, ModelCredential credential) {
    validateCredential(credential);

    String modelsUrl = buildModelsUrl(credential.getEndpoint());
    String apiKey = credential.getApiKey();
    RpcResult result = doFetch(modelsUrl, apiKey);
    if (!result.isSuccess() && !StringUtils.startsWithIgnoreCase(apiKey, "Bearer ")) {
      result = doFetch(modelsUrl, "Bearer " + apiKey);
    }

    if (!result.isSuccess()) {
      throwRemoteModelsError("remote models fetch failed: " + result.getMessage());
    }

    List<ModelConfigInfo> models = parseModels(provider, result.getResponse());
    if (CollectionUtils.isEmpty(models)) {
      throwRemoteModelsError("remote models response is empty");
    }
    return models;
  }

  private RpcResult doFetch(String modelsUrl, String authorization) {
    Map<String, Object> headers = Maps.newHashMap();
    headers.put(HttpHeaders.AUTHORIZATION, authorization);
    return httpClientManager.doGet(modelsUrl, headers, Maps.newHashMap());
  }

  private void validateCredential(ModelCredential credential) {
    if (credential == null
        || StringUtils.isBlank(credential.getEndpoint())
        || StringUtils.isBlank(credential.getApiKey())) {
      throwRemoteModelsError("remote models endpoint or api_key is empty");
    }
  }

  private String buildModelsUrl(String endpoint) {
    String normalizedEndpoint = StringUtils.stripEnd(endpoint.trim(), "/");
    return normalizedEndpoint + "/models";
  }

  private List<ModelConfigInfo> parseModels(String provider, Object response) {
    try {
      JsonNode root = JsonUtils.fromJson(String.valueOf(response));
      JsonNode data = root.path("data");
      if (!data.isArray()) {
        throwRemoteModelsError("remote models response data is not array");
      }

      List<ModelConfigInfo> models = Lists.newArrayList();
      for (JsonNode item : data) {
        String modelId = item.path("id").asText();
        if (StringUtils.isBlank(modelId)) {
          continue;
        }

        ModelConfigInfo model = new ModelConfigInfo();
        model.setProvider(provider);
        model.setModelId(modelId);
        model.setName(modelId);
        model.setEnable(true);
        model.setSource(DataSourceEnum.custom.name());
        model.setType(inferModelType(modelId));
        model.setMode(ModelConfigInfo.ModeEnum.chat.name());
        model.setTags(Lists.newArrayList());
        models.add(model);
      }
      return models;
    } catch (BizException e) {
      throw e;
    } catch (Exception e) {
      log.error("parse OpenAI-compatible remote models failed", e);
      throwRemoteModelsError("remote models response is invalid");
      return Lists.newArrayList();
    }
  }

  private String inferModelType(String modelId) {
    String lowerModelId = modelId.toLowerCase();
    if (lowerModelId.contains("embedding") || lowerModelId.contains("embed")) {
      return ModelConfigInfo.ModelTypeEnum.text_embedding.name();
    }
    if (lowerModelId.contains("rerank")) {
      return ModelConfigInfo.ModelTypeEnum.rerank.name();
    }
    return ModelConfigInfo.ModelTypeEnum.llm.name();
  }

  private void throwRemoteModelsError(String message) {
    throw new BizException(ErrorCode.INVALID_PARAMS.toError("input_params", message));
  }
}
