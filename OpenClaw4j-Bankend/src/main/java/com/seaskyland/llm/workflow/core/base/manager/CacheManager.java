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

package com.seaskyland.llm.workflow.core.base.manager;

import com.seaskyland.llm.workflow.core.base.entity.LimitEntity;
import com.seaskyland.llm.workflow.core.base.manager.cache.CacheStore;
import com.seaskyland.llm.workflow.core.base.manager.cache.RedissonCacheStore;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.redisson.api.RSet;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.time.Duration;
import java.util.ArrayList;
import java.util.List;
import java.util.Objects;

/**
 * Unified cache manager that delegates all operations to the active {@link CacheStore}.
 *
 * <p>The backing store is selected via {@code cache.type} in {@code application.yml}:
 * <ul>
 *   <li>{@code cache.type=redis} (default) – uses Redis / Redisson</li>
 *   <li>{@code cache.type=jvm} – uses JVM in-memory storage (no Redis required)</li>
 * </ul>
 *
 * <p>All public methods preserve the original API so existing callers require no changes.
 *
 * @since 1.0.0.3
 */
@Slf4j
@Component
public class CacheManager {

	/** Default TTL for key-value entries (24 hours). */
	private static final Duration DEFAULT_MAX_TTL = Duration.ofHours(24);

	/** The active cache store backend (Redis or JVM). */
	private final CacheStore cacheStore;

	/** Application name used as prefix for all Redis/cache keys. */
	@Value("${spring.application.name}")
	private String prefix;

	public CacheManager(CacheStore cacheStore) {
		this.cacheStore = cacheStore;
	}

	// -------------------------------------------------------------------------
	// Key-Value operations
	// -------------------------------------------------------------------------

	/**
	 * Stores a value with the default TTL (24 hours).
	 */
	public <V> void put(String key, V value) {
		cacheStore.put(getPrefix() + key, value, DEFAULT_MAX_TTL);
	}

	/**
	 * Stores a value with a custom TTL.
	 */
	public <V> void put(String key, V value, Duration duration) {
		cacheStore.put(getPrefix() + key, value, duration);
	}

	/**
	 * Retrieves a value; returns {@code null} if absent or expired.
	 */
	public <V> V get(String key) {
		return cacheStore.get(getPrefix() + key);
	}

	/**
	 * Deletes a key. Returns {@code true} if the key existed.
	 */
	public boolean delete(String key) {
		return cacheStore.delete(getPrefix() + key);
	}

	/**
	 * Deletes multiple keys. Returns the number of keys actually removed.
	 */
	public long delete(List<String> keys) {
		if (keys == null || keys.isEmpty()) {
			return 0;
		}
		List<String> prefixedKeys = new ArrayList<>(keys.size());
		for (String key : keys) {
			prefixedKeys.add(getPrefix() + key);
		}
		return cacheStore.delete(prefixedKeys);
	}

	/**
	 * Returns {@code true} if the key exists and has not expired.
	 */
	public <V> boolean exists(String key) {
		return cacheStore.exists(getPrefix() + key);
	}

	// -------------------------------------------------------------------------
	// Atomic counter operations
	// -------------------------------------------------------------------------

	/**
	 * Atomically increments and returns the counter value. Key is used without prefix
	 * (consistent with the original behaviour).
	 */
	public long incrementAndGet(String key) {
		return cacheStore.incrementAndGet(key);
	}

	/**
	 * Atomically increments the counter, sets its TTL, and returns the new value.
	 * Key is used without prefix.
	 */
	public long incrementExpire(String key, Duration duration) {
		return cacheStore.incrementExpire(key, duration);
	}

	/**
	 * Returns the current counter value (0 if absent). Key is used without prefix.
	 */
	public long getIncrement(String key) {
		return cacheStore.getIncrement(key);
	}

	/**
	 * Increments a counter that is backed by a {@link LimitEntity} with time-based
	 * expiry. If the key doesn't exist yet, it is initialised with count=1 and the
	 * specified TTL.
	 *
	 * <p>Unlike the plain {@link #incrementAndGet(String)} method, this variant uses the
	 * key-value store and applies the application-name prefix.
	 */
	public long incrementAndGet(String key, long timeMsToLive) {
		String newKey = getPrefix() + key;
		if (Objects.isNull(cacheStore.<LimitEntity>get(newKey))) {
			LimitEntity limitEntity = new LimitEntity();
			limitEntity.setCount(1);
			long remainTime = System.currentTimeMillis() + timeMsToLive;
			limitEntity.setTime(remainTime);
			cacheStore.put(newKey, limitEntity, Duration.ofMillis(timeMsToLive));
			return 1;
		}
		else {
			LimitEntity entity = cacheStore.get(newKey);
			int count = entity.getCount();
			entity.setCount(count + 1);
			long remainTimeAt = entity.getTime();
			long currentTime = System.currentTimeMillis();
			cacheStore.put(newKey, entity, Duration.ofMillis(remainTimeAt - currentTime));
			return count + 1;
		}
	}

