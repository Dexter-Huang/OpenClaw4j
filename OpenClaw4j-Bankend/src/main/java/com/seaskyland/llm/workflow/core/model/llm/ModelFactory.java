/*
 * Copyright 2024-2025 the original author or authors.
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
package com.seaskyland.llm.workflow.core.model.llm;

import com.seaskyland.llm.workflow.runtime.domain.knowledgebase.IndexConfig;
import com.seaskyland.llm.workflow.runtime.enums.ErrorCode;
import com.seaskyland.llm.workflow.runtime.exception.BizException;
import com.seaskyland.llm.workflow.core.base.manager.ProviderManager;
import com.seaskyland.llm.workflow.core.model.llm.domain.ModelConfigInfo;
import com.seaskyland.llm.workflow.core.model.llm.domain.ModelCredential;
import com.seaskyland.llm.workflow.core.model.llm.domain.ProviderConfigInfo;
import com.seaskyland.llm.workflow.core.model.embedding.EmbeddingModelDimension;
import com.seaskyland.llm.workflow.core.utils.api.ApiUtils;
import jakarta.annotation.Resource;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.ai.chat.model.ChatModel;
import org.springframework.ai.document.MetadataMode;
import org.springframework.ai.embedding.EmbeddingModel;
import org.springframework.ai.openai.OpenAiChatModel;
import org.springframework.ai.openai.OpenAiChatOptions;
import org.springframework.ai.openai.OpenAiEmbeddingModel;
import org.springframework.ai.openai.OpenAiEmbeddingOptions;
import org.springframework.stereotype.Component;
import org.springframework.http.HttpHeaders;

import java.util.LinkedHashMap;
import java.util.Map;

import static com.seaskyland.llm.workflow.core.rag.RagConstants.DEFAULT_DIMENSION;

/**
 * Factory class for creating various AI model instances including chat, embedding, and
 * document ranking models.
 *
 * @since 1.0.0.3
 */

@Slf4j
@Component
public class ModelFactory {

	/** Map of available AI model providers */
	@Resource
	private Map<String, ModelProvider> providerMap;

	/** Manager for handling provider configurations */
	@Resource
	private ProviderManager providerManager;

	/**
	 * Creates and returns a chat model instance for the specified provider
	 * @param provider The provider name
	 * @return ChatModel instance
	 */
	public ChatModel getChatModel(String provider) {
		ModelCredential credential = getModelCredential(provider, null);
		// TODO will adapt other provider in future, now it's only for OpenAI compatible
		// API

		return OpenAiChatModel.builder().options(buildOpenAiChatOptions(credential)).build();
	}

	/**
	 * Creates and returns an embedding model instance with specified configuration
	 * @param metadataMode The metadata mode for the embedding model
	 * @param indexConfig The index configuration containing model details
	 * @return EmbeddingModel instance
	 */
	public EmbeddingModel getEmbeddingModel(MetadataMode metadataMode, IndexConfig indexConfig) {
		ModelCredential credential = getModelCredential(indexConfig.getEmbeddingProvider(),
				indexConfig.getEmbeddingModel());

		int dimension = EmbeddingModelDimension.getDimension(indexConfig.getEmbeddingModel(), DEFAULT_DIMENSION);

		return OpenAiEmbeddingModel.builder()
			.metadataMode(metadataMode)
			.options(buildOpenAiEmbeddingOptions(credential, indexConfig.getEmbeddingModel(), dimension))
			.build();
	}

	/**
	 * Generates a cache key for model instances
	 * @param modelConfig The model configuration
	 * @return Cache key string
	 */
	private String getModelInstanceKey(ModelConfigInfo modelConfig) {
		return modelConfig.getProvider() + ":" + modelConfig.getModelId();
	}

	/**
	 * Retrieves model credentials for the specified provider and model
	 * @param provider The provider name
	 * @param modelId The model identifier
	 * @return ModelCredential instance
	 */
	private ModelCredential getModelCredential(String provider, String modelId) {
		ProviderConfigInfo providerDetail = providerManager.getProviderDetail(provider, false);
		ModelCredential credential = providerDetail.getCredential();

		if (StringUtils.isBlank(credential.getApiKey())) {
			throw new BizException(ErrorCode.INVALID_PARAMS.toError("apikey", "api key is invalid."));
		}

		return credential;
	}

	/**
	 * Builds OpenAI chat options with the provided credentials.
	 * @param credential The model credentials
	 * @return OpenAiChatOptions instance
	 */
	private OpenAiChatOptions buildOpenAiChatOptions(ModelCredential credential) {
		OpenAiChatOptions.Builder builder = OpenAiChatOptions.builder()
			.apiKey(credential.getApiKey())
			.customHeaders(toSingleValueMap(ApiUtils.getBaseHeaders()));

		if (StringUtils.isNotBlank(credential.getEndpoint())) {
			builder.baseUrl(credential.getEndpoint());
			log.debug("Built OpenAI chat options - endpoint: {}", credential.getEndpoint());
		}

		return builder.build();
	}

	/**
	 * Builds OpenAI embedding options with the provided credentials.
	 * @param credential The model credentials
	 * @param modelId The embedding model identifier
	 * @param dimension The expected embedding dimension
	 * @return OpenAiEmbeddingOptions instance
	 */
	private OpenAiEmbeddingOptions buildOpenAiEmbeddingOptions(ModelCredential credential, String modelId, int dimension) {
		OpenAiEmbeddingOptions.Builder builder = OpenAiEmbeddingOptions.builder()
			.apiKey(credential.getApiKey())
			.customHeaders(toSingleValueMap(ApiUtils.getBaseHeaders()))
			.model(modelId)
			.dimensions(dimension);

		if (StringUtils.isNotBlank(credential.getEndpoint())) {
			builder.baseUrl(credential.getEndpoint());
			log.debug("Built OpenAI embedding options - endpoint: {}, model: {}", credential.getEndpoint(), modelId);
		}

		return builder.build();
	}

	private Map<String, String> toSingleValueMap(HttpHeaders headers) {
		Map<String, String> singleValueHeaders = new LinkedHashMap<>();
		headers.forEach((key, values) -> {
			if (values != null && !values.isEmpty()) {
				singleValueHeaders.put(key, values.get(0));
			}
		});
		return singleValueHeaders;
	}

}
