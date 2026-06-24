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

package com.seaskyland.llm.workflow.core.rag.vectorstore.sqlite;

import com.seaskyland.llm.workflow.core.config.StudioProperties;
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
import org.apache.commons.lang3.StringUtils;
import org.springframework.ai.document.Document;
import org.springframework.ai.document.MetadataMode;
import org.springframework.ai.embedding.EmbeddingModel;
import org.springframework.ai.vectorstore.SearchRequest;
import org.springframework.ai.vectorstore.VectorStore;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Collectors;

import static com.seaskyland.llm.workflow.core.rag.RagConstants.DEFAULT_DIMENSION;

/**
 * SQLite-backed vector store service using the
 * <a href="https://github.com/asg017/sqlite-vec">sqlite-vec</a> extension for
 * ANN (approximate nearest-neighbour) similarity search.
 *
 * <p>Each knowledge-base index maps to two SQLite tables inside the application's
 * existing database file:
 * <ul>
 *   <li>{@code vec_chunk_{name}} – plain table for text content and metadata.</li>
 *   <li>{@code vec_embedding_{name}} – {@code vec0} virtual table for vector search.</li>
 * </ul>
 *
 * <h3>Extension loading priority</h3>
 * <ol>
 *   <li>If {@code spring.ai.alibaba.studio.sqlite-vec-extension-path} is set and the
 *       file exists, it is used directly.</li>
 *   <li>Otherwise the bundled extension inside the classpath at
 *       {@code sqlite/vec0.dll} (Windows), {@code sqlite/vec0.so} (Linux) or
 *       {@code sqlite/vec0.dylib} (macOS) is extracted to a temporary directory at
 *       startup and the resulting path is used automatically.</li>
 * </ol>
 *
 * <h3>Minimal configuration</h3>
 * <pre>{@code
 * spring:
 *   ai:
 *     alibaba:
 *       studio:
 *         vector-store-type: sqlite
 * }</pre>
 * Just set {@code vector-store-type: sqlite} — the bundled {@code vec0.dll / .so / .dylib}
 * under {@code src/main/resources/sqlite/} is extracted automatically.
 *
 * @since 1.0.0.3
 */
@Service
@Slf4j
@Qualifier("sqliteVectorStoreService")
public class SqliteVectorStoreService implements VectorStoreService {

	/** Classpath directory that contains the bundled sqlite-vec native libraries. */
	private static final String CLASSPATH_EXTENSION_DIR = "sqlite";

	/** Pattern for extracting a literal doc_id from a simple SpEL equality expression. */
	private static final Pattern DOC_ID_PATTERN = Pattern.compile(
			"#metadata\\[.doc_id.\\]\\s*==\\s*['\"]([^'\"]+)['\"]");

	/** Factory for creating embedding models per index. */
	private final ModelFactory modelFactory;

	/** JDBC URL of the SQLite database (e.g. {@code jdbc:sqlite:./data/openclaw.db}). */
	private final String dbUrl;

	/**
	 * Resolved absolute filesystem path to the sqlite-vec native extension.
	 * Determined once at construction time via {@link #resolveExtensionPath}.
	 */
	private final String vecExtensionPath;

	/** Per-index cache of {@link SqliteVectorStore} instances. */
	private final Map<String, SqliteVectorStore> storeCache = new ConcurrentHashMap<>();

	public SqliteVectorStoreService(
			ModelFactory modelFactory,
			@Value("${spring.datasource.url}") String dbUrl,
			StudioProperties studioProperties) {
		this.modelFactory     = modelFactory;
		this.dbUrl            = dbUrl;
		this.vecExtensionPath = resolveExtensionPath(studioProperties.getSqliteVecExtensionPath());
	}

	// -------------------------------------------------------------------------
	// VectorStoreService implementation
	// -------------------------------------------------------------------------

	/**
	 * Creates the underlying SQLite tables for the given index.
	 * If the tables already exist this is a no-op.
	 * @param indexConfig Index configuration (name + embedding model)
	 */
	@Override
	public void createIndex(IndexConfig indexConfig) {
		getOrCreateStore(indexConfig);
		log.info("SQLite vector store index '{}' is ready", indexConfig.getName());
	}

