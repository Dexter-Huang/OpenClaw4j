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

package com.seaskyland.llm.workflow.core.rag.vectorstore.simple;

import com.seaskyland.llm.workflow.core.model.embedding.DefaultBatchingStrategy;
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
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.stereotype.Service;
import org.springframework.util.CollectionUtils;

import java.io.File;
import java.io.IOException;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.stream.Collectors;

import static com.seaskyland.llm.workflow.core.rag.RagConstants.KEY_ENABLED;

import javax.annotation.PreDestroy;

/**
 * Simple vector store service implementation. Provides functionality for managing
 * vector indices and document chunks using Spring AI's SimpleVectorStore.
 *
 * @since 1.0.0.3
 */
@Service
@Slf4j
@Qualifier("simpleVectorStoreService")
public class SimpleVectorStoreService implements VectorStoreService {

	/** Factory for creating embedding models */
	private final ModelFactory modelFactory;

	/** Storage path for vector store files */
	private final String storagePath;

	/** Cache for vector store instances to prevent data loss */
	private final Map<String, CustomSimpleVectorStore> vectorStoreCache = new ConcurrentHashMap<>();

	/** Map to keep track of index names and their file paths */
	private final Map<String, Path> indexPathMap = new ConcurrentHashMap<>();

	public SimpleVectorStoreService(ModelFactory modelFactory) {
		this.modelFactory = modelFactory;
		// Default storage path - could be made configurable
		this.storagePath = System.getProperty("user.home") + "/saa/vector_stores";
		// Create the storage directory if it doesn't exist
		File directory = new File(this.storagePath);
		if (!directory.exists()) {
			directory.mkdirs();
		}
	}

	/**
	 * Creates a new SimpleVectorStore index
	 * @param indexConfig Configuration for the index including name and embedding model
	 */
	@Override
	public void createIndex(IndexConfig indexConfig) {
		String indexName = indexConfig.getName();

		if (indexName == null || indexName.trim().isEmpty()) {
			throw new IllegalArgumentException("Index name must be provided");
		}

		try {
			// Create an empty vector store file
			Path indexPath = Paths.get(storagePath, indexName + ".json");
			File indexFile = indexPath.toFile();
			
			if (!indexFile.exists()) {
				indexFile.createNewFile();
				log.info("Created simple vector store index file: {}", indexFile.getAbsolutePath());
			} else {
				log.info("Vector store index file already exists: {}", indexFile.getAbsolutePath());
			}
		} catch (IOException e) {
			throw new RuntimeException("Failed to create index file for: " + indexName, e);
		}
	}

	/**
	 * Deletes an existing SimpleVectorStore index
	 * @param indexConfig Configuration containing the index name to delete
	 */
	@Override
	public void deleteIndex(IndexConfig indexConfig) {
		String indexName = indexConfig.getName();
		
		try {
			Path indexPath = Paths.get(storagePath, indexName + ".json");
			File indexFile = indexPath.toFile();
			
			if (indexFile.exists()) {
				if (indexFile.delete()) {
					log.info("Deleted simple vector store index file: {}", indexFile.getAbsolutePath());
				} else {
					log.warn("Failed to delete simple vector store index file: {}", indexFile.getAbsolutePath());
				}
			} else {
				log.warn("Index file not found for deletion: {}", indexFile.getAbsolutePath());
			}
			
			// Remove from cache if present
			vectorStoreCache.remove(indexName);
			indexPathMap.remove(indexName);
		} catch (Exception e) {
			throw new RuntimeException("Failed to delete index: " + indexName, e);
		}
	}

	/**
	 * Creates and returns a vector store instance for the specified index
	 * @param indexConfig Configuration for the index
	 * @return Configured vector store instance
	 */
	@Override
	public VectorStore getVectorStore(IndexConfig indexConfig) {
		String indexName = indexConfig.getName();
		
		// Return cached instance if available
		return vectorStoreCache.computeIfAbsent(indexName, name -> {
			EmbeddingModel embeddingModel = modelFactory.getEmbeddingModel(MetadataMode.EMBED, indexConfig);
			Path indexPath = Paths.get(storagePath, name + ".json");
			
			// Store the index path for shutdown save
			indexPathMap.put(name, indexPath);

			CustomSimpleVectorStore vectorStore =
					CustomSimpleVectorStore.builder(embeddingModel)
							.batchingStrategy(new DefaultBatchingStrategy())
							.build();
			
			// Load existing data if file exists
			File indexFile = indexPath.toFile();
			if (indexFile.exists() && indexFile.length() > 0) {
				vectorStore.load(indexFile);
			}
			
			return vectorStore;
		});
	}

