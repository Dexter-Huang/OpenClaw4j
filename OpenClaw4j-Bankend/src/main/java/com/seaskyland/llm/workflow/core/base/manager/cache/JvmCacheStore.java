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

import jakarta.annotation.PreDestroy;
import java.time.Duration;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.locks.ReentrantLock;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;

/**
 * JVM in-memory {@link CacheStore} implementation backed by {@link ConcurrentHashMap}.
 *
 * <p>Features:
 *
 * <ul>
 *   <li>TTL support – entries are lazily evicted on access and proactively cleaned every 60 seconds
 *       by a background daemon thread.
 *   <li>Atomic counters – using {@link AtomicLong} per key.
 *   <li>Set operations – using {@link ConcurrentHashMap#newKeySet()} per key.
 *   <li>Map operations – nested {@link ConcurrentHashMap} per key.
 *   <li>Lock operations – TTL-based string locks ({@code lock}/{@code unlock}) and {@link
 *       ReentrantLock}-based blocking locks ({@code lockWithRedisson}/{@code unlockRedisson}).
 * </ul>
 *
 * <p><strong>Note:</strong> This implementation is intended for single-node deployments or
 * development/testing scenarios. It does <em>not</em> support distributed locking across multiple
 * JVM instances.
 *
 * @since 1.0.0.3
 */
@Slf4j
public class JvmCacheStore implements CacheStore {

  // -------------------------------------------------------------------------
  // Storage maps
  // -------------------------------------------------------------------------

  /** General key-value store with optional TTL. */
  private final ConcurrentHashMap<String, CacheEntry> cache = new ConcurrentHashMap<>();

  /** Atomic counters (no TTL unless managed via a separate entry in {@code cache}). */
  private final ConcurrentHashMap<String, AtomicLong> atomicCounters = new ConcurrentHashMap<>();

  /** Set collections per key (no TTL – managed separately if needed). */
  private final ConcurrentHashMap<String, Set<Object>> sets = new ConcurrentHashMap<>();

  /** Nested maps per key (used by removeMapKey). */
  private final ConcurrentHashMap<String, ConcurrentHashMap<String, Object>> mapCaches =
      new ConcurrentHashMap<>();

  /** TTL-based lock store (mimics the Redis string-lock pattern). */
  private final ConcurrentHashMap<String, String> lockStore = new ConcurrentHashMap<>();

  /** ReentrantLock store for blocking-wait lock semantics. */
  private final ConcurrentHashMap<String, ReentrantLock> reentrantLocks = new ConcurrentHashMap<>();

  // -------------------------------------------------------------------------
  // Background cleaner
  // -------------------------------------------------------------------------

  private final ScheduledExecutorService cleaner;

  public JvmCacheStore() {
    cleaner =
        Executors.newSingleThreadScheduledExecutor(
            r -> {
              Thread t = new Thread(r, "jvm-cache-cleaner");
              t.setDaemon(true);
              return t;
            });
    // Proactively remove expired entries every 60 seconds
    cleaner.scheduleAtFixedRate(this::evictExpired, 60, 60, TimeUnit.SECONDS);
    log.info("JvmCacheStore initialized – cache.type=jvm (single-node, no Redis required)");
  }

  @PreDestroy
  public void destroy() {
    cleaner.shutdownNow();
  }

  // -------------------------------------------------------------------------
  // Key-Value
  // -------------------------------------------------------------------------

  @Override
  public <V> void put(String key, V value, Duration duration) {
    long expiryMs =
        (duration == null || duration.isZero() || duration.isNegative())
            ? 0L
            : System.currentTimeMillis() + duration.toMillis();
    cache.put(key, new CacheEntry(value, expiryMs));
  }

  @Override
  @SuppressWarnings("unchecked")
  public <V> V get(String key) {
    CacheEntry entry = cache.get(key);
    if (entry == null) {
      return null;
    }
    if (entry.isExpired()) {
      cache.remove(key);
      return null;
    }
    return (V) entry.value;
  }

  @Override
  public boolean delete(String key) {
    return cache.remove(key) != null;
  }

  @Override
  public long delete(List<String> keys) {
    if (keys == null || keys.isEmpty()) {
      return 0;
    }
    long removed = 0;
    for (String key : keys) {
      if (cache.remove(key) != null) {
        removed++;
      }
    }
    return removed;
  }

  @Override
  public <V> boolean exists(String key) {
    CacheEntry entry = cache.get(key);
    if (entry == null) {
      return false;
    }
    if (entry.isExpired()) {
      cache.remove(key);
      return false;
    }
    return true;
  }

  // -------------------------------------------------------------------------
  // Atomic counters
  // -------------------------------------------------------------------------

  @Override
  public long incrementAndGet(String key) {
    return atomicCounters.computeIfAbsent(key, k -> new AtomicLong(0)).incrementAndGet();
  }

  @Override
  public long incrementExpire(String key, Duration duration) {
    AtomicLong counter = atomicCounters.computeIfAbsent(key, k -> new AtomicLong(0));
    // Use a sentinel entry in the main cache to track the expiry of this counter
    if (duration != null && !duration.isZero() && !duration.isNegative()) {
      long expiryMs = System.currentTimeMillis() + duration.toMillis();
      cache.put(key + "__counter_expiry__", new CacheEntry(key, expiryMs));
    }
    return counter.incrementAndGet();
  }

  @Override
  public long getIncrement(String key) {
    AtomicLong counter = atomicCounters.get(key);
    return counter == null ? 0L : counter.get();
  }

  @Override
  public long decrementAndGet(String key) {
    return atomicCounters.computeIfAbsent(key, k -> new AtomicLong(0)).decrementAndGet();
  }

