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

import com.seaskyland.llm.workflow.core.model.embedding.DefaultBatchingStrategy;
import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.ai.document.Document;
import org.springframework.ai.embedding.EmbeddingModel;
import org.springframework.ai.embedding.EmbeddingOptions;
import org.springframework.ai.observation.conventions.VectorStoreProvider;
import org.springframework.ai.observation.conventions.VectorStoreSimilarityMetric;
import org.springframework.ai.vectorstore.AbstractVectorStoreBuilder;
import org.springframework.ai.vectorstore.SearchRequest;
import org.springframework.ai.vectorstore.filter.Filter;
import org.springframework.ai.vectorstore.filter.FilterExpressionConverter;
import org.springframework.ai.vectorstore.filter.converter.PrintFilterExpressionConverter;
import org.springframework.ai.vectorstore.observation.AbstractObservationVectorStore;
import org.springframework.ai.vectorstore.observation.VectorStoreObservationContext;
import org.springframework.expression.ExpressionParser;
import org.springframework.expression.spel.standard.SpelExpressionParser;
import org.springframework.expression.spel.support.StandardEvaluationContext;
import org.sqlite.SQLiteConfig;

import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.sql.*;
import java.util.*;
import java.util.function.Predicate;

import static com.seaskyland.llm.workflow.core.rag.RagConstants.*;

/**
 * SQLite-backed vector store implementation that uses the
 * <a href="https://github.com/asg017/sqlite-vec">sqlite-vec</a> extension for
 * approximate nearest-neighbour (ANN) similarity search.
 *
 * <p><b>Schema</b><br>
 * For every index (knowledge-base) two tables are created inside the same SQLite file:
 * <ul>
 *   <li>{@code vec_chunk_{indexName}} – regular table holding text content and metadata.</li>
 *   <li>{@code vec_embedding_{indexName}} – {@code vec0} virtual table holding the float32
 *       embeddings (requires the sqlite-vec extension to be loaded).</li>
 * </ul>
 *
 * <p><b>Similarity search</b><br>
 * Uses sqlite-vec ANN syntax:
 * <pre>{@code
 * SELECT chunk_id, distance
 * FROM   vec_embedding_{name}
 * WHERE  embedding MATCH ?
 *   AND  k = ?
 * ORDER BY distance
 * }</pre>
 *
 * The {@code distance} returned by sqlite-vec is the L2 (Euclidean) distance between
 * float32 vectors.  For embedding models that produce unit-normalised vectors the
 * conversion to cosine similarity is: {@code score = 1 − distance² / 2}.
 *
 * <p><b>Extension loading</b><br>
 * Each JDBC connection is created via {@link DriverManager} with
 * {@link SQLiteConfig#enableLoadExtension(boolean) extension loading enabled}, and
 * then immediately executes {@code SELECT load_extension('<path>')} if a non-blank
 * extension path has been supplied through
 * {@link SqliteVectorStoreBuilder#vecExtensionPath(String)}.
 *
 * @since 1.0.0.3
 */
public class SqliteVectorStore extends AbstractObservationVectorStore {

	private static final Logger log = LoggerFactory.getLogger(SqliteVectorStore.class);

	/** Name of the index (used as table-name suffix). */
	private final String indexName;

	/** Vector dimension — must match the embedding model's output. */
	private final int dimension;

	/** JDBC URL of the SQLite database file, e.g. {@code jdbc:sqlite:./data/openclaw.db}. */
	private final String dbUrl;

	/**
	 * Absolute path to the sqlite-vec native extension file (.dll/.so/.dylib).
	 * May be {@code null} or blank if the extension is pre-loaded through another mechanism.
	 */
	private final String vecExtensionPath;

	/** Used to evaluate Spring AI filter expressions against document metadata. */
	private final ExpressionParser expressionParser = new SpelExpressionParser();

	/** Converts Spring AI FilterExpression into SpEL strings. */
	private final FilterExpressionConverter filterExpressionConverter =
			new PrintFilterExpressionConverter();

	// -------------------------------------------------------------------------
	// Constructor (via builder)
	// -------------------------------------------------------------------------