	/**
	 * Atomically decrements and returns the counter value. Key is used without prefix.
	 */
	public long decrementAndGet(String key) {
		return cacheStore.decrementAndGet(key);
	}

	// -------------------------------------------------------------------------
	// Set operations
	// -------------------------------------------------------------------------

	/**
	 * Returns the size of the set stored at {@code key}. Key is used without prefix.
	 */
	public int getSetSize(String key) {
		return cacheStore.getSetSize(key);
	}

	/**
	 * Returns the Redisson {@link RSet} for {@code key}, or {@code null} when running in
	 * JVM mode. Key is used without prefix.
	 *
	 * <p><strong>Note:</strong> This method is only fully supported when
	 * {@code cache.type=redis}. In JVM mode it returns {@code null}; use the typed helper
	 * methods ({@link #addSet}, {@link #containsInSet}, etc.) instead.
	 */
	public <V> RSet<V> getSet(String key) {
		if (StringUtils.isEmpty(key)) {
			return null;
		}
		if (cacheStore instanceof RedissonCacheStore) {
			return ((RedissonCacheStore) cacheStore).getSet(key);
		}
		log.warn("getSet() is only supported in Redis mode (cache.type=redis). Returning null for key: {}", key);
		return null;
	}

	/**
	 * Adds a single element to the set at {@code key}. Key is used without prefix.
	 */
	public <V> void addSet(String key, V uniqueId) {
		cacheStore.addSet(key, uniqueId);
	}

	/**
	 * Adds multiple elements to the set at {@code key}. Key is used without prefix.
	 */
	public <V> void addSet(String key, List<V> uniqueIds) {
		cacheStore.addSet(key, uniqueIds);
	}

	/**
	 * Adds an element only when the set size is below {@code fixedSize}. Key is used
	 * without prefix.
	 */
	public <V> boolean addSet(String key, V uniqueId, int fixedSize) {
		return cacheStore.addSet(key, uniqueId, fixedSize);
	}

	/**
	 * Returns {@code true} if {@code uniqueId} exists in the set. Key is used without
	 * prefix.
	 */
	public <V> boolean containsInSet(String key, V uniqueId) {
		return cacheStore.containsInSet(key, uniqueId);
	}

	/**
	 * Removes {@code uniqueId} from the set. Key is used without prefix.
	 */
	public <V> boolean removeSet(String key, V uniqueId) {
		return cacheStore.removeSet(key, uniqueId);
	}

	/**
	 * Returns {@code true} if the set is empty or does not exist. Key is used without
	 * prefix.
	 */
	public <V> boolean isEmptySet(String key) {
		return cacheStore.isEmptySet(key);
	}

	// -------------------------------------------------------------------------
	// Map operations
	// -------------------------------------------------------------------------

	/**
	 * Removes {@code mapKey} from the map stored at {@code key}. Key is used without
	 * prefix.
	 */
	public <V> boolean removeMapKey(String key, String mapKey) {
		return cacheStore.removeMapKey(key, mapKey);
	}

	// -------------------------------------------------------------------------
	// Lock operations
	// -------------------------------------------------------------------------

	/**
	 * Acquires a TTL-based distributed lock.
	 *
	 * @param key      lock identifier (prefix is applied internally)
	 * @param duration lock expiry duration
	 * @return {@code true} if the lock was obtained
	 */
	public boolean lock(String key, Duration duration) {
		return cacheStore.lock(getPrefix() + key, duration);
	}

	/**
	 * Acquires a lock with blocking-wait support.
	 *
	 * @param key          lock identifier (prefix is applied internally)
	 * @param lockDuration duration to hold the lock
	 * @param waitTimeout  maximum time to wait for the lock
	 * @return {@code true} if the lock was obtained within the timeout
	 */
	public boolean lockWithRedisson(String key, Duration lockDuration, Duration waitTimeout) {
		return cacheStore.lockWithRedisson(getPrefix() + key, lockDuration, waitTimeout);
	}

	/**
	 * Releases a TTL-based lock.
	 */
	public void unlock(String key) {
		cacheStore.unlock(getPrefix() + key);
	}

	/**
	 * Releases a blocking-wait lock held by the current thread.
	 */
	public void unlockRedisson(String key) {
		cacheStore.unlockRedisson(getPrefix() + key);
	}

	// -------------------------------------------------------------------------
	// Utilities
	// -------------------------------------------------------------------------

	/**
	 * Returns the key prefix ({@code "<appName>:"}).
	 */
	public String getPrefix() {
		return prefix + ":";
	}

}