  // -------------------------------------------------------------------------
  // Set operations
  // -------------------------------------------------------------------------

  @Override
  public int getSetSize(String key) {
    if (StringUtils.isEmpty(key)) {
      return 0;
    }
    Set<Object> set = sets.get(key);
    return set == null ? 0 : set.size();
  }

  @Override
  public <V> void addSet(String key, V uniqueId) {
    if (StringUtils.isEmpty(key) || uniqueId == null) {
      return;
    }
    sets.computeIfAbsent(key, k -> ConcurrentHashMap.newKeySet()).add(uniqueId);
  }

  @Override
  public <V> void addSet(String key, List<V> uniqueIds) {
    if (StringUtils.isEmpty(key) || uniqueIds == null || uniqueIds.isEmpty()) {
      return;
    }
    sets.computeIfAbsent(key, k -> ConcurrentHashMap.newKeySet()).addAll(uniqueIds);
  }

  @Override
  public <V> boolean addSet(String key, V uniqueId, int fixedSize) {
    if (StringUtils.isEmpty(key) || uniqueId == null) {
      return false;
    }
    Set<Object> set = sets.computeIfAbsent(key, k -> ConcurrentHashMap.newKeySet());
    // Use the set itself as the monitor to ensure size-check + add is atomic
    synchronized (set) {
      if (set.size() < fixedSize) {
        set.add(uniqueId);
        return true;
      }
    }
    return false;
  }

  @Override
  public <V> boolean containsInSet(String key, V uniqueId) {
    if (StringUtils.isEmpty(key) || uniqueId == null) {
      return false;
    }
    Set<Object> set = sets.get(key);
    return set != null && set.contains(uniqueId);
  }

  @Override
  public <V> boolean removeSet(String key, V uniqueId) {
    if (StringUtils.isEmpty(key) || uniqueId == null) {
      return false;
    }
    Set<Object> set = sets.get(key);
    if (set != null) {
      set.remove(uniqueId);
    }
    return true;
  }

  @Override
  public <V> boolean isEmptySet(String key) {
    if (StringUtils.isEmpty(key)) {
      return false;
    }
    Set<Object> set = sets.get(key);
    return set == null || set.isEmpty();
  }

  // -------------------------------------------------------------------------
  // Map operations
  // -------------------------------------------------------------------------

  @Override
  public <V> boolean removeMapKey(String key, String mapKey) {
    Map<String, Object> map = mapCaches.get(key);
    if (map != null) {
      map.remove(mapKey);
    }
    return true;
  }

  // -------------------------------------------------------------------------
  // Lock operations
  // -------------------------------------------------------------------------

  /**
   * Simple TTL-based lock backed by a {@link ConcurrentHashMap}. Uses compare-and-swap semantics to
   * detect and recover from expired (dead) locks.
   */
  @Override
  public boolean lock(String key, Duration duration) {
    long now = System.currentTimeMillis();
    long expireTime = now + duration.toMillis();
    String expireTimeStr = String.valueOf(expireTime);

    // Try to insert (atomic putIfAbsent)
    String existing = lockStore.putIfAbsent(key, expireTimeStr);
    if (existing == null) {
      return true; // successfully acquired
    }

    // Check whether the existing lock has expired (deadlock recovery)
    try {
      long currentExpireTime = Long.parseLong(existing);
      if (currentExpireTime < now) {
        // Replace only if the value hasn't changed (another thread may have just taken it)
        return lockStore.replace(key, existing, expireTimeStr);
      }
    } catch (NumberFormatException e) {
      log.error("Invalid lock expiry value for key: {}", key, e);
    }
    return false;
  }

  /**
   * Blocking-wait lock using a per-key {@link ReentrantLock}.
   *
   * <p><strong>Note:</strong> The {@code lockDuration} parameter is accepted for API compatibility
   * but is NOT enforced in the JVM implementation – the lock is held until {@link
   * #unlockRedisson(String)} is explicitly called (or the JVM exits).
   */
  @Override
  public boolean lockWithRedisson(String key, Duration lockDuration, Duration waitTimeout) {
    ReentrantLock lock = reentrantLocks.computeIfAbsent(key, k -> new ReentrantLock());
    try {
      return lock.tryLock(waitTimeout.toMillis(), TimeUnit.MILLISECONDS);
    } catch (InterruptedException e) {
      Thread.currentThread().interrupt();
      log.warn("JVM lock wait interrupted for key: {}", key);
      return false;
    }
  }

  @Override
  public void unlock(String key) {
    lockStore.remove(key);
  }

  @Override
  public void unlockRedisson(String key) {
    ReentrantLock lock = reentrantLocks.get(key);
    if (lock != null && lock.isHeldByCurrentThread()) {
      lock.unlock();
    }
  }

  // -------------------------------------------------------------------------
  // Background eviction
  // -------------------------------------------------------------------------

  private void evictExpired() {
    try {
      cache.entrySet().removeIf(e -> e.getValue().isExpired());
    } catch (Exception ex) {
      log.warn("Error during JVM cache eviction", ex);
    }
  }

  // -------------------------------------------------------------------------
  // Inner class
  // -------------------------------------------------------------------------

  /** Wraps a cached value together with its expiry timestamp. */
  private static final class CacheEntry {

    final Object value;

    /** Absolute expiry time in epoch-millis. 0 means no expiry. */
    final long expiryMs;

    CacheEntry(Object value, long expiryMs) {
      this.value = value;
      this.expiryMs = expiryMs;
    }

    boolean isExpired() {
      return expiryMs > 0 && System.currentTimeMillis() > expiryMs;
    }
  }
}