	protected SqliteVectorStore(SqliteVectorStoreBuilder builder) {
		super(builder);
		this.indexName      = sanitizeName(builder.indexName);
		this.dimension      = builder.dimension;
		this.dbUrl          = builder.dbUrl;
		this.vecExtensionPath = builder.vecExtensionPath;
		initSchema();
	}

	// -------------------------------------------------------------------------
	// Public builder factory
	// -------------------------------------------------------------------------

	public static SqliteVectorStoreBuilder builder(EmbeddingModel embeddingModel) {
		return new SqliteVectorStoreBuilder(embeddingModel);
	}

	// -------------------------------------------------------------------------
	// AbstractObservationVectorStore overrides
	// -------------------------------------------------------------------------

	@Override
	public void doAdd(List<Document> documents) {
		if (documents == null || documents.isEmpty()) {
			return;
		}

		// Batch-generate embeddings for all documents at once
		List<float[]> embeddings = this.embeddingModel.embed(
				documents,
				EmbeddingOptions.builder().build(),
				new DefaultBatchingStrategy());

		String metaSql = "INSERT OR REPLACE INTO " + metaTable()
				+ " (chunk_id, doc_id, doc_name, title, page_number, workspace_id, enabled, content)"
				+ " VALUES (?,?,?,?,?,?,?,?)";
		String vecSql = "INSERT OR REPLACE INTO " + vecTable()
				+ " (chunk_id, embedding) VALUES (?,?)";

		try (Connection conn = openConnection()) {
			conn.setAutoCommit(false);
			try (PreparedStatement metaPs = conn.prepareStatement(metaSql);
				 PreparedStatement vecPs  = conn.prepareStatement(vecSql)) {

				for (int i = 0; i < documents.size(); i++) {
					Document doc = documents.get(i);
					Map<String, Object> meta = doc.getMetadata();

					// Metadata table row
					metaPs.setString(1, doc.getId());
					metaPs.setString(2, toString(meta.get(KEY_DOC_ID)));
					metaPs.setString(3, toString(meta.get(KEY_DOC_NAME)));
					metaPs.setString(4, toString(meta.get(KEY_TITLE)));
					Object pageNum = meta.get("page_number");
					if (pageNum != null) {
						metaPs.setInt(5, ((Number) pageNum).intValue());
					}
					else {
						metaPs.setNull(5, Types.INTEGER);
					}
					metaPs.setString(6, toString(meta.get(KEY_WORKSPACE_ID)));
					Object enabled = meta.get(KEY_ENABLED);
					metaPs.setInt(7, Boolean.TRUE.equals(enabled) ? 1 : 0);
					metaPs.setString(8, doc.getText());
					metaPs.addBatch();

					// vec0 virtual table row — embedding serialised as little-endian float32 blob
					vecPs.setString(1, doc.getId());
					vecPs.setBytes(2, floatArrayToBytes(embeddings.get(i)));
					vecPs.addBatch();
				}

				metaPs.executeBatch();
				vecPs.executeBatch();
			}
			conn.commit();
		}
		catch (SQLException e) {
			throw new RuntimeException(
					"Failed to add documents to SQLite vector store index: " + indexName, e);
		}
	}

	@Override
	public void doDelete(List<String> idList) {
		if (idList == null || idList.isEmpty()) {
			return;
		}
		String placeholders = String.join(",", Collections.nCopies(idList.size(), "?"));
		String delMeta = "DELETE FROM " + metaTable() + " WHERE chunk_id IN (" + placeholders + ")";
		String delVec  = "DELETE FROM " + vecTable()  + " WHERE chunk_id IN (" + placeholders + ")";

		try (Connection conn = openConnection()) {
			conn.setAutoCommit(false);
			try (PreparedStatement ps1 = conn.prepareStatement(delMeta);
				 PreparedStatement ps2 = conn.prepareStatement(delVec)) {
				for (int i = 0; i < idList.size(); i++) {
					ps1.setString(i + 1, idList.get(i));
					ps2.setString(i + 1, idList.get(i));
				}
				ps1.executeUpdate();
				ps2.executeUpdate();
			}
			conn.commit();
		}
		catch (SQLException e) {
			throw new RuntimeException(
					"Failed to delete documents from SQLite vector store index: " + indexName, e);
		}
	}

