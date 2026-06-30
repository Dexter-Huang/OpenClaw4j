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

package com.seaskyland.llm.workflow.core.base.mq.jvm;

import com.seaskyland.llm.workflow.core.base.mq.MqConsumerHandler;
import com.seaskyland.llm.workflow.core.base.mq.MqMessage;
import com.seaskyland.llm.workflow.core.base.mq.SendCallback;
import com.seaskyland.llm.workflow.core.base.mq.SendResult;
import jakarta.annotation.PreDestroy;
import java.util.List;
import java.util.UUID;
import java.util.concurrent.*;
import lombok.extern.slf4j.Slf4j;

/**
 * In-process publish/subscribe message bus backed entirely by JVM memory.
 *
 * <p>This bus is the shared infrastructure used by {@link JvmMqProducer} and {@link JvmMqConsumer}.
 * It provides:
 *
 * <ul>
 *   <li><b>Synchronous publish</b> – delivered inline on the calling thread.
 *   <li><b>Asynchronous publish</b> – dispatched to a cached thread pool.
 *   <li><b>Delayed publish</b> – scheduled via a single-thread scheduler.
 * </ul>
 *
 * <p><strong>Note:</strong> This implementation is single-node only. Messages are never persisted
 * and are lost on JVM restart or application crash.
 *
 * @since 1.0.0.3
 */
@Slf4j
public class JvmMessageBus {

  /**
   * topic → all registered handlers for that topic. Uses {@link CopyOnWriteArrayList} so that
   * iteration during dispatch is safe even if handlers are added concurrently.
   */
  private final ConcurrentHashMap<String, CopyOnWriteArrayList<MqConsumerHandler<MqMessage>>>
      subscribers = new ConcurrentHashMap<>();

  /** Thread pool used for async message dispatch. */
  private final ExecutorService asyncExecutor =
      Executors.newCachedThreadPool(
          r -> {
            Thread t = new Thread(r, "jvm-mq-async");
            t.setDaemon(true);
            return t;
          });

  /** Scheduler used for delayed message delivery. */
  private final ScheduledExecutorService scheduler =
      Executors.newSingleThreadScheduledExecutor(
          r -> {
            Thread t = new Thread(r, "jvm-mq-delay");
            t.setDaemon(true);
            return t;
          });

  // -------------------------------------------------------------------------
  // Subscription
  // -------------------------------------------------------------------------

  /**
   * Registers {@code handler} to receive messages published on {@code topic}. Multiple handlers per
   * topic are supported.
   */
  public void subscribe(String topic, MqConsumerHandler<MqMessage> handler) {
    subscribers.computeIfAbsent(topic, k -> new CopyOnWriteArrayList<>()).add(handler);
    log.info("[JvmMQ] subscribed to topic '{}'", topic);
  }

  // -------------------------------------------------------------------------
  // Publish
  // -------------------------------------------------------------------------

  /**
   * Publishes {@code message} synchronously. All handlers registered for the message's topic are
   * invoked on the calling thread before this method returns.
   *
   * @return number of handlers notified
   */
  public int publish(MqMessage message) {
    List<MqConsumerHandler<MqMessage>> handlers = subscribers.get(message.getTopic());
    if (handlers == null || handlers.isEmpty()) {
      log.debug("[JvmMQ] no subscribers for topic '{}', message dropped", message.getTopic());
      return 0;
    }
    int count = 0;
    for (MqConsumerHandler<MqMessage> handler : handlers) {
      try {
        handler.handle(message);
        count++;
      } catch (Exception e) {
        log.error("[JvmMQ] handler error on topic '{}': {}", message.getTopic(), e.getMessage(), e);
      }
    }
    return count;
  }

  /**
   * Publishes {@code message} asynchronously. The caller is not blocked; delivery happens on the
   * internal thread pool. {@code callback} is notified after all handlers have been called (or if
   * an error occurs).
   */
  public void publishAsync(MqMessage message, SendCallback callback) {
    asyncExecutor.submit(
        () -> {
          try {
            publish(message);
            if (callback != null) {
              callback.onSuccess(buildResult(message));
            }
          } catch (Exception e) {
            log.error(
                "[JvmMQ] async publish error on topic '{}': {}",
                message.getTopic(),
                e.getMessage(),
                e);
            if (callback != null) {
              callback.onError(e);
            }
          }
        });
  }

  /** Schedules {@code message} to be published after {@code delaySeconds} seconds. */
  public void publishDelay(MqMessage message, int delaySeconds) {
    scheduler.schedule(() -> publish(message), delaySeconds, TimeUnit.SECONDS);
    log.debug(
        "[JvmMQ] delayed message queued for topic '{}', delay={}s",
        message.getTopic(),
        delaySeconds);
  }

  // -------------------------------------------------------------------------
  // Lifecycle
  // -------------------------------------------------------------------------

  @PreDestroy
  public void shutdown() {
    log.info("[JvmMQ] shutting down message bus");
    asyncExecutor.shutdown();
    scheduler.shutdown();
  }

  // -------------------------------------------------------------------------
  // Helpers
  // -------------------------------------------------------------------------

  private static SendResult buildResult(MqMessage message) {
    String id =
        message.getMessageId() != null ? message.getMessageId() : UUID.randomUUID().toString();
    return SendResult.builder().messageId(id).build();
  }
}
