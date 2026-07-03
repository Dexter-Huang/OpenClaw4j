package com.seaskyland.llm.workflow.core.rag.retriever;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

import com.seaskyland.llm.workflow.core.rag.vectorstore.VectorStoreFactory;
import com.seaskyland.llm.workflow.core.rag.vectorstore.VectorStoreService;
import com.seaskyland.llm.workflow.runtime.domain.app.FileSearchOptions;
import com.seaskyland.llm.workflow.runtime.domain.knowledgebase.IndexConfig;
import com.seaskyland.llm.workflow.runtime.domain.knowledgebase.KnowledgeBase;
import java.util.List;
import java.util.Map;
import org.junit.jupiter.api.Test;
import org.springframework.ai.document.Document;
import org.springframework.ai.rag.Query;
import org.springframework.ai.vectorstore.SearchRequest;
import org.springframework.ai.vectorstore.VectorStore;

class KnowledgeBaseDocumentRetrieverTest {

  @Test
  void retrieveFiltersDocumentsWithAppSearchOptions() {
    VectorStore vectorStore = mock(VectorStore.class);
    VectorStoreService vectorStoreService = mock(VectorStoreService.class);
    VectorStoreFactory vectorStoreFactory = mock(VectorStoreFactory.class);
    when(vectorStoreFactory.getVectorStoreService()).thenReturn(vectorStoreService);
    when(vectorStoreService.getVectorStore(any())).thenReturn(vectorStore);

    Document relevantDocument =
        Document.builder().id("chunk-1").text("relevant").metadata(Map.of()).score(0.8).build();
    Document weakDocument =
        Document.builder().id("chunk-2").text("weak").metadata(Map.of()).score(0.3).build();
    when(vectorStore.similaritySearch(any(SearchRequest.class)))
        .thenReturn(List.of(relevantDocument, weakDocument));

    KnowledgeBase knowledgeBase = new KnowledgeBase();
    knowledgeBase.setKbId("kb-1");
    knowledgeBase.setWorkspaceId("workspace-1");
    knowledgeBase.setIndexConfig(new IndexConfig());
    FileSearchOptions knowledgeBaseSearchConfig = new FileSearchOptions();
    knowledgeBaseSearchConfig.setTopK(1);
    knowledgeBaseSearchConfig.setSimilarityThreshold(0.5f);
    knowledgeBase.setSearchConfig(knowledgeBaseSearchConfig);

    FileSearchOptions appSearchOptions = new FileSearchOptions();
    appSearchOptions.setKbIds(List.of("kb-1"));
    appSearchOptions.setTopK(1);
    appSearchOptions.setSimilarityThreshold(0.5f);
    KnowledgeBaseDocumentRetriever retriever =
        new KnowledgeBaseDocumentRetriever(
            List.of(knowledgeBase), vectorStoreFactory, appSearchOptions);

    List<Document> documents =
        retriever.retrieve(Query.builder().text("question").context(Map.of()).build());

    assertThat(documents).containsExactly(relevantDocument);
  }
}