	@Override
	public void doDelete(Filter.Expression filterExpression) {
		// Resolve matching IDs via the metadata table and then delegate to id-based delete
		List<Document> matched = fetchAllMetadata(filterExpression);
		List<String> ids = matched.stream().map(Document::getId).toList();
		if (!ids.isEmpty()) {
			doDelete(ids);
		}
	}

	/**
	 * Performs ANN similarity search using sqlite-vec syntax:
	 * <pre>{@code WHERE embedding MATCH ? AND k = ? ORDER BY distance}</pre>
	 *
	 * <p>The raw L2 distance from sqlite-vec is converted to a cosine-similarity score
	 * assuming unit-normalised vectors: {@code score = 1 − distance² / 2}.
	 * Post-match metadata filters (Spring AI {@link Filter.Expression}) are applied
	 * in-memory after retrieving the top-k candidates.
	 */
	@Override
	public List<Document> doSimilaritySearch(SearchRequest request) {
		String queryText = request.getQuery();
		int topK = request.getTopK();
		double threshold = request.getSimilarityThreshold();

		// Embed the query string
		float[] queryEmbedding = this.embeddingModel.embed(queryText);

		/*
		 * sqlite-vec ANN query:
		 *   SELECT chunk_id, distance
		 *   FROM   vec_embedding_{name}
		 *   WHERE  embedding MATCH ?      -- query vector (little-endian float32 blob)
		 *     AND  k = ?                  -- number of nearest neighbours to return
		 *   ORDER BY distance
		 */
		String sql = "SELECT v.chunk_id, v.distance,"
				+ " m.doc_id, m.doc_name, m.title, m.page_number,"
				+ " m.workspace_id, m.enabled, m.content"
				+ " FROM " + vecTable() + " v"
				+ " JOIN " + metaTable() + " m ON v.chunk_id = m.chunk_id"
				+ " WHERE v.embedding MATCH ?"
				+ "   AND k = ?"
				+ " ORDER BY v.distance";

		List<Document> results = new ArrayList<>();
		Predicate<Document> filterPredicate = buildFilterPredicate(request);

		try (Connection conn = openConnection();
			 PreparedStatement ps = conn.prepareStatement(sql)) {

			ps.setBytes(1, floatArrayToBytes(queryEmbedding));
			ps.setInt(2, topK);

			try (ResultSet rs = ps.executeQuery()) {
				while (rs.next()) {
					double l2Distance = rs.getDouble("distance");
					// Convert L2 distance to cosine similarity (valid for unit-normalised vectors)
					double score = 1.0 - (l2Distance * l2Distance) / 2.0;

					if (score < threshold) {
						continue;
					}

					Document doc = buildDocument(rs, score);

					if (filterPredicate.test(doc)) {
						results.add(doc);
					}
				}
			}
		}
		catch (SQLException e) {
			throw new RuntimeException(
					"Failed to perform similarity search on SQLite vector store index: " + indexName, e);
		}

		return results;
	}

	@Override
	public VectorStoreObservationContext.Builder createObservationContextBuilder(String operationName) {
		return VectorStoreObservationContext.builder(VectorStoreProvider.SIMPLE.value(), operationName)
				.dimensions(this.embeddingModel.dimensions())
				.collectionName(this.indexName)
				.similarityMetric(VectorStoreSimilarityMetric.COSINE.value());
	}

	// -------------------------------------------------------------------------
	// Additional operations used by SqliteVectorStoreService
	// -------------------------------------------------------------------------

