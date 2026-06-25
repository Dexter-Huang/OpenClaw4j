package com.seaskyland.llm;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
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

	private static final Duration REQUEST_TIMEOUT = Duration.ofSeconds(3);

	private final ApplicationContext applicationContext;

	private final Environment environment;

	private final List<String> warmupPaths;

	public LeydenProfiledTrainingRunner(ApplicationContext applicationContext, Environment environment,
			@Value("${openclaw4j.leyden.training.warmup-paths:/console/v1/system/health,/console/v1/system/global-config,/api/prompts?pageNo=1&pageSize=1,/api/observability/overview}") String warmupPaths) {
		this.applicationContext = applicationContext;
		this.environment = environment;
		this.warmupPaths = parseWarmupPaths(warmupPaths);
	}

	@Override
	public void onApplicationEvent(ApplicationReadyEvent event) {
		int port = environment.getProperty("local.server.port", Integer.class, 0);
		if (port <= 0) {
			log.warn("Leyden profiled training cannot resolve local.server.port; exiting training context");
			exit();
			return;
		}

		HttpClient client = HttpClient.newBuilder()
			.connectTimeout(REQUEST_TIMEOUT)
			.followRedirects(HttpClient.Redirect.NEVER)
			.build();
		for (String path : warmupPaths) {
			warmup(client, port, path);
		}
		exit();
	}

	private void warmup(HttpClient client, int port, String path) {
		URI uri = URI.create("http://127.0.0.1:" + port + normalizePath(path));
		try {
			HttpRequest request = HttpRequest.newBuilder(uri)
				.timeout(REQUEST_TIMEOUT)
				.GET()
				.header("Accept", "application/json,text/plain,*/*")
				.build();
			HttpResponse<Void> response = client.send(request, HttpResponse.BodyHandlers.discarding());
			log.info("Leyden profiled warmup {} -> {}", uri.getRawPath(), response.statusCode());
		}
		catch (Exception ex) {
			log.warn("Leyden profiled warmup {} failed: {}", uri.getRawPath(), ex.toString());
		}
	}

	private void exit() {
		int code = SpringApplication.exit(applicationContext, () -> 0);
		System.exit(code);
	}

	private static List<String> parseWarmupPaths(String value) {
		return Arrays.stream(value.split(",")).map(String::trim).filter(path -> !path.isBlank()).toList();
	}

	private static String normalizePath(String path) {
		return path.startsWith("/") ? path : "/" + path;
	}

}
