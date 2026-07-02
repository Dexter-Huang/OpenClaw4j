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

package com.seaskyland.llm.workflow.core.rag.vectorstore.pgvector;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.seaskyland.llm.workflow.core.model.embedding.DefaultBatchingStrategy;
import com.seaskyland.llm.workflow.core.model.embedding.EmbeddingModelDimension;
import com.seaskyland.llm.workflow.core.model.llm.ModelFactory;
import com.seaskyland.llm.workflow.core.rag.DocumentChunkConverter;
import com.seaskyland.llm.workflow.core.rag.vectorstore.VectorStoreService;
import com.seaskyland.llm.workflow.runtime.domain.PagingList;
import com.seaskyland.llm.workflow.runtime.domain.knowledgebase.DocumentChunk;
import com.seaskyland.llm.workflow.runtime.domain.knowledgebase.IndexConfig;
import com.seaskyland.llm.workflow.runtime.enums.ErrorCode;
import com.seaskyland.llm.workflow.runtime.exception.BizException;
import lombok.extern.slf4j.Slf4j;
import org.springframework.ai.document.Document;
import org.springframework.ai.document.MetadataMode;
import org.springframework.ai.embedding.EmbeddingModel;
import org.springframework.ai.vectorstore.SearchRequest;
import org.springframework.ai.vectorstore.VectorStore;
import org.springframework.ai.vectorstore.pgvector.PgVectorStore;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;

import java.sql.ResultSet;
import java.sql.SQLException;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Collectors;

import static com.seaskyland.llm.workflow.core.rag.RagConstants.DEFAULT_DIMENSION;

/**
 * PostgreSQL pgvector-backed vector store service for project RAG data.
 *
 * <p>Each knowledge-base index maps to an isolated pgvector table in the configured
 * schema. Spring AI handles embedding persistence and similarity search; this service
 * adds the project-specific list and status-update operations used by the knowledge
 * management UI.
 */
@Service
@Slf4j
@Qualifier("pgvectorVectorStoreService")
public class PgVectorStoreService implements VectorStoreService {

	private static final Pattern DOC_ID_PATTERN = Pattern.compile(
			"#metadata\\[.doc_id.\\]\\s*==\\s*['\"]([^'\"]+)['\"]");

	private static final TypeReference<Map<String, Object>> METADATA_TYPE = new TypeReference<>() {
	};

	private final ModelFactory modelFactory;

	private final JdbcTemplate jdbcTemplate;

	private final ObjectMapper objectMapper;

	private final String schemaName;

	private final Map<String, PgVectorStore> storeCache = new ConcurrentHashMap<>();

	public PgVectorStoreService(ModelFactory modelFactory, JdbcTemplate jdbcTemplate, ObjectMapper objectMapper,
			@Value("${spring.ai.vectorstore.pgvector.schema-name:openclaw_rag}") String schemaName) {
		this.modelFactory = modelFactory;
		this.jdbcTemplate = jdbcTemplate;
		this.objectMapper = objectMapper;
		this.schemaName = sanitizeName(schemaName);
	}

	@Override
	public void createIndex(IndexConfig indexConfig) {
		getOrCreateStore(indexConfig);
		log.info("pgvector RAG index '{}' is ready", indexConfig.getName());
	}

	@Override
	public void deleteIndex(IndexConfig indexConfig) {
		storeCache.remove(indexConfig.getName());
		jdbcTemplate.execute("DROP TABLE IF EXISTS " + tableName(indexConfig));
		log.info("pgvector RAG index '{}' deleted", indexConfig.getName());
	}

	@Override
	public VectorStore getVectorStore(IndexConfig indexConfig) {
		return getOrCreateStore(indexConfig);
	}

	@Override
	public PagingList<DocumentChunk> listDocumentChunks(IndexConfig indexConfig, SearchRequest searchRequest) {
		try {
			int from = searchRequest.getFrom();
			int size = searchRequest.getTopK();
			String docId = extractDocId(searchRequest);

			long total = countAll(indexConfig, docId);
			List<DocumentChunk> chunks = listAll(indexConfig, from, size, docId).stream()
				.map(DocumentChunkConverter::toDocumentChunk)
				.collect(Collectors.toList());

			int current = size > 0 ? (from / size) + 1 : 1;
			return new PagingList<>(current, size, total, chunks);
		}
		catch (Exception e) {
			throw new BizException(ErrorCode.DOCUMENT_RETRIEVAL_ERROR.toError(), e);
		}
	}

	@Override
	public void updateDocumentChunks(IndexConfig indexConfig, List<DocumentChunk> chunks) {
		try {
			List<Document> documents = chunks.stream()
				.map(DocumentChunkConverter::toDocument)
				.collect(Collectors.toList());
			getOrCreateStore(indexConfig).add(documents);
		}
		catch (Exception e) {
			throw new BizException(ErrorCode.UPDATE_DOCUMENT_CHUNK_ERROR.toError(), e);
		}
	}

