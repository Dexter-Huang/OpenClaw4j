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

import static com.seaskyland.llm.workflow.core.config.CacheConfigProperties.CACHE_TYPE_PREFIX;
import static com.seaskyland.llm.workflow.core.config.CacheConfigProperties.REDIS;
import static com.seaskyland.llm.workflow.core.config.MqConfigProperties.REDISSON;

import com.seaskyland.llm.workflow.core.base.manager.cache.CacheStore;
import com.seaskyland.llm.workflow.core.base.manager.cache.RedissonCacheStore;
import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
import java.util.Arrays;
import java.util.LinkedHashMap;
import org.redisson.Redisson;
import org.redisson.api.RedissonClient;
import org.redisson.codec.JsonJacksonCodec;
import org.redisson.config.Config;
import org.redisson.config.HostPortNatMapper;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnExpression;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/**
 * Redisson configuration.
 *
 * <p>This configuration is only loaded when Redis is actually needed, i.e. when either:
 *
 * <ul>
 *   <li>{@code cache.type=redis} (or not configured – defaults to Redis), <strong>or</strong>
 *   <li>{@code mq.type=REDISSON} (Redisson-based message queue).
 * </ul>
 *
 * <p>If you set {@code cache.type=jvm} <em>and</em> use a non-Redisson MQ, this config class is
 * skipped entirely and no Redis connection is established.
 *
 * @since 1.0.0.3
 */
@Configuration
@ConditionalOnExpression(
    "'${"
        + "cache.type"
        + ":"
        + REDIS
        + "}'.equals('"
        + REDIS
        + "') "
        + "or '${"
        + "mq.type"
        + ":"
        + REDISSON
        + "}'.equalsIgnoreCase('"
        + REDISSON
        + "')")
public class RedissonConfig {

  @Value("${spring.data.redis.host:127.0.0.1}")
  private String host;

  @Value("${spring.data.redis.port:6379}")
  private Integer port;

  @Value("${spring.data.redis.password:}")
  private String password;

  @Value("${spring.data.redis.database:0}")
  private Integer database;

  @Value("${spring.data.redis.sentinel.master:}")
  private String sentinelMaster;

  @Value("${spring.data.redis.sentinel.nodes:}")
  private String sentinelNodes;

  @Value("${spring.data.redis.sentinel.nat-map:}")
  private String sentinelNatMap;

  @Value("${spring.data.redis.sentinel.username:}")
  private String sentinelUsername;

  @Value("${spring.data.redis.sentinel.password:}")
  private String sentinelPassword;

  /** Creates and configures a {@link RedissonClient} bean. */
  @Bean
  public RedissonClient redissonClient() {
    Config config = new Config();

    // Use custom Jackson codec shared with the rest of the application
    var codec = new JsonJacksonCodec(JsonUtils.getObjectMapper());
    config.setCodec(codec);

    if (hasText(sentinelMaster) && hasText(sentinelNodes)) {
      String[] sentinelAddresses =
          Arrays.stream(sentinelNodes.split(","))
              .map(String::trim)
              .filter(RedissonConfig::hasText)
              .map(RedissonConfig::toRedisAddress)
              .toArray(String[]::new);

      var sentinelServers =
          config
              .useSentinelServers()
              .setMasterName(sentinelMaster)
              .addSentinelAddress(sentinelAddresses)
              .setCheckSentinelsList(false)
              .setSentinelsDiscovery(false)
              .setDatabase(database);

      if (hasText(password)) {
        sentinelServers.setPassword(password);
      }
      if (hasText(sentinelUsername)) {
        sentinelServers.setSentinelUsername(sentinelUsername);
      }
      if (hasText(sentinelPassword)) {
        sentinelServers.setSentinelPassword(sentinelPassword);
      }
      if (hasText(sentinelNatMap)) {
        HostPortNatMapper natMapper = new HostPortNatMapper();
        natMapper.setHostsPortMap(parseNatMap(sentinelNatMap));
        sentinelServers.setNatMapper(natMapper);
      }
    } else {
      var singleServer =
          config
              .useSingleServer()
              .setAddress(toRedisAddress(host + ":" + port))
              .setDatabase(database);

      if (hasText(password)) {
        singleServer.setPassword(password);
      }
    }

    return Redisson.create(config);
  }

  private static boolean hasText(String value) {
    return value != null && !value.isBlank();
  }

  private static String toRedisAddress(String address) {
    if (address.startsWith("redis://") || address.startsWith("rediss://")) {
      return address;
    }
    return "redis://" + address;
  }

  private static LinkedHashMap<String, String> parseNatMap(String value) {
    LinkedHashMap<String, String> hostsPortMap = new LinkedHashMap<>();
    Arrays.stream(value.split(","))
        .map(String::trim)
        .filter(RedissonConfig::hasText)
        .forEach(
            entry -> {
              String[] mapping = entry.split("=", 2);
              if (mapping.length == 2 && hasText(mapping[0]) && hasText(mapping[1])) {
                hostsPortMap.put(mapping[0].trim(), mapping[1].trim());
              }
            });
    return hostsPortMap;
  }

  /**
   * Exposes a {@link CacheStore} backed by Redisson. Only registered when {@code cache.type=REDIS}
   * (the default).
   */
  @Bean
  @ConditionalOnProperty(name = CACHE_TYPE_PREFIX, havingValue = REDIS, matchIfMissing = true)
  public CacheStore redissonCacheStore(RedissonClient redissonClient) {
    return new RedissonCacheStore(redissonClient);
  }
}