	/**
	 * Drops the SQLite tables associated with the given index.
	 * @param indexConfig Index configuration
	 */
	@Override
	public void deleteIndex(IndexConfig indexConfig) {
		SqliteVectorStore store = storeCache.remove(indexConfig.getName());
		if (store != null) {
			store.dropTables();
		}
		log.info("SQLite vector store index '{}' deleted", indexConfig.getName());
	}

	/**
	 * Returns the {@link VectorStore} instance for the given index,
	 * creating it (and its tables) on first access.
	 * @param indexConfig Index configuration
	 * @return configured vector store
	 */
	@Override
	public VectorStore getVectorStore(IndexConfig indexConfig) {
		return getOrCreateStore(indexConfig);
	}

	/**
	 * Lists document chunks with pagination.  If the search request carries a
	 * {@code doc_id} filter it is translated to a SQL {@code WHERE doc_id = ?}
	 * predicate; otherwise all chunks are returned (paginated).
	 * @param indexConfig   Index configuration
	 * @param searchRequest Pagination parameters
	 * @return Paginated list of document chunks
	 */
	@Override
	public PagingList<DocumentChunk> listDocumentChunks(IndexConfig indexConfig,
			SearchRequest searchRequest) {
		try {
			SqliteVectorStore store = getOrCreateStore(indexConfig);

			int from = searchRequest.getFrom();
			int size = searchRequest.getTopK();
			String docId = extractDocId(searchRequest);

			long total = store.countAll(docId);
			List<Document> docs = store.listAll(from, size, docId);

			List<DocumentChunk> chunks = docs.stream()
					.map(DocumentChunkConverter::toDocumentChunk)
					.collect(Collectors.toList());

			int current = size > 0 ? (from / size) + 1 : 1;
			return new PagingList<>(current, size, total, chunks);
		}
		catch (Exception e) {
			throw new BizException(ErrorCode.DOCUMENT_RETRIEVAL_ERROR.toError(), e);
		}
	}

	/**
	 * Re-embeds and upserts the supplied document chunks.
	 * @param indexConfig Index configuration
	 * @param chunks      Chunks to upsert
	 */
	@Override
	public void updateDocumentChunks(IndexConfig indexConfig, List<DocumentChunk> chunks) {
		try {
			SqliteVectorStore store = getOrCreateStore(indexConfig);
			List<Document> documents = chunks.stream()
					.map(DocumentChunkConverter::toDocument)
					.collect(Collectors.toList());
			store.add(documents);
		}
		catch (Exception e) {
			throw new BizException(ErrorCode.UPDATE_DOCUMENT_CHUNK_ERROR.toError(), e);
		}
	}

	/**
	 * Updates the {@code enabled} flag of the specified chunks directly in the
	 * metadata table (no re-embedding required).
	 * @param indexConfig Index configuration
	 * @param chunkIds    IDs of the chunks to update
	 * @param enabled     New enabled state
	 */
	@Override
	public void updateDocumentChunkStatus(IndexConfig indexConfig, List<String> chunkIds,
			boolean enabled) {
		try {
			SqliteVectorStore store = getOrCreateStore(indexConfig);
			store.updateEnabled(chunkIds, enabled);
		}
		catch (Exception e) {
			throw new BizException(ErrorCode.UPDATE_DOCUMENT_CHUNK_ERROR.toError(), e);
		}
	}

	// -------------------------------------------------------------------------
	// Private helpers
	// -------------------------------------------------------------------------

	/**
	 * Determines the sqlite-vec native extension path using the following priority:
	 * <ol>
	 *   <li>If {@code configured} is non-blank and points to an existing file → use it.</li>
	 *   <li>Try to extract the bundled classpath resource
	 *       {@code sqlite/vec0.{dll|so|dylib}} to a JVM temp directory → use that.</li>
	 *   <li>Log a warning and return {@code null} (schema-only mode).</li>
	 * </ol>
	 */
	private static String resolveExtensionPath(String configured) {
		// 1. Explicitly configured path takes highest priority
		if (StringUtils.isNotBlank(configured)) {
			File file = new File(configured);
			if (file.exists()) {
				log.info("Using configured sqlite-vec extension: {}", configured);
				return configured;
			}
			log.warn("Configured sqlite-vec extension not found at '{}', "
					+ "falling back to bundled classpath resource.", configured);
		}

		// 2. Try to extract the bundled extension from classpath
		String extracted = extractBundledExtension();
		if (extracted != null) {
			return extracted;
		}

		// 3. No extension available
		log.warn("sqlite-vec native extension could not be found. "
				+ "Place vec0.dll / vec0.so / vec0.dylib under src/main/resources/sqlite/ "
				+ "or set spring.ai.alibaba.studio.sqlite-vec-extension-path. "
				+ "ANN queries will fail at runtime.");
		return null;
	}

