package com.seaskyland.llm.workflow.core.rag.vectorstore;

import com.seaskyland.llm.workflow.core.config.StudioProperties;
import com.seaskyland.llm.workflow.runtime.domain.PagingList;
import com.seaskyland.llm.workflow.runtime.domain.knowledgebase.DocumentChunk;
import com.seaskyland.llm.workflow.runtime.domain.knowledgebase.IndexConfig;
import org.junit.jupiter.api.Test;
import org.springframework.ai.vectorstore.SearchRequest;
import org.springframework.ai.vectorstore.VectorStore;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertSame;

class VectorStoreFactoryTest {

	@Test
	void returnsPgvectorServiceWhenConfigured() {
		StudioProperties properties = new StudioProperties();
		properties.setVectorStoreType("pgvector");
		VectorStoreService pgvectorService = new StubVectorStoreService();

		VectorStoreFactory factory = new VectorStoreFactory(properties,
				Map.of("pgvectorVectorStoreService", pgvectorService));

		assertSame(pgvectorService, factory.getVectorStoreService());
	}

	private static final class StubVectorStoreService implements VectorStoreService {

		@Override
		public void createIndex(IndexConfig indexConfig) {
		}

		@Override
		public void deleteIndex(IndexConfig indexConfig) {
		}

		@Override
		public VectorStore getVectorStore(IndexConfig indexConfig) {
			return null;
		}

		@Override
		public PagingList<DocumentChunk> listDocumentChunks(IndexConfig indexConfig, SearchRequest searchRequest) {
			return null;
		}

		@Override
		public void updateDocumentChunks(IndexConfig indexConfig, List<DocumentChunk> chunks) {
		}

		@Override
		public void updateDocumentChunkStatus(IndexConfig indexConfig, List<String> chunkIds, boolean enabled) {
		}

	}

}
