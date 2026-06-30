package com.seaskyland.llm.workflow.core.base.mq.rocketmq;

import com.seaskyland.llm.workflow.core.base.mq.MqMessage;
import com.seaskyland.llm.workflow.core.base.mq.MqProducer;
import com.seaskyland.llm.workflow.core.base.mq.SendCallback;
import com.seaskyland.llm.workflow.core.base.mq.SendResult;
import com.seaskyland.llm.workflow.core.config.MqConfigProperties;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;
import lombok.extern.slf4j.Slf4j;
import org.apache.rocketmq.client.apis.ClientConfiguration;
import org.apache.rocketmq.client.apis.ClientException;
import org.apache.rocketmq.client.apis.ClientServiceProvider;
import org.apache.rocketmq.client.apis.message.Message;
import org.apache.rocketmq.client.apis.message.MessageBuilder;
import org.apache.rocketmq.client.apis.producer.Producer;
import org.apache.rocketmq.client.apis.producer.SendReceipt;
import org.springframework.util.CollectionUtils;

@Slf4j
public class RocketMqProducer implements MqProducer {
  private final Producer producer;
  private final ClientServiceProvider provider;
  private final MqConfigProperties mqConfigProperties;
  private final String topic;

  public RocketMqProducer(
      MqConfigProperties mqConfigProperties,
      ClientConfiguration clientConfiguration,
      String topic) {
    this.topic = topic;
    this.mqConfigProperties = mqConfigProperties;
    ClientServiceProvider provider = ClientServiceProvider.loadService();
    try {
      this.producer =
          provider
              .newProducerBuilder()
              .setTopics(this.topic)
              .setMaxAttempts(mqConfigProperties.getMaxAttempts())
              .setClientConfiguration(clientConfiguration)
              .build();
      this.provider = ClientServiceProvider.loadService();
    } catch (ClientException e) {
      log.error("Failed to create RocketMQ producer", e);
      throw new RuntimeException(e);
    }
  }

  /**
   * Sends a message synchronously
   *
   * @param message Message to be sent
   * @return Send result containing message ID
   */
  @Override
  public SendResult send(MqMessage message) {
    try {
      SendReceipt sendReceipt = this.producer.send(buildMessage(message));
      return SendResult.builder().messageId(sendReceipt.getMessageId().toString()).build();
    } catch (Exception e) {
      log.error("Failed to send message, topic: {}", message.getTopic(), e);
      throw new RuntimeException("Failed to send message", e);
    }
  }

  /**
   * Sends messages asynchronously with callback
   *
   * @param message message to be sent
   */
  @Override
  public void sendAsync(MqMessage message, SendCallback callback) {
    CompletableFuture<SendReceipt> future = this.producer.sendAsync(buildMessage(message));
    SendReceipt sendReceipt = null;
    try {
      sendReceipt = future.get(mqConfigProperties.getSendMessageTimeoutMs(), TimeUnit.MILLISECONDS);
      callback.onSuccess(
          SendResult.builder().messageId(sendReceipt.getMessageId().toString()).build());
    } catch (Exception e) {
      callback.onError(e);
      throw new RuntimeException(e);
    }
  }

  @Override
  public SendResult sendDelay(MqMessage message, int delaySeconds) {
    long deliveryTimestamp = System.currentTimeMillis() + (delaySeconds * 1000L);
    message.setDeliveryTimestamp(deliveryTimestamp);
    SendReceipt sendReceipt = null;
    try {
      sendReceipt = producer.send(buildMessage(message));
      return SendResult.builder().messageId(sendReceipt.getMessageId().toString()).build();
    } catch (Exception e) {
      log.error(
          "Failed to send delayed message, topic: {}, delay: {}s",
          message.getTopic(),
          delaySeconds,
          e);
      throw new RuntimeException("Failed to send delayed message", e);
    }
  }

  /**
   * Builds a RocketMQ message from MqMessage
   *
   * @param message Source message
   * @return Built RocketMQ message
   */
  private Message buildMessage(MqMessage message) {
    MessageBuilder msg =
        this.provider
            .newMessageBuilder()
            .setTopic(message.getTopic())
            .setKeys(message.getKeys() == null ? null : message.getKeys().toArray(new String[] {}))
            .setBody(message.getBody().getBytes())
            .setTag(message.getTag());

    if (message.getDeliveryTimestamp() != null) {
      msg.setDeliveryTimestamp(message.getDeliveryTimestamp());
    }

    if (!CollectionUtils.isEmpty(message.getProperties())) {
      for (Map.Entry<String, String> entry : message.getProperties().entrySet()) {
        msg.addProperty(entry.getKey(), entry.getValue());
      }
    }

    return msg.build();
  }
}
