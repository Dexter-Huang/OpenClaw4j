package com.seaskyland.llm.workflow.core.base.mq;

import java.util.concurrent.CompletableFuture;

public interface MqProducer {
    SendResult send(MqMessage message);
    void sendAsync(MqMessage message, SendCallback callback);
    SendResult sendDelay(MqMessage message, int delaySeconds);
}
