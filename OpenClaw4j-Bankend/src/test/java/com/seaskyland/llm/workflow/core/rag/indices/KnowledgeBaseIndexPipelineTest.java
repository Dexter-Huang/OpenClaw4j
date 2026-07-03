package com.seaskyland.llm.workflow.core.rag.indices;

import static org.assertj.core.api.Assertions.assertThat;

import com.seaskyland.llm.workflow.runtime.domain.knowledgebase.ProcessConfig;
import com.seaskyland.llm.workflow.runtime.enums.ChunkType;
import java.util.List;
import org.junit.jupiter.api.Test;
import org.springframework.ai.document.Document;

class KnowledgeBaseIndexPipelineTest {

  @Test
  void lengthChunkingUsesTokenSplitterDefaults() {
    KnowledgeBaseIndexPipeline pipeline = new KnowledgeBaseIndexPipeline(null, null, null);
    ProcessConfig processConfig = new ProcessConfig();
    processConfig.setChunkType(ChunkType.LENGTH);
    processConfig.setChunkSize(80);
    processConfig.setChunkOverlap(10);

    List<Document> chunks =
        pipeline.transform(
            List.of(new Document("第一段内容。第二段内容？第三段内容！This text should be chunked.")), processConfig);

    assertThat(chunks).isNotEmpty();
  }
}
