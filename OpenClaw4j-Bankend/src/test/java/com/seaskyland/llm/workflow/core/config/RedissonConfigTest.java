package com.seaskyland.llm.workflow.core.config;

import org.junit.jupiter.api.Test;
import org.redisson.config.Config;
import org.redisson.config.SentinelServersConfig;
import org.redisson.config.SingleServerConfig;
import org.springframework.boot.data.redis.autoconfigure.DataRedisProperties;

import java.lang.reflect.Method;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;

class RedissonConfigTest {

	@Test
	void buildsSentinelConfigFromSpringDataRedisProperties() throws Exception {
		DataRedisProperties properties = new DataRedisProperties();
		properties.setDatabase(3);
		properties.setUsername("redis-user");
		properties.setPassword("redis-secret");

		DataRedisProperties.Sentinel sentinel = new DataRedisProperties.Sentinel();
		sentinel.setMaster("openclaw4j-master");
		sentinel.setNodes(List.of("localhost:26379", "redis://localhost:26380"));
		sentinel.setUsername("sentinel-user");
		sentinel.setPassword("sentinel-secret");
		properties.setSentinel(sentinel);

		Config config = new RedissonConfig(properties).buildConfig();

		assertThat(config.isSentinelConfig()).isTrue();
		SentinelServersConfig sentinelConfig = getConfig(config, "getSentinelServersConfig");
		assertThat(sentinelConfig.getMasterName()).isEqualTo("openclaw4j-master");
		assertThat(sentinelConfig.getDatabase()).isEqualTo(3);
		assertThat(sentinelConfig.getSentinelAddresses())
			.containsExactly("redis://localhost:26379", "redis://localhost:26380");
		assertThat(config.getUsername()).isEqualTo("redis-user");
		assertThat(config.getPassword()).isEqualTo("redis-secret");
		assertThat(sentinelConfig.getSentinelUsername()).isEqualTo("sentinel-user");
		assertThat(sentinelConfig.getSentinelPassword()).isEqualTo("sentinel-secret");
	}

	@Test
	void fallsBackToSingleServerWhenSentinelIsNotConfigured() throws Exception {
		DataRedisProperties properties = new DataRedisProperties();
		properties.setHost("localhost");
		properties.setPort(6379);
		properties.setDatabase(1);

		Config config = new RedissonConfig(properties).buildConfig();

		assertThat(config.isSingleConfig()).isTrue();
		SingleServerConfig singleServerConfig = getConfig(config, "getSingleServerConfig");
		assertThat(singleServerConfig.getAddress()).isEqualTo("redis://localhost:6379");
		assertThat(singleServerConfig.getDatabase()).isEqualTo(1);
	}

	@SuppressWarnings("unchecked")
	private static <T> T getConfig(Config config, String methodName) throws Exception {
		Method method = Config.class.getDeclaredMethod(methodName);
		method.setAccessible(true);
		return (T) method.invoke(config);
	}

}
