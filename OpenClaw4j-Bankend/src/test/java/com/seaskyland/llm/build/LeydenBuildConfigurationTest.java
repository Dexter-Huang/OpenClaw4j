package com.seaskyland.llm.build;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;

import org.junit.jupiter.api.Test;

class LeydenBuildConfigurationTest {

	private static final Path PROJECT_DIR = Path.of("").toAbsolutePath();

	@Test
	void mavenBuildKeepsGraalVmNativeImageInExperimentProfileOnly() throws IOException {
		String pom = read("pom.xml");

		assertTrue(pom.contains("<id>leyden</id>"));
		assertTrue(pom.contains("<id>native-experiment</id>"));
		assertTrue(pom.contains("org.graalvm.buildtools"));
		assertTrue(pom.contains("native-maven-plugin"));
		assertTrue(pom.contains("<skipNativeBuild>true</skipNativeBuild>"));

		String leydenProfile = profileBlock(pom, "leyden");
		String nativeProfile = profileBlock(pom, "native-experiment");
		assertFalse(leydenProfile.contains("native-maven-plugin"));
		assertFalse(leydenProfile.contains("org.graalvm.buildtools"));
		assertTrue(nativeProfile.contains("native-maven-plugin"));
		assertTrue(nativeProfile.contains("<imageName>OpenClaw4j-Bankend-native</imageName>"));
		assertTrue(nativeProfile.contains("maven-compiler-plugin"));
		assertTrue(nativeProfile.contains("annotationProcessorPaths"));
		assertTrue(nativeProfile.contains("org.projectlombok"));
	}

	@Test
	void leydenTrainingAndRuntimeScriptsUseHotSpotAotCache() throws IOException {
		String trainScript = read("scripts/train-leyden-aot.ps1");
		String runScript = read("scripts/run-leyden-aot.ps1");

		assertTrue(trainScript.contains("-XX:AOTCacheOutput="));
		assertTrue(trainScript.contains("-Djarmode=tools"));
		assertTrue(trainScript.contains("TrainingTimeoutSeconds"));
		assertTrue(trainScript.contains("TrainingTimeoutSeconds = 900"));
		assertTrue(trainScript.contains("Start-Job"));
		assertTrue(trainScript.contains("profiled"));
		assertTrue(trainScript.contains("openclaw4j.leyden.training.profiled=true"));
		assertTrue(trainScript.contains("LeydenTrainingApplication"));
		assertTrue(trainScript.contains("Get-LeydenClasspath"));
		assertTrue(trainScript.contains("$appJar"));
		assertTrue(runScript.contains("-XX:AOTCache="));
		assertTrue(runScript.contains("Get-LeydenClasspath"));
		assertTrue(runScript.contains("LLMApplication"));
		assertTrue(runScript.contains("$appJar"));
		assertFalse(trainScript.contains("native-image"));
		assertFalse(runScript.contains("native-image"));
	}

	@Test
	void dockerfileBuildsAndRunsExtractedApplicationWithAotCache() throws IOException {
		String dockerfile = read("Dockerfile");

		assertTrue(dockerfile.contains("maven:3.9-eclipse-temurin-26"));
		assertTrue(dockerfile.contains("eclipse-temurin:26-jre"));
		assertTrue(dockerfile.contains("-Djarmode=tools"));
		assertTrue(dockerfile.contains("-XX:+UnlockExperimentalVMOptions"));
		assertTrue(dockerfile.contains("-XX:+UseCompactObjectHeaders"));
		assertTrue(dockerfile.contains("-XX:AOTCacheOutput=openclaw4j.aot"));
		assertTrue(dockerfile.contains("-XX:AOTCache=openclaw4j.aot"));
		assertTrue(dockerfile.contains("test -f openclaw4j.aot"));
		assertTrue(dockerfile.contains("ls -lh openclaw4j.aot"));
		assertTrue(dockerfile.contains("openclaw4j.leyden.training.profiled=true"));
		assertTrue(dockerfile.contains("-jar OpenClaw4j-Bankend.jar"));
		assertTrue(dockerfile.contains("JAVA_OPTS"));
		assertFalse(dockerfile.contains("classpath.txt"));
		assertFalse(dockerfile.contains("native-image"));
		assertFalse(dockerfile.contains("graalvm"));
		assertFalse(dockerfile.contains("UseZGC"));
		assertFalse(dockerfile.contains("UseG1GC"));
	}

	@Test
	void profiledTrainingRunnerWarmsLocalEndpointsAndExitsNormally() throws IOException {
		String runner = read("src/main/java/com/seaskyland/llm/LeydenProfiledTrainingRunner.java");
		String benchmarkScript = read("scripts/benchmark-leyden-startup.ps1");

		assertTrue(runner.contains("@ConditionalOnProperty"));
		assertTrue(runner.contains("ApplicationReadyEvent"));
		assertTrue(runner.contains("local.server.port"));
		assertTrue(runner.contains("/console/v1/system/health"));
		assertTrue(runner.contains("SpringApplication.exit"));
		assertTrue(benchmarkScript.contains("Started LLMApplication in"));
		assertTrue(benchmarkScript.contains("Group-Object leyden"));
	}

	@Test
	void githubActionsBuildsAndPublishesOptimizedLeydenDockerImageOnPush() throws IOException {
		String workflow = Files.readString(PROJECT_DIR.getParent().resolve(".github/workflows/backend-native.yml"));

		assertTrue(workflow.contains("packages: write"));
		assertTrue(workflow.contains("docker/login-action@v3"));
		assertTrue(workflow.contains("docker/metadata-action@v5"));
		assertTrue(workflow.contains("docker/build-push-action@v6"));
		assertTrue(workflow.contains("context: ./OpenClaw4j-Bankend"));
		assertTrue(workflow.contains("push: ${{ github.event_name != 'pull_request' }}"));
		assertTrue(workflow.contains("ghcr.io/dexter-huang/openclaw4j-backend"));
		assertTrue(workflow.contains("OPENCLAW4J_IMAGE_DIGEST"));
		assertFalse(workflow.contains("graalvm/setup-graalvm"));
		assertFalse(workflow.contains("native:compile"));
		assertFalse(workflow.contains("OpenClaw4j-Bankend-native"));
	}

	private String read(String relativePath) throws IOException {
		return Files.readString(PROJECT_DIR.resolve(relativePath));
	}

	private String profileBlock(String pom, String profileId) {
		String marker = "<id>" + profileId + "</id>";
		int idIndex = pom.indexOf(marker);
		assertTrue(idIndex >= 0, "Missing Maven profile: " + profileId);

		int start = pom.lastIndexOf("<profile>", idIndex);
		int end = pom.indexOf("</profile>", idIndex);
		assertTrue(start >= 0 && end > idIndex, "Invalid Maven profile block: " + profileId);
		return pom.substring(start, end + "</profile>".length());
	}

}
