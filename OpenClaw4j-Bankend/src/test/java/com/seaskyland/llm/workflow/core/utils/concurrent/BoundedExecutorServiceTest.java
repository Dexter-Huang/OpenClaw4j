package com.seaskyland.llm.workflow.core.utils.concurrent;

import static org.assertj.core.api.Assertions.assertThat;

import com.seaskyland.llm.workflow.core.context.RequestContextHolder;
import com.seaskyland.llm.workflow.runtime.domain.RequestContext;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;
import org.junit.jupiter.api.Test;

class BoundedExecutorServiceTest {

  @Test
  void limitsConcurrentTasks() throws Exception {
    try (BoundedExecutorService executor =
        new BoundedExecutorService(Executors.newVirtualThreadPerTaskExecutor(), 2)) {
      AtomicInteger running = new AtomicInteger();
      AtomicInteger maxRunning = new AtomicInteger();
      List<Future<?>> futures = new ArrayList<>();

      for (int i = 0; i < 4; i++) {
        futures.add(
            executor.submit(
                () -> {
                  int current = running.incrementAndGet();
                  maxRunning.accumulateAndGet(current, Math::max);
                  try {
                    Thread.sleep(100);
                  } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                    throw new RuntimeException(e);
                  } finally {
                    running.decrementAndGet();
                  }
                }));
      }

      for (Future<?> future : futures) {
        future.get(1, TimeUnit.SECONDS);
      }

      assertThat(maxRunning).hasValueLessThanOrEqualTo(2);
    }
  }

  @Test
  void threadPoolUtilsRunsWorkflowNodesOnVirtualThreads() throws Exception {
    Future<Boolean> future =
        ThreadPoolUtils.nodeExecutorService.submit(() -> Thread.currentThread().isVirtual());

    assertThat(future.get(1, TimeUnit.SECONDS)).isTrue();
  }

  @Test
  void threadPoolUtilsPropagatesRequestContextToVirtualThreads() throws Exception {
    RequestContext context = new RequestContext();
    context.setAccountId("account-virtual-thread");

    String accountId =
        RequestContextHolder.callWithRequestContext(
            context,
            () ->
                ThreadPoolUtils.nodeExecutorService
                    .submit(() -> RequestContextHolder.getRequestContext().getAccountId())
                    .get(1, TimeUnit.SECONDS));

    assertThat(accountId).isEqualTo("account-virtual-thread");
  }
}
