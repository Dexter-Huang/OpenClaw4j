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

import static com.seaskyland.llm.workflow.core.config.CacheConfigProperties.CACHE_TYPE_PREFIX;
import static com.seaskyland.llm.workflow.core.config.CacheConfigProperties.JVM;

import com.seaskyland.llm.workflow.core.base.manager.CacheManager;
import com.seaskyland.llm.workflow.core.base.manager.cache.CacheStore;
import com.seaskyland.llm.workflow.core.base.manager.cache.JvmCacheStore;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/**
 * Configuration for the JVM in-memory cache backend.
 *
 * <p>Activated by setting {@code cache.type=JVM} in {@code application.yml}. No Redis connection is
 * required in this mode.
 *
 * <p><strong>Limitations in JVM mode:</strong>
 *
 * <ul>
 *   <li>Cache is local to the JVM – not shared across multiple instances.
 *   <li>{@link CacheManager#getSet(String)} returns {@code null}; use the typed set helper methods
 *       instead.
 *   <li>Distributed locking ({@code lockWithRedisson}) uses {@link
 *       java.util.concurrent.locks.ReentrantLock} and is therefore <em>not</em> distributed.
 * </ul>
 *
 * @see CacheConfigProperties.CacheType#JVM
 * @since 1.0.0.3
 */
@Configuration
@ConditionalOnProperty(name = CACHE_TYPE_PREFIX, havingValue = JVM)
public class JvmCacheConfig {

  /** Registers the JVM in-memory {@link CacheStore} bean. */
  @Bean
  public CacheStore jvmCacheStore() {
    return new JvmCacheStore();
  }
}