	/**
	 * Returns a paginated list of documents from the metadata table.
	 * Optionally filtered by {@code doc_id}.
	 * @param offset zero-based row offset
	 * @param limit  maximum number of rows to return
	 * @param docId  when non-blank, restricts results to chunks of this document
	 * @return list of documents (without score)
	 */
	public List<Document> listAll(int offset, int limit, String docId) {
		String sql = "SELECT chunk_id, doc_id, doc_name, title, page_number,"
				+ " workspace_id, enabled, content"
				+ " FROM " + metaTable()
				+ (StringUtils.isNotBlank(docId) ? " WHERE doc_id = ?" : "")
				+ " ORDER BY rowid"
				+ " LIMIT ? OFFSET ?";

		List<Document> results = new ArrayList<>();
		try (Connection conn = openConnection();
			 PreparedStatement ps = conn.prepareStatement(sql)) {

			int idx = 1;
			if (StringUtils.isNotBlank(docId)) {
				ps.setString(idx++, docId);
			}
			ps.setInt(idx++, limit);
			ps.setInt(idx,   offset);

			try (ResultSet rs = ps.executeQuery()) {
				while (rs.next()) {
					results.add(buildDocument(rs, null));
				}
			}
		}
		catch (SQLException e) {
			throw new RuntimeException(
					"Failed to list documents from SQLite vector store index: " + indexName, e);
		}
		return results;
	}

	/**
	 * Returns the total number of chunks in the metadata table,
	 * optionally scoped to a single document.
	 * @param docId when non-blank, counts only chunks belonging to this document
	 * @return total row count
	 */
	public long countAll(String docId) {
		String sql = "SELECT COUNT(*) FROM " + metaTable()
				+ (StringUtils.isNotBlank(docId) ? " WHERE doc_id = ?" : "");

		try (Connection conn = openConnection();
			 PreparedStatement ps = conn.prepareStatement(sql)) {

			if (StringUtils.isNotBlank(docId)) {
				ps.setString(1, docId);
			}
			try (ResultSet rs = ps.executeQuery()) {
				return rs.next() ? rs.getLong(1) : 0L;
			}
		}
		catch (SQLException e) {
			throw new RuntimeException(
					"Failed to count documents in SQLite vector store index: " + indexName, e);
		}
	}

	/**
	 * Updates the {@code enabled} flag for a list of chunk IDs directly in the
	 * metadata table (no re-embedding required).
	 * @param chunkIds list of chunk IDs to update
	 * @param enabled  new enabled state
	 */
	public void updateEnabled(List<String> chunkIds, boolean enabled) {
		if (chunkIds == null || chunkIds.isEmpty()) {
			return;
		}
		String placeholders = String.join(",", Collections.nCopies(chunkIds.size(), "?"));
		String sql = "UPDATE " + metaTable()
				+ " SET enabled = ?"
				+ " WHERE chunk_id IN (" + placeholders + ")";

		try (Connection conn = openConnection();
			 PreparedStatement ps = conn.prepareStatement(sql)) {

			ps.setInt(1, enabled ? 1 : 0);
			for (int i = 0; i < chunkIds.size(); i++) {
				ps.setString(i + 2, chunkIds.get(i));
			}
			ps.executeUpdate();
		}
		catch (SQLException e) {
			throw new RuntimeException(
					"Failed to update enabled status in SQLite vector store index: " + indexName, e);
		}
	}

	/**
	 * Drops both tables created by this index — used by
	 * {@link SqliteVectorStoreService#deleteIndex}.
	 */
	public void dropTables() {
		try (Connection conn = openConnection();
			 Statement stmt = conn.createStatement()) {
			stmt.execute("DROP TABLE IF EXISTS " + metaTable());
			stmt.execute("DROP TABLE IF EXISTS " + vecTable());
			log.info("Dropped SQLite vector store tables for index: {}", indexName);
		}
		catch (SQLException e) {
			throw new RuntimeException(
					"Failed to drop tables for SQLite vector store index: " + indexName, e);
		}
	}

	// -------------------------------------------------------------------------
	// Private helpers
	// -------------------------------------------------------------------------

	/** Table name that stores chunk text and metadata. */
	private String metaTable() {
		return "vec_chunk_" + indexName;
	}

	/** Table name that stores float32 embeddings (vec0 virtual table). */
	private String vecTable() {
		return "vec_embedding_" + indexName;
	}

