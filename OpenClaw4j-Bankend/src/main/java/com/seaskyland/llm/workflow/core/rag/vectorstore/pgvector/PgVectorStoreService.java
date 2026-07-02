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

import static com.seaskyland.llm.workflow.core.rag.RagConstants.KEY_ENABLED;

import com.seaskyland.llm.workflow.core.model.llm.ModelFactory;
import com.seaskyland.llm.workflow.core.rag.DocumentChunkConverter;
import com.seaskyland.llm.workflow.core.rag.vectorstore.VectorStoreService;
import com.seaskyland.llm.workflow.runtime.domain.PagingList;
import com.seaskyland.llm.workflow.runtime.domain.knowledgebase.DocumentChunk;
import com.seaskyland.llm.workflow.runtime.domain.knowledgebase.IndexConfig;
import com.seaskyland.llm.workflow.runtime.enums.ErrorCode;
import com.seaskyland.llm.workflow.runtime.exception.BizException;
import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.regex.Pattern;
import java.util.stream.Collectors;
import lombok.extern.slf4j.Slf4j;
import org.springframework.ai.document.Document;
import org.springframework.ai.document.MetadataMode;
import org.springframework.ai.embedding.EmbeddingModel;
import org.springframework.ai.vectorstore.SearchRequest;
import org.springframework.ai.vectorstore.VectorStore;
import org.springframework.ai.vectorstore.pgvector.PgVectorStore;
import org.springframework.ai.vectorstore.pgvector.PgVectorStore.PgDistanceType;
import org.springframework.ai.vectorstore.pgvector.PgVectorStore.PgIdType;
import org.springframework.ai.vectorstore.pgvector.PgVectorStore.PgIndexType;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;
import org.springframework.util.CollectionUtils;
import org.springframework.util.StringUtils;

/** Vector store service backed by PostgreSQL pgvector. */
@Service
@Slf4j
@Qualifier("pgVectorStoreService")
public class PgVectorStoreService implements VectorStoreService {

  private static final Pattern VALID_IDENTIFIER = Pattern.compile("[A-Za-z_][A-Za-z0-9_]*");

  private final ModelFactory modelFactory;

  private final JdbcTemplate jdbcTemplate;

  private final String schemaName;

  private final boolean initializeSchema;

  private final PgIndexType indexType;

  private final PgDistanceType distanceType;

  private final Map<String, PgVectorStore> vectorStoreCache = new ConcurrentHashMap<>();

  public PgVectorStoreService(
      ModelFactory modelFactory,
      JdbcTemplate jdbcTemplate,
      @Value("${spring.ai.vectorstore.pgvector.schema-name:openclaw_rag}") String schemaName,
      @Value("${spring.ai.vectorstore.pgvector.initialize-schema:true}") boolean initializeSchema,
      @Value("${spring.ai.vectorstore.pgvector.index-type:HNSW}") String indexType,
      @Value("${spring.ai.vectorstore.pgvector.distance-type:COSINE_DISTANCE}") String distanceType) {
    this.modelFactory = modelFactory;
    this.jdbcTemplate = jdbcTemplate;
    this.schemaName = requireIdentifier(schemaName);
    this.initializeSchema = initializeSchema;
    this.indexType = PgIndexType.valueOf(indexType);
    this.distanceType = PgDistanceType.valueOf(distanceType);
  }

  @Override
  public void createIndex(IndexConfig indexConfig) {
    getPgVectorStore(indexConfig);
  }

  @Override
  public void deleteIndex(IndexConfig indexConfig) {
    String tableName = tableName(indexConfig);
    jdbcTemplate.execute("DROP TABLE IF EXISTS " + qualifiedTable(tableName));
    vectorStoreCache.remove(tableName);
    log.info("Deleted pgvector table {}", qualifiedTable(tableName));
  }

  @Override
  public VectorStore getVectorStore(IndexConfig indexConfig) {
    return getPgVectorStore(indexConfig);
  }

  @Override
  public PagingList<DocumentChunk> listDocumentChunks(
      IndexConfig indexConfig, SearchRequest searchRequest) {
    try {
      PgVectorStore vectorStore = getPgVectorStore(indexConfig);
      String tableName = tableName(indexConfig);
      String whereClause = whereClause(vectorStore, searchRequest);
      int from = Math.max(searchRequest.getFrom(), 0);
      int size = Math.max(searchRequest.getTopK(), 1);

      Long total =
          jdbcTemplate.queryForObject(
              "SELECT COUNT(*) FROM " + qualifiedTable(tableName) + whereClause, Long.class);
      List<DocumentChunk> chunks =
          jdbcTemplate
              .query(
                  "SELECT id, content, metadata FROM "
                      + qualifiedTable(tableName)
                      + whereClause
                      + " ORDER BY id LIMIT ? OFFSET ?",
                  (rs, rowNum) -> toDocumentChunk(rs),
                  size,
                  from)
              .stream()
              .collect(Collectors.toList());

      int current = (from / size) + 1;
      return new PagingList<>(current, size, total == null ? 0L : total, chunks);
    } catch (Exception e) {
      throw new BizException(ErrorCode.DOCUMENT_RETRIEVAL_ERROR.toError(), e);
    }
  }

