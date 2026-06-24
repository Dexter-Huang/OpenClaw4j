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

import lombok.Data;
import lombok.Getter;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Configuration;

/**
 * RocketMQ configuration properties for multiple producers and consumers
 *
 * @since 1.0.0.3
 */
@Data
@Configuration
@ConfigurationProperties(prefix = "mq")
public class MqConfigProperties {

    public static final String MQ_TYPE_PREFIX = "mq.type";

    public static final String ROCKET_MQ = "ROCKET_MQ";

    public static final String REDISSON = "REDISSON";

    /** JVM in-process message bus – no external broker required. */
    public static final String JVM = "JVM";


    private MqType type = MqType.ROCKET_MQ;

	/** RocketMQ server endpoints */
	private String endpoints;

	/** Maximum number of retry attempts for sending messages */
	private int maxAttempts = 1;

	/** Message sending timeout in milliseconds */
	private int sendMessageTimeoutMs = 3000;

	/** Maximum number of cached messages */
	private int maxCacheMessageCount = 1024;

	/** Maximum size of cached messages in bytes */
	private int maxCacheMessageSizeInBytes = 64 * 1024 * 1024;

	/** Number of consumer threads */
	private int consumptionThreadCount = 20;

	/** Topic for document indexing */
	private String documentIndexTopic = "topic_saa_studio_document_index";

	/** Consumer group for document indexing */
	private String documentIndexGroup = "group_saa_studio_document_index";

    @Getter
    public enum MqType {

        ROCKET_MQ(MqConfigProperties.ROCKET_MQ),
        REDISSON(MqConfigProperties.REDISSON),
        /** JVM in-process message bus – no external broker required. */
        JVM(MqConfigProperties.JVM),
        ;

        private final String mqType;


        MqType(String mqType) {
            this.mqType = mqType;
        }

        public static MqType getMqType(String mqType) {
            for (MqType mq : MqType.values()) {
                if (mq.name().equals(mqType)) {
                    return mq;
                }
            }
            return null;
        }
    }

}
