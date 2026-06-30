package com.seaskyland.llm.workflow.core.base.mq;

import java.util.concurrent.CompletableFuture;

public interface MqProducerHandler<INPUT, OUTPUT> {

  OUTPUT send(INPUT var1);

  CompletableFuture<OUTPUT> sendAsync(INPUT var1);
}
