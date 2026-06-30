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

import com.seaskyland.llm.workflow.core.base.mq.MqMessage;
import com.seaskyland.llm.workflow.core.base.mq.MqProducer;
import com.seaskyland.llm.workflow.core.base.mq.SendCallback;
import com.seaskyland.llm.workflow.core.base.mq.SendResult;
import java.util.UUID;
import lombok.extern.slf4j.Slf4j;

/**
 * JVM in-process {@link MqProducer} implementation backed by {@link JvmMessageBus}.
 *
 * <p>All three send methods (sync, async, delayed) are fully supported without any external broker
 * dependency.
 *
 * @since 1.0.0.3
 */
@Slf4j
public class JvmMqProducer implements MqProducer {

  private final JvmMessageBus messageBus;

  public JvmMqProducer(JvmMessageBus messageBus) {
    this.messageBus = messageBus;
  }

  /** Publishes the message synchronously and returns after all subscribers have been notified. */
  @Override
  public SendResult send(MqMessage message) {
    ensureMessageId(message);
    log.debug("[JvmMQ] send topic='{}' messageId='{}'", message.getTopic(), message.getMessageId());
    messageBus.publish(message);
    return SendResult.builder().messageId(message.getMessageId()).build();
  }

  /**
   * Publishes the message asynchronously; returns immediately without waiting for delivery. The
   * supplied {@code callback} is invoked upon completion or failure.
   */
  @Override
  public void sendAsync(MqMessage message, SendCallback callback) {
    ensureMessageId(message);
    log.debug(
        "[JvmMQ] sendAsync topic='{}' messageId='{}'", message.getTopic(), message.getMessageId());
    messageBus.publishAsync(message, callback);
  }

  /** Schedules the message to be delivered after {@code delaySeconds} seconds. */
  @Override
  public SendResult sendDelay(MqMessage message, int delaySeconds) {
    ensureMessageId(message);
    log.debug(
        "[JvmMQ] sendDelay topic='{}' delay={}s messageId='{}'",
        message.getTopic(),
        delaySeconds,
        message.getMessageId());
    messageBus.publishDelay(message, delaySeconds);
    return SendResult.builder().messageId(message.getMessageId()).build();
  }

  // -------------------------------------------------------------------------
  // Helpers
  // -------------------------------------------------------------------------

  /** Assigns a random UUID as messageId if the message does not already have one. */
  private static void ensureMessageId(MqMessage message) {
    if (message.getMessageId() == null || message.getMessageId().isBlank()) {
      message.setMessageId(UUID.randomUUID().toString());
    }
  }
}
