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

import com.seaskyland.llm.workflow.core.base.mq.MqConsumer;
import com.seaskyland.llm.workflow.core.base.mq.MqConsumerHandler;
import com.seaskyland.llm.workflow.core.base.mq.MqMessage;
import lombok.extern.slf4j.Slf4j;

/**
 * JVM in-process {@link MqConsumer} implementation backed by {@link JvmMessageBus}.
 *
 * <p>Subscribing a handler via {@link #subscribe(String, String, MqConsumerHandler)} registers it
 * on the shared {@link JvmMessageBus}; messages are delivered in-process without any network or
 * serialization overhead.
 *
 * @since 1.0.0.3
 */
@Slf4j
public class JvmMqConsumer implements MqConsumer {

  private final JvmMessageBus messageBus;

  public JvmMqConsumer(JvmMessageBus messageBus) {
    this.messageBus = messageBus;
  }

  /**
   * Registers {@code handler} to receive messages for the given {@code topic}. The {@code group}
   * parameter is accepted for API compatibility but is not used in the JVM implementation (there is
   * no consumer group concept in-process).
   *
   * @param group consumer group name (ignored in JVM mode)
   * @param topic topic to subscribe to
   * @param handler handler invoked for every message on {@code topic}
   */
  @Override
  public void subscribe(String group, String topic, MqConsumerHandler<MqMessage> handler) {
    log.info("[JvmMQ] subscribing to topic '{}' (group='{}' is ignored in JVM mode)", topic, group);
    messageBus.subscribe(topic, handler);
  }

  /**
   * No-op in JVM mode: the shared {@link JvmMessageBus} manages its own lifecycle and shuts down
   * via its own {@code @PreDestroy} hook.
   */
  @Override
  public void shutdown() {
    log.info("[JvmMQ] JvmMqConsumer shutdown (bus lifecycle managed separately)");
  }
}
