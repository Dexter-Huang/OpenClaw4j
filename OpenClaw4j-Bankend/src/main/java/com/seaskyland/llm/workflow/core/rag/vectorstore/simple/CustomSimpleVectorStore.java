package com.seaskyland.llm.workflow.core.rag.vectorstore.simple;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.ObjectWriter;
import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.io.Writer;
import java.nio.charset.StandardCharsets;
import java.nio.file.FileAlreadyExistsException;
import java.nio.file.Files;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.function.Predicate;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.ai.document.Document;
import org.springframework.ai.embedding.EmbeddingModel;
import org.springframework.ai.observation.conventions.VectorStoreProvider;
import org.springframework.ai.observation.conventions.VectorStoreSimilarityMetric;
import org.springframework.ai.vectorstore.AbstractVectorStoreBuilder;
import org.springframework.ai.vectorstore.SearchRequest;
import org.springframework.ai.vectorstore.filter.Filter;
import org.springframework.ai.vectorstore.filter.FilterExpressionConverter;
import org.springframework.ai.vectorstore.filter.converter.PrintFilterExpressionConverter;
import org.springframework.ai.vectorstore.observation.AbstractObservationVectorStore;
import org.springframework.ai.vectorstore.observation.VectorStoreObservationContext;
import org.springframework.core.io.Resource;
import org.springframework.expression.ExpressionParser;
import org.springframework.expression.spel.standard.SpelExpressionParser;
import org.springframework.expression.spel.support.StandardEvaluationContext;

/**
 * SimpleVectorStore is a simple implementation of the VectorStore interface.
 *
 * <p>It also provides methods to save the current state of the vectors to a file, and to load
 * vectors from a file.
 *
 * <p>For a deeper understanding of the mathematical concepts and computations involved in
 * calculating similarity scores among vectors, refer to this
 * [resource](https://docs.spring.io/spring-ai/reference/api/vectordbs.html#_understanding_vectors).
 *
 * @author Raphael Yu
 * @author Dingmeng Xue
 * @author Mark Pollack
 * @author Christian Tzolov
 * @author Sebastien Deleuze
 * @author Ilayaperumal Gopinathan
 * @author Thomas Vitale
 * @author Jemin Huh
 */
public class CustomSimpleVectorStore extends AbstractObservationVectorStore {

  private static final Logger logger = LoggerFactory.getLogger(CustomSimpleVectorStore.class);

  private final ObjectMapper objectMapper;

  private final ExpressionParser expressionParser;

  private final FilterExpressionConverter filterExpressionConverter;

  protected Map<String, StoreContent> store = new ConcurrentHashMap<>();

  protected CustomSimpleVectorStore(CustomSimpleVectorStoreBuilder builder) {
    super(builder);
    this.objectMapper = new ObjectMapper().findAndRegisterModules();
    this.expressionParser = new SpelExpressionParser();
    // Use our custom filter expression converter instead of the default one
    this.filterExpressionConverter = new PrintFilterExpressionConverter();
  }

  /**
   * Creates an instance of SimpleVectorStore builder.
   *
   * @return the SimpleVectorStore builder.
   */
  public static CustomSimpleVectorStoreBuilder builder(EmbeddingModel embeddingModel) {
    return new CustomSimpleVectorStoreBuilder(embeddingModel);
  }

  @Override
  public void doAdd(List<Document> documents) {
    Objects.requireNonNull(documents, "Documents list cannot be null");
    if (documents.isEmpty()) {
      throw new IllegalArgumentException("Documents list cannot be empty");
    }

    for (Document document : documents) {
      logger.info("Calling EmbeddingModel for document id = {}", document.getId());
      float[] embedding = this.embeddingModel.embed(document);
      StoreContent storeContent =
          new StoreContent(document.getId(), document.getText(), document.getMetadata(), embedding);
      this.store.put(document.getId(), storeContent);
    }
  }

  @Override
  public void doDelete(List<String> idList) {
    for (String id : idList) {
      this.store.remove(id);
    }
  }

  @Override
  public void doDelete(Filter.Expression filterExpression) {
    Predicate<StoreContent> documentFilterPredicate =
        doFilterPredicate(SearchRequest.builder().filterExpression(filterExpression).build());
    this.store.values().removeIf(documentFilterPredicate);
  }

  public static Predicate<StoreContent> doFilterPredicate2(SearchRequest request) {
    return request.hasFilterExpression()
        ? document -> {
          StandardEvaluationContext context = new StandardEvaluationContext();
          context.setVariable("metadata", document.getMetadata());
          return Boolean.TRUE.equals(
              new SpelExpressionParser()
                  .parseExpression(
                      new PrintFilterExpressionConverter()
                          .convertExpression(request.getFilterExpression()))
                  .getValue(context, Boolean.class));
        }
        : document -> true;
  }

  @Override
  public List<Document> doSimilaritySearch(SearchRequest request) {
    Predicate<StoreContent> documentFilterPredicate = doFilterPredicate(request);
    float[] userQueryEmbedding = getUserQueryEmbedding(request.getQuery());
    return this.store.values().stream()
        .filter(documentFilterPredicate)
        .map(
            content ->
                content.toDocument(
                    CustomSimpleVectorStore.EmbeddingMath.cosineSimilarity(
                        userQueryEmbedding, content.getEmbedding())))
        .filter(document -> document.getScore() >= request.getSimilarityThreshold())
        .sorted(Comparator.comparing(Document::getScore).reversed())
        .limit(request.getTopK())
        .toList();
  }

