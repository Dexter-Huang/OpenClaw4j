package com.seaskyland.llm;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.boot.context.event.ApplicationReadyEvent;
import org.springframework.context.ApplicationContext;
import org.springframework.context.ApplicationListener;
import org.springframework.core.env.Environment;
import org.springframework.stereotype.Component;

@Slf4j
@Component
@ConditionalOnProperty(name = "openclaw4j.leyden.training.profiled", havingValue = "true")
public class LeydenProfiledTrainingRunner implements ApplicationListener<ApplicationReadyEvent> {

  private static final Duration DEFAULT_REQUEST_TIMEOUT = Duration.ofSeconds(10);

  private final ApplicationContext applicationContext;

  private final Environment environment;

  private final List<String> warmupPaths;

  private final Duration requestTimeout;

  public LeydenProfiledTrainingRunner(
      ApplicationContext applicationContext,
      Environment environment,
      @Value(
              "${openclaw4j.leyden.training.warmup-paths:/console/v1/system/health,/console/v1/system/global-config,/api/prompts?pageNo=1&pageSize=1,/api/observability/overview}")
          String warmupPaths,
      @Value("${openclaw4j.leyden.training.request-timeout:10s}") Duration requestTimeout) {
    this.applicationContext = applicationContext;
    this.environment = environment;
    this.warmupPaths = parseWarmupPaths(warmupPaths);
    this.requestTimeout =
        requestTimeout.isZero() || requestTimeout.isNegative()
            ? DEFAULT_REQUEST_TIMEOUT
            : requestTimeout;
  }

  @Override
  public void onApplicationEvent(ApplicationReadyEvent event) {
    int port = environment.getProperty("local.server.port", Integer.class, 0);
    if (port <= 0) {
      log.warn(
          "Leyden profiled training cannot resolve local.server.port; exiting training context");
      exit();
      return;
    }

    HttpClient client =
        HttpClient.newBuilder()
            .connectTimeout(requestTimeout)
            .followRedirects(HttpClient.Redirect.NEVER)
            .build();
    List<WarmupResult> results = new ArrayList<>();
    for (String path : warmupPaths) {
      results.add(warmup(client, port, path));
    }
    logSummary(results);
    exit();
  }

  private WarmupResult warmup(HttpClient client, int port, String path) {
    URI uri = URI.create("http://127.0.0.1:" + port + normalizePath(path));
    long startedAt = System.nanoTime();
    try {
      HttpRequest request =
          HttpRequest.newBuilder(uri)
              .timeout(requestTimeout)
              .GET()
              .header("Accept", "application/json,text/plain,*/*")
              .build();
      HttpResponse<Void> response = client.send(request, HttpResponse.BodyHandlers.discarding());
      long elapsedMillis = elapsedMillis(startedAt);
      log.info(
          "Leyden profiled warmup {} -> {} ({} ms)",
          uri.getRawPath(),
          response.statusCode(),
          elapsedMillis);
      return new WarmupResult(uri.getRawPath(), true, response.statusCode(), elapsedMillis, null);
    } catch (Exception ex) {
      long elapsedMillis = elapsedMillis(startedAt);
      log.warn(
          "Leyden profiled warmup {} failed after {} ms: {}",
          uri.getRawPath(),
          elapsedMillis,
          ex.toString());
      return new WarmupResult(uri.getRawPath(), false, 0, elapsedMillis, ex.toString());
    }
  }

  private void logSummary(List<WarmupResult> results) {
    long succeeded = results.stream().filter(WarmupResult::success).count();
    long failed = results.size() - succeeded;
    long elapsedMillis = results.stream().mapToLong(WarmupResult::elapsedMillis).sum();
    log.info(
        "Leyden profiled warmup summary: {} succeeded, {} failed, {} ms total",
        succeeded,
        failed,
        elapsedMillis);
  }

  private void exit() {
    int code = SpringApplication.exit(applicationContext, () -> 0);
    System.exit(code);
  }

  private static long elapsedMillis(long startedAt) {
    return Duration.ofNanos(System.nanoTime() - startedAt).toMillis();
  }

  private static List<String> parseWarmupPaths(String value) {
    return Arrays.stream(value.split(","))
        .map(String::trim)
        .filter(path -> !path.isBlank())
        .toList();
  }

  private static String normalizePath(String path) {
    return path.startsWith("/") ? path : "/" + path;
  }

  private record WarmupResult(
      String path, boolean success, int statusCode, long elapsedMillis, String error) {}
}
