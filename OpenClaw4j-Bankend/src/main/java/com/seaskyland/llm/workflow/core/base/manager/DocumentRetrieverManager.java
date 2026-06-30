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

import com.seaskyland.llm.workflow.core.rag.DocumentChunkConverter;
import com.seaskyland.llm.workflow.core.rag.KnowledgeBaseService;
import com.seaskyland.llm.workflow.core.rag.retriever.KnowledgeBaseDocumentRetriever;
import com.seaskyland.llm.workflow.core.rag.vectorstore.VectorStoreFactory;
import com.seaskyland.llm.workflow.runtime.domain.app.FileSearchOptions;
import com.seaskyland.llm.workflow.runtime.domain.knowledgebase.DocumentChunk;
import com.seaskyland.llm.workflow.runtime.domain.knowledgebase.KnowledgeBase;
import java.util.List;
import lombok.RequiredArgsConstructor;
import org.springframework.ai.document.Document;
import org.springframework.ai.rag.Query;
import org.springframework.ai.rag.retrieval.search.DocumentRetriever;
import org.springframework.stereotype.Component;

/**
 * Manager class for document retrieval operations. Handles the creation and execution of document
 * retrievers for knowledge bases.
 *
 * @since 1.0.0.3
 */
@Component
@RequiredArgsConstructor
public class DocumentRetrieverManager {

  /** Factory for creating vector stores */
  private final VectorStoreFactory vectorStoreFactory;

  /** Service for managing knowledge bases */
  private final KnowledgeBaseService knowledgeBaseService;

  /**
   * Creates a document retriever for the specified search options.
   *
   * @param searchOptions Options for file search
   * @return Configured document retriever
   */
  public DocumentRetriever getDocumentRetriever(FileSearchOptions searchOptions) {
    List<KnowledgeBase> knowledgeBases =
        knowledgeBaseService.listKnowledgeBases(searchOptions.getKbIds());
    return new KnowledgeBaseDocumentRetriever(knowledgeBases, vectorStoreFactory, searchOptions);
  }

  /**
   * Retrieves document chunks based on the query and search options.
   *
   * @param query Search query
   * @param searchOptions Options for file search
   * @return List of retrieved document chunks
   */
  public List<DocumentChunk> retrieve(Query query, FileSearchOptions searchOptions) {
    DocumentRetriever documentRetriever = getDocumentRetriever(searchOptions);
    List<Document> documents = documentRetriever.retrieve(query);
    return documents.stream().map(DocumentChunkConverter::toDocumentChunk).toList();
  }
}
