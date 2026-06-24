/*
 * Copyright 2024 the original author or authors.
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

package com.seaskyland.llm.workflow.core.config;

import com.seaskyland.llm.workflow.core.base.manager.cache.CacheStore;
import com.seaskyland.llm.workflow.core.base.manager.cache.RedissonCacheStore;
import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
import org.redisson.Redisson;
import org.redisson.api.RedissonClient;
import org.redisson.codec.JsonJacksonCodec;
import org.redisson.config.Config;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnExpression;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import static com.seaskyland.llm.workflow.core.config.CacheConfigProperties.CACHE_TYPE_PREFIX;
import static com.seaskyland.llm.workflow.core.config.CacheConfigProperties.REDIS;
import static com.seaskyland.llm.workflow.core.config.MqConfigProperties.REDISSON;

/**
 * Redisson configuration.
 *
 * <p>This configuration is only loaded when Redis is actually needed, i.e. when either:
 * <ul>
 *   <li>{@code cache.type=redis} (or not configured – defaults to Redis), <strong>or</strong></li>
 *   <li>{@code mq.type=REDISSON} (Redisson-based message queue).</li>
 * </ul>
 *
 * <p>If you set {@code cache.type=jvm} <em>and</em> use a non-Redisson MQ, this config
 * class is skipped entirely and no Redis connection is established.
 *
 * @since 1.0.0.3
 */
@Configuration
@ConditionalOnExpression(
		"'${" + "cache.type" + ":" + REDIS + "}'.equals('" + REDIS + "') "
		+ "or '${" + "mq.type" + ":" + REDISSON + "}'.equalsIgnoreCase('" + REDISSON + "')"
)
public class RedissonConfig {

	@Value("${spring.data.redis.host}")
	private String host;

	@Value("${spring.data.redis.port:6379}")
	private Integer port;

	@Value("${spring.data.redis.password:}")
	private String password;

	@Value("${spring.data.redis.database:0}")
	private Integer database;

	/**
	 * Creates and configures a {@link RedissonClient} bean.
	 */
	@Bean
	public RedissonClient redissonClient() {
		Config config = new Config();

		// Use custom Jackson codec shared with the rest of the application
		var codec = new JsonJacksonCodec(JsonUtils.getObjectMapper());
		config.setCodec(codec);

		config.useSingleServer().setAddress("redis://" + host + ":" + port).setDatabase(database);

		if (password != null && !password.isEmpty()) {
			config.useSingleServer().setPassword(password);
		}

		return Redisson.create(config);
	}

	/**
	 * Exposes a {@link CacheStore} backed by Redisson.
	 * Only registered when {@code cache.type=REDIS} (the default).
	 */
	@Bean
	@ConditionalOnProperty(name = CACHE_TYPE_PREFIX, havingValue = REDIS, matchIfMissing = true)
	public CacheStore redissonCacheStore(RedissonClient redissonClient) {
		return new RedissonCacheStore(redissonClient);
	}

}
