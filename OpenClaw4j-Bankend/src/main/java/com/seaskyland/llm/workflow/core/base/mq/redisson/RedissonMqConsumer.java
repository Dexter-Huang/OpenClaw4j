package com.seaskyland.llm.workflow.core.base.mq.redisson;

import cn.hutool.json.JSONUtil;
import com.alibaba.fastjson.JSON;
import com.seaskyland.llm.workflow.core.base.mq.MqConsumer;
import com.seaskyland.llm.workflow.core.base.mq.MqConsumerHandler;
import com.seaskyland.llm.workflow.core.base.mq.MqMessage;
import jakarta.annotation.PreDestroy;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.redisson.api.RTopic;
import org.redisson.api.RedissonClient;
import org.redisson.codec.JsonJacksonCodec;
import org.redisson.codec.SerializationCodec;


@Slf4j
public class RedissonMqConsumer implements MqConsumer {
    public static final JsonJacksonCodec JSON_CODEC = new JsonJacksonCodec();

    private final RedissonClient redissonClient;

    public RedissonMqConsumer(RedissonClient redissonClient) {
        this.redissonClient = redissonClient;
    }
    @Override
    public void subscribe(String group, String topic, MqConsumerHandler<MqMessage> handler) {
        log.info("订阅topic：{}", topic);
        RTopic rTopic = redissonClient.getTopic(topic, JSON_CODEC);
        rTopic.addListener(MqMessage.class, (charSequence, mqMessage) -> {
            if (handler != null) {
                handler.handle(mqMessage);
            }
        });
    }

    @PreDestroy
    @Override
    public void shutdown() {
        redissonClient.shutdown();
    }
}