	/**
	 * Lists document chunks from the index with pagination support
	 * @param indexConfig Index configuration
	 * @param searchRequest Search parameters including pagination and filters
	 * @return Paginated list of document chunks
	 */
	@Override
	public PagingList<DocumentChunk> listDocumentChunks(IndexConfig indexConfig, SearchRequest searchRequest) {
		try {
			VectorStore vectorStore = getVectorStore(indexConfig);
			
			// For SimpleVectorStore, we'll need to simulate listing since it doesn't have a native query mechanism
			// We'll use a search with a wildcard to get all documents
			SearchRequest listRequest = SearchRequest.builder()
					.topK(Integer.MAX_VALUE)
					.filterExpression(searchRequest.getFilterExpression())
					.similarityThreshold(0.0)
					.build();
			
			List<Document> documents = vectorStore.similaritySearch(listRequest);
			
			// Apply pagination manually
			int from = searchRequest.getFrom();
			int size = searchRequest.getTopK();
			
			List<DocumentChunk> chunks;
			if (!CollectionUtils.isEmpty(documents)) {
				// Apply manual pagination
				int toIndex = Math.min(from + size, documents.size());
				List<Document> pagedDocuments = documents.subList(from, toIndex);
				
				chunks = pagedDocuments.stream()
						.map(DocumentChunkConverter::toDocumentChunk)
						.collect(Collectors.toList());
			} else {
				chunks = new ArrayList<>();
			}
			
			long total = documents.size();
			int current = (from / size) + 1;
			
			return new PagingList<>(current, size, total, chunks);
		} catch (Exception e) {
			throw new BizException(ErrorCode.DOCUMENT_RETRIEVAL_ERROR.toError(), e);
		}
	}

	/**
	 * Updates multiple document chunks in the index
	 * @param indexConfig Index configuration
	 * @param chunks List of document chunks to update
	 */
	@Override
	public void updateDocumentChunks(IndexConfig indexConfig, List<DocumentChunk> chunks) {
		try {
			CustomSimpleVectorStore vectorStore =
					(CustomSimpleVectorStore) getVectorStore(indexConfig);
			
			List<Document> documents = chunks.stream()
					.map(DocumentChunkConverter::toDocument)
					.collect(Collectors.toList());
			
			// Add/update documents in the vector store
			vectorStore.add(documents);
			
			// Save the updated vector store
			Path indexPath = Paths.get(storagePath, indexConfig.getName() + ".json");
			vectorStore.save(indexPath.toFile());
		} catch (Exception e) {
			throw new BizException(ErrorCode.UPDATE_DOCUMENT_CHUNK_ERROR.toError(), e);
		}
	}

	/**
	 * Updates the enabled status of multiple document chunks
	 * @param indexConfig Index configuration
	 * @param chunkIds List of chunk IDs to update
	 * @param enabled New enabled status
	 */
	@Override
	public void updateDocumentChunkStatus(IndexConfig indexConfig, List<String> chunkIds, boolean enabled) {
		try {
			CustomSimpleVectorStore vectorStore =
					(CustomSimpleVectorStore) getVectorStore(indexConfig);
			
			// Get all documents
			SearchRequest listRequest = SearchRequest.builder().topK(Integer.MAX_VALUE).build();
			List<Document> allDocuments = vectorStore.similaritySearch(listRequest);
			
			// Update documents that match the chunk IDs
			List<Document> documentsToUpdate = new ArrayList<>();
			for (Document document : allDocuments) {
				if (chunkIds.contains(document.getId())) {
					// Update the enabled status in metadata
					Map<String, Object> metadata = new HashMap<>(document.getMetadata());
					metadata.put(KEY_ENABLED, enabled);
					
					// Create updated document
					Document updatedDocument = new Document(document.getId(), document.getText(), metadata);
					documentsToUpdate.add(updatedDocument);
				}
			}
			
			// Delete old documents and add updated ones
			if (!documentsToUpdate.isEmpty()) {
				vectorStore.delete(chunkIds);
				vectorStore.add(documentsToUpdate);
				
				// Save the updated vector store
				Path indexPath = Paths.get(storagePath, indexConfig.getName() + ".json");
				vectorStore.save(indexPath.toFile());
			}
		} catch (Exception e) {
			throw new BizException(ErrorCode.UPDATE_DOCUMENT_CHUNK_ERROR.toError(), e);
		}
	}
	
	/**
	 * Shutdown hook to save all vector stores before application shutdown
	 */
	@PreDestroy
	public void shutdown() {
		log.info("Shutting down SimpleVectorStoreService, saving all vector stores...");
		for (Map.Entry<String, CustomSimpleVectorStore> entry : vectorStoreCache.entrySet()) {
			String indexName = entry.getKey();
			CustomSimpleVectorStore vectorStore = entry.getValue();
			Path indexPath = indexPathMap.get(indexName);
			
			if (indexPath != null) {
				try {
					vectorStore.save(indexPath.toFile());
					log.info("Saved vector store for index: {}", indexName);
				} catch (Exception e) {
					log.error("Failed to save vector store for index: {}", indexName, e);
				}
			} else {
				log.warn("Index path not found for index: {}", indexName);
			}
		}
		log.info("Finished shutting down SimpleVectorStoreService");
	}
}