	@Override
	public void updateDocumentChunkStatus(IndexConfig indexConfig, List<String> chunkIds, boolean enabled) {
		if (chunkIds == null || chunkIds.isEmpty()) {
			return;
		}
		try {
			String placeholders = chunkIds.stream().map(id -> "?").collect(Collectors.joining(", "));
			Object[] args = new Object[chunkIds.size() + 1];
			args[0] = enabled;
			for (int i = 0; i < chunkIds.size(); i++) {
				args[i + 1] = chunkIds.get(i);
			}
			jdbcTemplate.update("UPDATE " + tableName(indexConfig)
					+ " SET metadata = jsonb_set(metadata::jsonb, '{enabled}', to_jsonb(?::boolean), true)::json "
					+ "WHERE id IN (" + placeholders + ")", args);
		}
		catch (Exception e) {
			throw new BizException(ErrorCode.UPDATE_DOCUMENT_CHUNK_ERROR.toError(), e);
		}
	}

	private PgVectorStore getOrCreateStore(IndexConfig indexConfig) {
		return storeCache.computeIfAbsent(indexConfig.getName(), name -> {
			EmbeddingModel embeddingModel = modelFactory.getEmbeddingModel(MetadataMode.EMBED, indexConfig);
			int dimension = EmbeddingModelDimension.getDimension(indexConfig.getEmbeddingModel(), DEFAULT_DIMENSION);

			PgVectorStore store = PgVectorStore.builder(jdbcTemplate, embeddingModel)
				.schemaName(schemaName)
				.vectorTableName(tableNameSuffix(name))
				.idType(PgVectorStore.PgIdType.TEXT)
				.dimensions(dimension)
				.distanceType(PgVectorStore.PgDistanceType.COSINE_DISTANCE)
				.indexType(PgVectorStore.PgIndexType.HNSW)
				.initializeSchema(true)
				.vectorTableValidationsEnabled(true)
				.batchingStrategy(new DefaultBatchingStrategy())
				.build();
			try {
				store.afterPropertiesSet();
			}
			catch (Exception e) {
				throw new IllegalStateException("Failed to initialize pgvector RAG index: " + name, e);
			}
			return store;
		});
	}

	private long countAll(IndexConfig indexConfig, String docId) {
		if (docId == null) {
			Long total = jdbcTemplate.queryForObject("SELECT COUNT(*) FROM " + tableName(indexConfig), Long.class);
			return total == null ? 0L : total;
		}
		Long total = jdbcTemplate.queryForObject(
				"SELECT COUNT(*) FROM " + tableName(indexConfig) + " WHERE metadata->>'doc_id' = ?",
				Long.class, docId);
		return total == null ? 0L : total;
	}

	private List<Document> listAll(IndexConfig indexConfig, int from, int size, String docId) {
		if (docId == null) {
			return jdbcTemplate.query("SELECT id, content, metadata FROM " + tableName(indexConfig)
					+ " ORDER BY id OFFSET ? LIMIT ?", this::mapDocument, from, size);
		}
		return jdbcTemplate.query("SELECT id, content, metadata FROM " + tableName(indexConfig)
				+ " WHERE metadata->>'doc_id' = ? ORDER BY id OFFSET ? LIMIT ?", this::mapDocument, docId, from, size);
	}

	private Document mapDocument(ResultSet rs, int rowNum) throws SQLException {
		Map<String, Object> metadata;
		try {
			String rawMetadata = rs.getString("metadata");
			metadata = rawMetadata == null || rawMetadata.isBlank()
					? new HashMap<>()
					: objectMapper.readValue(rawMetadata, METADATA_TYPE);
		}
		catch (Exception e) {
			throw new SQLException("Failed to read pgvector document metadata", e);
		}
		return Document.builder()
			.id(rs.getString("id"))
			.text(rs.getString("content"))
			.metadata(metadata)
			.build();
	}

	private String extractDocId(SearchRequest request) {
		if (!request.hasFilterExpression()) {
			return null;
		}
		try {
			org.springframework.ai.vectorstore.filter.converter.PrintFilterExpressionConverter converter =
					new org.springframework.ai.vectorstore.filter.converter.PrintFilterExpressionConverter();
			String spel = converter.convertExpression(request.getFilterExpression());
			Matcher matcher = DOC_ID_PATTERN.matcher(spel);
			return matcher.find() ? matcher.group(1) : null;
		}
		catch (Exception e) {
			log.debug("Could not extract doc_id from filter expression", e);
			return null;
		}
	}

	private String tableName(IndexConfig indexConfig) {
		return schemaName + "." + tableNameSuffix(indexConfig.getName());
	}

	private static String tableNameSuffix(String indexName) {
		return "rag_" + sanitizeName(indexName);
	}

	private static String sanitizeName(String name) {
		String sanitized = name == null ? "default" : name.toLowerCase().replaceAll("[^a-z0-9_]", "_");
		sanitized = sanitized.replaceAll("_+", "_").replaceAll("^_+|_+$", "");
		return sanitized.isBlank() ? "default" : sanitized;
	}

}