  private Predicate<StoreContent> doFilterPredicate(SearchRequest request) {
    return request.hasFilterExpression()
        ? document -> {
          StandardEvaluationContext context = new StandardEvaluationContext();
          context.setVariable("metadata", document.getMetadata());
          return Boolean.TRUE.equals(
              this.expressionParser
                  .parseExpression(
                      this.filterExpressionConverter.convertExpression(
                          request.getFilterExpression()))
                  .getValue(context, Boolean.class));
        }
        : document -> true;
  }

  /**
   * Serialize the vector store content into a file in JSON format.
   *
   * @param file the file to save the vector store content
   */
  public void save(File file) {
    String json = getVectorDbAsJson();
    try {
      if (!file.exists()) {
        logger.info("Creating new vector store file: {}", file);
        try {
          Files.createFile(file.toPath());
        } catch (FileAlreadyExistsException e) {
          throw new RuntimeException("File already exists: " + file, e);
        } catch (IOException e) {
          throw new RuntimeException(
              "Failed to create new file: " + file + ". Reason: " + e.getMessage(), e);
        }
      } else {
        logger.info("Overwriting existing vector store file: {}", file);
      }
      try (OutputStream stream = new FileOutputStream(file);
          Writer writer = new OutputStreamWriter(stream, StandardCharsets.UTF_8)) {
        writer.write(json);
        writer.flush();
      }
    } catch (IOException ex) {
      logger.error("IOException occurred while saving vector store file.", ex);
      throw new RuntimeException(ex);
    } catch (SecurityException ex) {
      logger.error("SecurityException occurred while saving vector store file.", ex);
      throw new RuntimeException(ex);
    } catch (NullPointerException ex) {
      logger.error("NullPointerException occurred while saving vector store file.", ex);
      throw new RuntimeException(ex);
    }
  }

  /**
   * Deserialize the vector store content from a file in JSON format into memory.
   *
   * @param file the file to load the vector store content
   */
  public void load(File file) {
    TypeReference<HashMap<String, StoreContent>> typeRef = new TypeReference<>() {};

    try {
      this.store.putAll(this.objectMapper.readValue(file, typeRef));
    } catch (IOException ex) {
      throw new RuntimeException(ex);
    }
  }

  /**
   * Deserialize the vector store content from a resource in JSON format into memory.
   *
   * @param resource the resource to load the vector store content
   */
  public void load(Resource resource) {
    TypeReference<HashMap<String, StoreContent>> typeRef = new TypeReference<>() {};

    try {
      this.store.putAll(this.objectMapper.readValue(resource.getInputStream(), typeRef));
    } catch (IOException ex) {
      throw new RuntimeException(ex);
    }
  }

  private String getVectorDbAsJson() {
    ObjectWriter objectWriter = this.objectMapper.writerWithDefaultPrettyPrinter();
    try {
      return objectWriter.writeValueAsString(this.store);
    } catch (JsonProcessingException e) {
      throw new RuntimeException("Error serializing documentMap to JSON.", e);
    }
  }

  private float[] getUserQueryEmbedding(String query) {
    return this.embeddingModel.embed(query);
  }

  @Override
  public VectorStoreObservationContext.Builder createObservationContextBuilder(
      String operationName) {

    return VectorStoreObservationContext.builder(VectorStoreProvider.SIMPLE.value(), operationName)
        .dimensions(this.embeddingModel.dimensions())
        .collectionName("in-memory-map")
        .similarityMetric(VectorStoreSimilarityMetric.COSINE.value());
  }

  public static final class EmbeddingMath {

    private EmbeddingMath() {
      throw new UnsupportedOperationException("This is a utility class and cannot be instantiated");
    }

    public static double cosineSimilarity(float[] vectorX, float[] vectorY) {
      if (vectorX == null || vectorY == null) {
        throw new RuntimeException("Vectors must not be null");
      }
      if (vectorX.length != vectorY.length) {
        throw new IllegalArgumentException("Vectors lengths must be equal");
      }

      float dotProduct = dotProduct(vectorX, vectorY);
      float normX = norm(vectorX);
      float normY = norm(vectorY);

      if (normX == 0 || normY == 0) {
        throw new IllegalArgumentException("Vectors cannot have zero norm");
      }

      return dotProduct / (Math.sqrt(normX) * Math.sqrt(normY));
    }

    public static float dotProduct(float[] vectorX, float[] vectorY) {
      if (vectorX.length != vectorY.length) {
        throw new IllegalArgumentException("Vectors lengths must be equal");
      }

      float result = 0;
      for (int i = 0; i < vectorX.length; ++i) {
        result += vectorX[i] * vectorY[i];
      }

      return result;
    }

    public static float norm(float[] vector) {
      return dotProduct(vector, vector);
    }
  }

  protected record StoreContent(
      String id, String text, Map<String, Object> metadata, float[] embedding) {

    public String getId() {
      return this.id;
    }

    public String getText() {
      return this.text;
    }

    public Map<String, Object> getMetadata() {
      return this.metadata;
    }

    public float[] getEmbedding() {
      return this.embedding;
    }

    public Document toDocument(Double score) {
      Document.Builder builder =
          Document.builder().id(this.id).text(this.text).metadata(this.metadata);
      if (score != null) {
        builder.score(score);
      }
      return builder.build();
    }
  }

  public static final class CustomSimpleVectorStoreBuilder
      extends AbstractVectorStoreBuilder<CustomSimpleVectorStoreBuilder> {

    private CustomSimpleVectorStoreBuilder(EmbeddingModel embeddingModel) {
      super(embeddingModel);
    }

    @Override
    public CustomSimpleVectorStore build() {
      return new CustomSimpleVectorStore(this);
    }
  }
}