  @Override
  public void updateDocumentChunks(IndexConfig indexConfig, List<DocumentChunk> chunks) {
    try {
      if (CollectionUtils.isEmpty(chunks)) {
        return;
      }
      List<Document> documents =
          chunks.stream().map(DocumentChunkConverter::toDocument).collect(Collectors.toList());
      getPgVectorStore(indexConfig).add(documents);
    } catch (Exception e) {
      throw new BizException(ErrorCode.UPDATE_DOCUMENT_CHUNK_ERROR.toError(), e);
    }
  }

  @Override
  public void updateDocumentChunkStatus(
      IndexConfig indexConfig, List<String> chunkIds, boolean enabled) {
    try {
      if (CollectionUtils.isEmpty(chunkIds)) {
        return;
      }
      getPgVectorStore(indexConfig);
      String tableName = tableName(indexConfig);
      String placeholders = String.join(",", Collections.nCopies(chunkIds.size(), "?"));
      List<Object> args = new ArrayList<>();
      args.add(enabled);
      args.addAll(chunkIds);
      jdbcTemplate.update(
          "UPDATE "
              + qualifiedTable(tableName)
              + " SET metadata = jsonb_set(COALESCE(metadata, '{}'::json)::jsonb,"
              + " '{"
              + KEY_ENABLED
              + "}', to_jsonb(CAST(? AS boolean)))::json"
              + " WHERE id IN ("
              + placeholders
              + ")",
          args.toArray());
    } catch (Exception e) {
      throw new BizException(ErrorCode.UPDATE_DOCUMENT_CHUNK_ERROR.toError(), e);
    }
  }

  private PgVectorStore getPgVectorStore(IndexConfig indexConfig) {
    String tableName = tableName(indexConfig);
    return vectorStoreCache.computeIfAbsent(
        tableName,
        name -> {
          EmbeddingModel embeddingModel =
              modelFactory.getEmbeddingModel(MetadataMode.EMBED, indexConfig);
          PgVectorStore vectorStore =
              PgVectorStore.builder(jdbcTemplate, embeddingModel)
                  .schemaName(schemaName)
                  .vectorTableName(name)
                  .idType(PgIdType.TEXT)
                  .initializeSchema(initializeSchema)
                  .indexType(indexType)
                  .distanceType(distanceType)
                  .build();
          vectorStore.afterPropertiesSet();
          return vectorStore;
        });
  }

  private DocumentChunk toDocumentChunk(ResultSet rs) throws SQLException {
    String metadataJson = rs.getString("metadata");
    Map<String, Object> metadata =
        StringUtils.hasText(metadataJson) ? JsonUtils.fromJsonToMap(metadataJson) : Map.of();
    Document document =
        Document.builder()
            .id(rs.getString("id"))
            .text(rs.getString("content"))
            .metadata(metadata)
            .build();
    return DocumentChunkConverter.toDocumentChunk(document);
  }

  private String whereClause(PgVectorStore vectorStore, SearchRequest searchRequest) {
    if (!searchRequest.hasFilterExpression()) {
      return "";
    }
    String expression =
        vectorStore.filterExpressionConverter.convertExpression(
            searchRequest.getFilterExpression());
    return StringUtils.hasText(expression) ? " WHERE " + expression : "";
  }

  private String tableName(IndexConfig indexConfig) {
    String rawName = indexConfig == null ? null : indexConfig.getName();
    if (!StringUtils.hasText(rawName)) {
      throw new IllegalArgumentException("Index name must be provided");
    }
    String tableName = "kb_" + rawName.trim().replaceAll("[^A-Za-z0-9_]", "_").toLowerCase();
    if (tableName.length() > 63) {
      tableName = tableName.substring(0, 63);
    }
    return requireIdentifier(tableName);
  }

  private String qualifiedTable(String tableName) {
    return schemaName + "." + requireIdentifier(tableName);
  }

  private String requireIdentifier(String identifier) {
    if (!StringUtils.hasText(identifier) || !VALID_IDENTIFIER.matcher(identifier).matches()) {
      throw new IllegalArgumentException("Invalid PostgreSQL identifier: " + identifier);
    }
    return identifier;
  }
}
