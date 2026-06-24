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

package com.seaskyland.llm.workflow.core.base.mq;

import com.seaskyland.llm.workflow.core.config.MqConfigProperties;
import jakarta.annotation.PreDestroy;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.apache.rocketmq.client.apis.ClientConfiguration;
import org.apache.rocketmq.client.apis.ClientException;
import org.apache.rocketmq.client.apis.ClientServiceProvider;
import org.apache.rocketmq.client.apis.consumer.ConsumeResult;
import org.apache.rocketmq.client.apis.consumer.FilterExpression;
import org.apache.rocketmq.client.apis.consumer.PushConsumer;
import org.apache.rocketmq.client.apis.message.MessageView;
import org.springframework.stereotype.Component;
import org.springframework.util.CollectionUtils;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * MQ Consumer Manager Manages multiple consumers and provides message consumption
 * capabilities
 *
 * @since 1.0.0.3
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class MqConsumerManager {
	/**
	 * Subscribe to a topic with specified consumer group
	 * @param group consumer group name
	 * @param topic topic to subscribe
	 * @param handler message handler for processing messages
	 */
	public void subscribe(MqConsumer mqConsumer, String group, String topic, MqConsumerHandler<MqMessage> handler) {
		mqConsumer.subscribe(group, topic, handler);
	}



}