	/**
	 * Detects the platform-specific filename for the sqlite-vec extension
	 * ({@code vec0.dll} / {@code vec0.so} / {@code vec0.dylib}), loads it from
	 * the classpath directory {@code sqlite/}, copies it to a temporary file, and
	 * returns the absolute path of that file.
	 *
	 * <p>The temporary file is marked {@link File#deleteOnExit()} so it is removed
	 * when the JVM shuts down.
	 *
	 * @return absolute path of the extracted extension, or {@code null} if not found
	 */
	private static String extractBundledExtension() {
		String filename = platformExtensionFilename();
		String resourcePath = CLASSPATH_EXTENSION_DIR + "/" + filename;

		try (InputStream in = SqliteVectorStoreService.class
				.getClassLoader().getResourceAsStream(resourcePath)) {

			if (in == null) {
				log.debug("Bundled sqlite-vec extension not found on classpath at: {}", resourcePath);
				return null;
			}

			// Copy to a stable temp directory so SQLite can dlopen() it
			Path tempDir  = Files.createTempDirectory("openclaw-sqlite-vec");
			Path tempFile = tempDir.resolve(filename);
			Files.copy(in, tempFile, StandardCopyOption.REPLACE_EXISTING);

			// Clean up on JVM exit
			tempFile.toFile().deleteOnExit();
			tempDir.toFile().deleteOnExit();

			String path = tempFile.toAbsolutePath().toString();
			log.info("Extracted bundled sqlite-vec extension ({}) to: {}", filename, path);
			return path;
		}
		catch (IOException e) {
			log.warn("Failed to extract bundled sqlite-vec extension from classpath: {}",
					e.getMessage(), e);
			return null;
		}
	}

	/**
	 * Returns the platform-specific filename of the sqlite-vec native extension:
	 * <ul>
	 *   <li>Windows → {@code vec0.dll}</li>
	 *   <li>macOS   → {@code vec0.dylib}</li>
	 *   <li>Linux   → {@code vec0.so}</li>
	 * </ul>
	 */
	private static String platformExtensionFilename() {
		String os = System.getProperty("os.name", "").toLowerCase();
		if (os.contains("win")) {
			return "vec0.dll";
		}
		if (os.contains("mac")) {
			return "vec0.dylib";
		}
		return "vec0.so";
	}

	/**
	 * Returns a cached {@link SqliteVectorStore} for the given index,
	 * creating and initialising one if absent.
	 */
	private SqliteVectorStore getOrCreateStore(IndexConfig indexConfig) {
		return storeCache.computeIfAbsent(indexConfig.getName(), name -> {
			EmbeddingModel embeddingModel =
					modelFactory.getEmbeddingModel(MetadataMode.EMBED, indexConfig);
			int dimension = EmbeddingModelDimension.getDimension(
					indexConfig.getEmbeddingModel(), DEFAULT_DIMENSION);

			return SqliteVectorStore.builder(embeddingModel)
					.indexName(name)
					.dimension(dimension)
					.dbUrl(dbUrl)
					.vecExtensionPath(vecExtensionPath)
					.batchingStrategy(new DefaultBatchingStrategy())
					.build();
		});
	}

	/**
	 * Attempts to extract a {@code doc_id} literal from the filter expression of a
	 * {@link SearchRequest}.  Returns {@code null} for complex or absent expressions.
	 */
	private String extractDocId(SearchRequest request) {
		if (!request.hasFilterExpression()) {
			return null;
		}
		try {
			org.springframework.ai.vectorstore.filter.converter
					.PrintFilterExpressionConverter conv =
					new org.springframework.ai.vectorstore.filter.converter
							.PrintFilterExpressionConverter();
			String spel = conv.convertExpression(request.getFilterExpression());
			Matcher m = DOC_ID_PATTERN.matcher(spel);
			return m.find() ? m.group(1) : null;
		}
		catch (Exception e) {
			log.debug("Could not extract doc_id from filter expression", e);
			return null;
		}
	}

}