	/**
	 * Creates the metadata table and the vec0 virtual table if they do not already exist.
	 * The vec0 virtual table requires the sqlite-vec extension to be loaded on the
	 * connection; if the extension is absent the statement will throw and the error is
	 * re-thrown to the caller.
	 */
	private void initSchema() {
		try (Connection conn = openConnection();
			 Statement stmt = conn.createStatement()) {

			// Regular metadata table
			stmt.execute(
				"CREATE TABLE IF NOT EXISTS " + metaTable() + " ("
				+ "chunk_id     TEXT    PRIMARY KEY,"
				+ "doc_id       TEXT,"
				+ "doc_name     TEXT,"
				+ "title        TEXT,"
				+ "page_number  INTEGER,"
				+ "workspace_id TEXT,"
				+ "enabled      INTEGER NOT NULL DEFAULT 1,"
				+ "content      TEXT"
				+ ")"
			);
			stmt.execute(
				"CREATE INDEX IF NOT EXISTS idx_" + indexName + "_doc_id"
				+ " ON " + metaTable() + " (doc_id)"
			);

			/*
			 * sqlite-vec virtual table — ANN index over float32 vectors.
			 * Syntax: CREATE VIRTUAL TABLE ... USING vec0(
			 *           chunk_id TEXT PRIMARY KEY,
			 *           embedding float[<dim>]
			 *         )
			 */
			stmt.execute(
				"CREATE VIRTUAL TABLE IF NOT EXISTS " + vecTable() + " USING vec0("
				+ "chunk_id  TEXT PRIMARY KEY,"
				+ "embedding float[" + dimension + "]"
				+ ")"
			);

			log.info("SQLite vector store schema initialised for index '{}' (dim={})",
					indexName, dimension);
		}
		catch (SQLException e) {
			throw new RuntimeException(
					"Failed to initialise schema for SQLite vector store index: " + indexName
					+ ". Ensure the sqlite-vec extension is available and the path is configured"
					+ " via spring.ai.alibaba.studio.sqlite-vec-extension-path", e);
		}
	}

	/**
	 * Opens a new JDBC connection with sqlite-vec extension loading enabled.
	 * The extension is loaded on every new connection because SQLite extensions
	 * are not persisted between connections.
	 */
	private Connection openConnection() throws SQLException {
		SQLiteConfig config = new SQLiteConfig();

		if (StringUtils.isNotBlank(vecExtensionPath)) {
			// Enable C-level extension loading (required before load_extension() works)
			config.enableLoadExtension(true);
		}

		Connection conn = DriverManager.getConnection(dbUrl, config.toProperties());

		if (StringUtils.isNotBlank(vecExtensionPath)) {
			try (Statement stmt = conn.createStatement()) {
				// Load the sqlite-vec shared library
				stmt.execute("SELECT load_extension('" + vecExtensionPath.replace("'", "''") + "')");
			}
			catch (SQLException e) {
				conn.close();
				throw new SQLException(
						"Failed to load sqlite-vec extension from: " + vecExtensionPath
						+ ". Verify the file exists and is compatible with your platform.", e);
			}
		}

		return conn;
	}

	/**
	 * Fetches all documents from the metadata table that satisfy the given
	 * {@link Filter.Expression} (evaluated in-memory via SpEL).
	 */
	private List<Document> fetchAllMetadata(Filter.Expression filterExpression) {
		String sql = "SELECT chunk_id, doc_id, doc_name, title, page_number,"
				+ " workspace_id, enabled, content"
				+ " FROM " + metaTable();

		List<Document> all = new ArrayList<>();
		try (Connection conn = openConnection();
			 Statement stmt = conn.createStatement();
			 ResultSet rs = stmt.executeQuery(sql)) {

			while (rs.next()) {
				all.add(buildDocument(rs, null));
			}
		}
		catch (SQLException e) {
			throw new RuntimeException(
					"Failed to fetch metadata from SQLite vector store index: " + indexName, e);
		}

		if (filterExpression == null) {
			return all;
		}

		String spelExpr = filterExpressionConverter.convertExpression(filterExpression);
		ExpressionParser parser = new SpelExpressionParser();
		return all.stream().filter(doc -> {
			StandardEvaluationContext ctx = new StandardEvaluationContext();
			ctx.setVariable("metadata", doc.getMetadata());
			return Boolean.TRUE.equals(parser.parseExpression(spelExpr).getValue(ctx, Boolean.class));
		}).toList();
	}

