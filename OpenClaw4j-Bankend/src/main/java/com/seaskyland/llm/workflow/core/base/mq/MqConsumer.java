package com.seaskyland.llm.workflow.core.base.mq;

public interface MqConsumer {

  void subscribe(String group, String topic, MqConsumerHandler<MqMessage> handler);

  void shutdown();
}
