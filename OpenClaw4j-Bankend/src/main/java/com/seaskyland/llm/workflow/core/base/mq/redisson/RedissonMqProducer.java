package com.seaskyland.llm.workflow.core.base.mq.redisson;

import cn.hutool.json.JSONUtil;
import com.seaskyland.llm.workflow.core.base.mq.MqMessage;
import com.seaskyland.llm.workflow.core.base.mq.MqProducer;
import com.seaskyland.llm.workflow.core.base.mq.SendCallback;
import com.seaskyland.llm.workflow.core.base.mq.SendResult;
import com.seaskyland.llm.workflow.core.config.MqConfigProperties;
import lombok.extern.slf4j.Slf4j;
import org.redisson.api.*;
import org.redisson.codec.JsonJacksonCodec;

import java.util.concurrent.TimeUnit;


@Slf4j
public class RedissonMqProducer implements MqProducer {
    public static final JsonJacksonCodec JSON_CODEC = new JsonJacksonCodec();
    private final RedissonClient redissonClient;
    private final MqConfigProperties mqConfigProperties;
    private final String topic;
    private RDelayedQueue<MqMessage> delayQueue;
    private RBlockingQueue<MqMessage> blockingQueue;

    public RedissonMqProducer(RedissonClient redissonClient,
                              MqConfigProperties mqConfigProperties,
                              String topic) {
        this.redissonClient = redissonClient;
        this.mqConfigProperties = mqConfigProperties;
        this.topic = topic;
        this.blockingQueue = redissonClient.getBlockingQueue(this.topic);
        this.delayQueue = redissonClient.getDelayedQueue(blockingQueue);
    }

    @Override
    public SendResult send(MqMessage message) {
        RTopic rTopic = redissonClient.getTopic(this.topic, JSON_CODEC);
        long messageId = rTopic.publish(message);
        return SendResult.builder().messageId(String.valueOf(messageId)).build();
    }

    @Override
    public void sendAsync(MqMessage message, SendCallback callback) {
        RTopic rTopic = redissonClient.getTopic(this.topic, JSON_CODEC);
        RFuture<Long> future = rTopic.publishAsync(message);
        future.whenComplete((messageId, throwable) -> {
            if (throwable != null) {
                callback.onError(throwable);
            } else {
                callback.onSuccess(SendResult.builder().messageId(String.valueOf(messageId)).build());
            }
        });

    }

    @Override
    public SendResult sendDelay(MqMessage message, int delaySeconds) {
        this.delayQueue.offer(message, delaySeconds, TimeUnit.SECONDS);
        return SendResult.builder().messageId(message.getMessageId()).build();
    }

    // TODO
    private void startDelayQueueConsumer() {
        new Thread(() -> {
            while (true) {
                try {
                    MqMessage task = blockingQueue.take();
                    log.info("接收到延迟任务:{}", task);
                } catch (Exception e) {
                    e.printStackTrace();
                }
            }
        }, "SANYOU-Consumer").start();
    }
}