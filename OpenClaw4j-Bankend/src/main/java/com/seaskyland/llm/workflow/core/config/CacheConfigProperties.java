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

package com.seaskyland.llm.workflow.core.config;

import lombok.Data;
import lombok.Getter;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Configuration;

/**
 * Configuration properties for the cache backend.
 *
 * <p>Bind via {@code cache.*} in {@code application.yml}:
 *
 * <pre>
 * cache:
 *   type: REDIS   # or JVM
 * </pre>
 *
 * @since 1.0.0.3
 */
@Data
@Configuration
@ConfigurationProperties(prefix = "cache")
public class CacheConfigProperties {

  /** Full property key used in {@code @ConditionalOnProperty}. */
  public static final String CACHE_TYPE_PREFIX = "cache.type";

  /** String constant for the Redis backend – matches {@link CacheType#REDIS}. */
  public static final String REDIS = "REDIS";

  /** String constant for the JVM in-memory backend – matches {@link CacheType#JVM}. */
  public static final String JVM = "JVM";

  /** Active cache backend type. Defaults to {@link CacheType#REDIS}. */
  private CacheType type = CacheType.REDIS;

  /** Supported cache backend types. */
  @Getter
  public enum CacheType {

    /** Redis-backed distributed cache via Redisson (requires a running Redis instance). */
    REDIS(CacheConfigProperties.REDIS),

    /** JVM in-memory cache (no Redis required, single-node only). */
    JVM(CacheConfigProperties.JVM),
    ;

    private final String cacheType;

    CacheType(String cacheType) {
      this.cacheType = cacheType;
    }

    /**
     * Looks up a {@link CacheType} by its string name (case-sensitive).
     *
     * @param cacheType the string value (e.g. {@code "REDIS"})
     * @return the matching {@link CacheType}, or {@code null} if not found
     */
    public static CacheType getCacheType(String cacheType) {
      for (CacheType ct : CacheType.values()) {
        if (ct.name().equals(cacheType)) {
          return ct;
        }
      }
      return null;
    }
  }
}
