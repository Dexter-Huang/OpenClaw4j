package com.seaskyland.llm.workflow.core.base.mq.redisson;

import com.seaskyland.llm.workflow.core.base.mq.MqConsumer;
import com.seaskyland.llm.workflow.core.base.mq.MqProducer;
import com.seaskyland.llm.workflow.core.config.MqConfigProperties;
import lombok.Data;
import org.redisson.api.RedissonClient;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Data
@Configuration
@ConditionalOnProperty(
    name = MqConfigProperties.MQ_TYPE_PREFIX,
    havingValue = MqConfigProperties.REDISSON)
public class RedissonMqConfig {

  @Bean
  public MqProducer documentIndexProducer(
      RedissonClient redissonClient, MqConfigProperties mqConfigProperties) {
    return new RedissonMqProducer(
        redissonClient, mqConfigProperties, mqConfigProperties.getDocumentIndexTopic());
  }

  @Bean
  public MqConsumer documentIndexConsumer(
      RedissonClient redissonClient, MqConfigProperties mqConfigProperties) {
    return new RedissonMqConsumer(redissonClient);
  }
}
