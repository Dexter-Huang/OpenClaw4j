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

import java.time.Duration;
import java.util.List;

/**
 * Abstraction for cache storage backends. Implementations can be based on Redis (Redisson) or JVM
 * in-memory storage, selected via {@code cache.type} in {@code application.yml}.
 *
 * <p>All key parameters are received <em>as-is</em> (including any prefix already applied by the
 * caller). Implementations must NOT add extra prefix internally.
 *
 * @since 1.0.0.3
 */
public interface CacheStore {

  // -------------------------------------------------------------------------
  // Key-Value operations
  // -------------------------------------------------------------------------

  /** Stores a value with the given TTL. */
  <V> void put(String key, V value, Duration duration);

  /** Retrieves a value; returns {@code null} if missing or expired. */
  <V> V get(String key);

  /** Deletes a single key. Returns {@code true} if the key existed. */
  boolean delete(String key);

  /** Deletes multiple keys. Returns the number of keys actually removed. */
  long delete(List<String> keys);

  /** Returns {@code true} if the key exists and has not expired. */
  <V> boolean exists(String key);

  // -------------------------------------------------------------------------
  // Atomic counter operations
  // -------------------------------------------------------------------------

  /** Atomically increments the counter and returns the new value. */
  long incrementAndGet(String key);

  /** Atomically increments the counter, sets its expiry, and returns the new value. */
  long incrementExpire(String key, Duration duration);

  /** Returns the current value of the counter (0 if absent). */
  long getIncrement(String key);

  /** Atomically decrements the counter and returns the new value. */
  long decrementAndGet(String key);

  // -------------------------------------------------------------------------
  // Set operations
  // -------------------------------------------------------------------------

  /** Returns the size of the set identified by {@code key}. */
  int getSetSize(String key);

  /** Adds a single element to the set. */
  <V> void addSet(String key, V uniqueId);

  /** Adds multiple elements to the set. */
  <V> void addSet(String key, List<V> uniqueIds);

  /**
   * Adds an element only when the current set size is below {@code fixedSize}. Returns {@code true}
   * if the element was added.
   */
  <V> boolean addSet(String key, V uniqueId, int fixedSize);

  /** Returns {@code true} if the element exists in the set. */
  <V> boolean containsInSet(String key, V uniqueId);

  /** Removes an element from the set. */
  <V> boolean removeSet(String key, V uniqueId);

  /** Returns {@code true} if the set is empty or does not exist. */
  <V> boolean isEmptySet(String key);

  // -------------------------------------------------------------------------
  // Map operations
  // -------------------------------------------------------------------------

  /** Removes a map entry identified by {@code mapKey} inside the map at {@code key}. */
  <V> boolean removeMapKey(String key, String mapKey);

  // -------------------------------------------------------------------------
  // Distributed lock operations
  // -------------------------------------------------------------------------

  /** Acquires a simple TTL-based lock. Returns {@code true} if the lock was obtained. */
  boolean lock(String key, Duration duration);

  /**
   * Acquires a lock with blocking-wait semantics. Returns {@code true} if acquired within {@code
   * waitTimeout}.
   */
  boolean lockWithRedisson(String key, Duration lockDuration, Duration waitTimeout);

  /** Releases a simple TTL-based lock. */
  void unlock(String key);

  /** Releases a blocking-wait lock held by the current thread. */
  void unlockRedisson(String key);
}