	/** Builds a predicate from the {@link SearchRequest}'s filter expression for post-match filtering. */
	private Predicate<Document> buildFilterPredicate(SearchRequest request) {
		if (!request.hasFilterExpression()) {
			return doc -> true;
		}
		String spelExpr = filterExpressionConverter.convertExpression(request.getFilterExpression());
		return doc -> {
			StandardEvaluationContext ctx = new StandardEvaluationContext();
			ctx.setVariable("metadata", doc.getMetadata());
			return Boolean.TRUE.equals(
					expressionParser.parseExpression(spelExpr).getValue(ctx, Boolean.class));
		};
	}

	/** Builds a {@link Document} from the current row of a {@link ResultSet}. */
	private Document buildDocument(ResultSet rs, Double score) throws SQLException {
		Map<String, Object> meta = new LinkedHashMap<>();
		meta.put(KEY_DOC_ID,       rs.getString("doc_id"));
		meta.put(KEY_DOC_NAME,     rs.getString("doc_name"));
		meta.put(KEY_TITLE,        rs.getString("title"));
		meta.put("page_number",    rs.getInt("page_number"));
		meta.put(KEY_WORKSPACE_ID, rs.getString("workspace_id"));
		meta.put(KEY_ENABLED,      rs.getInt("enabled") == 1);

		Document.Builder builder = Document.builder()
				.id(rs.getString("chunk_id"))
				.text(rs.getString("content"))
				.metadata(meta);

		if (score != null) {
			builder.score(score);
		}

		return builder.build();
	}

	/**
	 * Serialises a {@code float[]} into a little-endian IEEE 754 byte array,
	 * which is the format expected by sqlite-vec for float32 vectors.
	 */
	private static byte[] floatArrayToBytes(float[] values) {
		ByteBuffer buf = ByteBuffer.allocate(values.length * Float.BYTES);
		buf.order(ByteOrder.LITTLE_ENDIAN);
		for (float v : values) {
			buf.putFloat(v);
		}
		return buf.array();
	}

	/** Replaces hyphens with underscores so the name is a valid SQL identifier. */
	private static String sanitizeName(String name) {
		return name == null ? "default" : name.replace("-", "_").replace(" ", "_");
	}

	private static String toString(Object obj) {
		return obj == null ? null : obj.toString();
	}

	// -------------------------------------------------------------------------
	// Builder
	// -------------------------------------------------------------------------

	/**
	 * Builder for {@link SqliteVectorStore}.
	 */
	public static final class SqliteVectorStoreBuilder
			extends AbstractVectorStoreBuilder<SqliteVectorStoreBuilder> {

		private String indexName;

		private int dimension = DEFAULT_DIMENSION;

		private String dbUrl;

		private String vecExtensionPath;

		private SqliteVectorStoreBuilder(EmbeddingModel embeddingModel) {
			super(embeddingModel);
		}

		/** Sets the index / table-name suffix. */
		public SqliteVectorStoreBuilder indexName(String indexName) {
			this.indexName = indexName;
			return this;
		}

		/** Sets the vector dimension (must match the embedding model). */
		public SqliteVectorStoreBuilder dimension(int dimension) {
			this.dimension = dimension;
			return this;
		}

		/** Sets the JDBC URL of the SQLite database. */
		public SqliteVectorStoreBuilder dbUrl(String dbUrl) {
			this.dbUrl = dbUrl;
			return this;
		}

		/** Sets the absolute path to the sqlite-vec native extension file. */
		public SqliteVectorStoreBuilder vecExtensionPath(String vecExtensionPath) {
			this.vecExtensionPath = vecExtensionPath;
			return this;
		}

		@Override
		public SqliteVectorStore build() {
			Objects.requireNonNull(indexName, "indexName must not be null");
			Objects.requireNonNull(dbUrl,     "dbUrl must not be null");
			return new SqliteVectorStore(this);
		}

	}

}
