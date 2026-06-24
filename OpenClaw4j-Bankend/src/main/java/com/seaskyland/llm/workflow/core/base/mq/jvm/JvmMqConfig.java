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
import com.seaskyland.llm.workflow.core.base.mq.MqProducer;
import com.seaskyland.llm.workflow.core.config.MqConfigProperties;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/**
 * Spring configuration for the JVM in-process message queue.
 *
 * <p>Activated when {@code mq.type=JVM} is set in {@code application.yml}.
 * No external broker (Redis, RocketMQ, …) is required.
 *
 * <p><strong>Limitations:</strong>
 * <ul>
 *   <li>Messages are delivered in-process only – not visible to other JVM instances.</li>
 *   <li>Messages are not persisted; they are lost on restart or crash.</li>
 *   <li>Consumer groups are not supported (all registered handlers receive every message).</li>
 * </ul>
 *
 * @see MqConfigProperties.MqType#JVM
 * @since 1.0.0.3
 */
@Configuration
@ConditionalOnProperty(name = MqConfigProperties.MQ_TYPE_PREFIX, havingValue = MqConfigProperties.JVM)
public class JvmMqConfig {

	/**
	 * The shared in-process pub/sub bus. Both producer and consumer reference this
	 * same bean so that published messages reach registered subscribers.
	 */
	@Bean
	public JvmMessageBus jvmMessageBus() {
		return new JvmMessageBus();
	}

	/**
	 * JVM-backed {@link MqProducer} for document-index messages.
	 */
	@Bean
	public MqProducer documentIndexProducer(JvmMessageBus jvmMessageBus) {
		return new JvmMqProducer(jvmMessageBus);
	}

	/**
	 * JVM-backed {@link MqConsumer} for document-index messages.
	 */
	@Bean
	public MqConsumer documentIndexConsumer(JvmMessageBus jvmMessageBus) {
		return new JvmMqConsumer(jvmMessageBus);
	}

}
