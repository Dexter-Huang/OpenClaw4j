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

package com.seaskyland.llm.workflow.core.base.manager.cache;

import com.seaskyland.llm.workflow.core.base.manager.CacheManager;
import java.time.Duration;
import java.util.List;
import java.util.concurrent.TimeUnit;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.redisson.api.*;

/**
 * Redis-backed {@link CacheStore} implementation using Redisson.
 *
 * <p>Keys received by every method are expected to be fully qualified (prefix already applied by
 * the caller – typically {@link CacheManager}).
 *
 * @since 1.0.0.3
 */
@Slf4j
public class RedissonCacheStore implements CacheStore {

  private final RedissonClient redissonClient;

  public RedissonCacheStore(RedissonClient redissonClient) {
    this.redissonClient = redissonClient;
  }

  // -------------------------------------------------------------------------
  // Key-Value
  // -------------------------------------------------------------------------

  @Override
  public <V> void put(String key, V value, Duration duration) {
    redissonClient.<V>getBucket(key).set(value, duration);
  }

  @Override
  public <V> V get(String key) {
    return redissonClient.<V>getBucket(key).get();
  }

  @Override
  public boolean delete(String key) {
    return redissonClient.getBucket(key).delete();
  }

  @Override
  public long delete(List<String> keys) {
    if (keys == null || keys.isEmpty()) {
      return 0;
    }
    return redissonClient.getKeys().delete(keys.toArray(new String[0]));
  }

  @Override
  public <V> boolean exists(String key) {
    return redissonClient.<V>getBucket(key).isExists();
  }

  // -------------------------------------------------------------------------
  // Atomic counters
  // -------------------------------------------------------------------------

  @Override
  public long incrementAndGet(String key) {
    return redissonClient.getAtomicLong(key).incrementAndGet();
  }

  @Override
  public long incrementExpire(String key, Duration duration) {
    RAtomicLong counter = redissonClient.getAtomicLong(key);
    counter.expireAsync(duration);
    return counter.incrementAndGet();
  }

  @Override
  public long getIncrement(String key) {
    return redissonClient.getAtomicLong(key).get();
  }

  @Override
  public long decrementAndGet(String key) {
    return redissonClient.getAtomicLong(key).decrementAndGet();
  }

  // -------------------------------------------------------------------------
  // Set operations
  // -------------------------------------------------------------------------

  @Override
  public int getSetSize(String key) {
    if (StringUtils.isEmpty(key)) {
      return 0;
    }
    try {
      return redissonClient.getSet(key).size();
    } catch (Exception e) {
      log.error("get set size error. key={}", key, e);
      return 0;
    }
  }

  @Override
  public <V> void addSet(String key, V uniqueId) {
    if (StringUtils.isEmpty(key) || uniqueId == null) {
      return;
    }
    redissonClient.<V>getSet(key).add(uniqueId);
  }

  @Override
  public <V> void addSet(String key, List<V> uniqueIds) {
    if (StringUtils.isEmpty(key) || uniqueIds == null || uniqueIds.isEmpty()) {
      return;
    }
    redissonClient.<V>getSet(key).addAll(uniqueIds);
  }

  @Override
  public <V> boolean addSet(String key, V uniqueId, int fixedSize) {
    if (StringUtils.isEmpty(key) || uniqueId == null) {
      return false;
    }
    RSet<V> rSet = redissonClient.getSet(key);
    if (rSet.size() < fixedSize) {
      rSet.add(uniqueId);
      return true;
    }
    return false;
  }

  @Override
  public <V> boolean containsInSet(String key, V uniqueId) {
    if (StringUtils.isEmpty(key) || uniqueId == null) {
      return false;
    }
    return redissonClient.<V>getSet(key).contains(uniqueId);
  }

  @Override
  public <V> boolean removeSet(String key, V uniqueId) {
    if (StringUtils.isEmpty(key) || uniqueId == null) {
      return false;
    }
    redissonClient.<V>getSet(key).remove(uniqueId);
    return true;
  }

  @Override
  public <V> boolean isEmptySet(String key) {
    if (StringUtils.isEmpty(key)) {
      return false;
    }
    return redissonClient.<V>getSet(key).isEmpty();
  }

  // -------------------------------------------------------------------------
  // Map operations
  // -------------------------------------------------------------------------

  @Override
  public <V> boolean removeMapKey(String key, String mapKey) {
    redissonClient.<String, V>getMapCache(key).remove(mapKey);
    return true;
  }

  // -------------------------------------------------------------------------
  // Lock operations
  // -------------------------------------------------------------------------

  @Override
  public boolean lock(String key, Duration duration) {
    long now = System.currentTimeMillis();
    long expireTime = now + duration.toMillis();
    String expireTimeStr = String.valueOf(expireTime);

    boolean result = redissonClient.getBucket(key).setIfAbsent(expireTimeStr, duration);
    if (result) {
      return true;
    }

    // Deadlock recovery: check if the stored lock has already expired
    String currentExpireTimeStr = (String) redissonClient.getBucket(key).get();
    if (currentExpireTimeStr != null) {
      long currentExpireTime = Long.parseLong(currentExpireTimeStr);
      if (currentExpireTime < now) {
        String lastExpireTimeStr =
            (String) redissonClient.getBucket(key).getAndSet(expireTimeStr, duration);
        return lastExpireTimeStr != null && lastExpireTimeStr.equals(currentExpireTimeStr);
      }
    }
    return false;
  }

  @Override
  public boolean lockWithRedisson(String key, Duration lockDuration, Duration waitTimeout) {
    RLock lock = redissonClient.getLock(key);
    try {
      return lock.tryLock(waitTimeout.toMillis(), lockDuration.toMillis(), TimeUnit.MILLISECONDS);
    } catch (InterruptedException e) {
      Thread.currentThread().interrupt();
      log.warn("Redisson lock wait interrupted for key: {}", key);
      return false;
    }
  }

  @Override
  public void unlock(String key) {
    redissonClient.getBucket(key).delete();
  }

  @Override
  public void unlockRedisson(String key) {
    RLock lock = redissonClient.getLock(key);
    if (lock.isHeldByCurrentThread()) {
      lock.unlock();
    }
  }

  // -------------------------------------------------------------------------
  // Extra – Redisson-specific helpers accessible via instanceof checks
  // -------------------------------------------------------------------------

  /**
   * Returns the underlying Redisson {@link RSet} for the given key. Only available in Redis mode;
   * callers should check via {@code instanceof RedissonCacheStore}.
   */
  public <V> RSet<V> getSet(String key) {
    if (StringUtils.isEmpty(key)) {
      return null;
    }
    return redissonClient.getSet(key);
  }
}
