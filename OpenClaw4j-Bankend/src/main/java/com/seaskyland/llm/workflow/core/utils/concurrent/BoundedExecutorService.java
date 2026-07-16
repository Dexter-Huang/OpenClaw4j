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

package com.seaskyland.llm.workflow.core.utils.concurrent;

import java.util.List;
import java.util.concurrent.AbstractExecutorService;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.RejectedExecutionException;
import java.util.concurrent.Semaphore;
import java.util.concurrent.TimeUnit;

/**
 * 带并发上限的 {@link ExecutorService} 包装器。
 *
 * <p>虚拟线程本身非常轻量，但 workflow 节点会访问 LLM、HTTP、DB 等外部资源，不能因为线程成本下降就移除
 * 对下游系统的背压。这里在提交任务前获取许可，确保同一时刻实际运行的任务数受控。
 */
class BoundedExecutorService extends AbstractExecutorService {

  private final ExecutorService delegate;

  private final Semaphore permits;

  BoundedExecutorService(ExecutorService delegate, int maxConcurrency) {
    if (maxConcurrency <= 0) {
      throw new IllegalArgumentException("maxConcurrency must be greater than 0");
    }
    this.delegate = delegate;
    this.permits = new Semaphore(maxConcurrency);
  }

  @Override
  public void execute(Runnable command) {
    acquirePermit();
    try {
      delegate.execute(
          () -> {
            try {
              command.run();
            } finally {
              permits.release();
            }
          });
    } catch (RuntimeException | Error ex) {
      permits.release();
      throw ex;
    }
  }

  @Override
  public void shutdown() {
    delegate.shutdown();
  }

  @Override
  public List<Runnable> shutdownNow() {
    return delegate.shutdownNow();
  }

  @Override
  public boolean isShutdown() {
    return delegate.isShutdown();
  }

  @Override
  public boolean isTerminated() {
    return delegate.isTerminated();
  }

  @Override
  public boolean awaitTermination(long timeout, TimeUnit unit) throws InterruptedException {
    return delegate.awaitTermination(timeout, unit);
  }

  private void acquirePermit() {
    try {
      permits.acquire();
    } catch (InterruptedException e) {
      Thread.currentThread().interrupt();
      throw new RejectedExecutionException("Interrupted while waiting for executor permit", e);
    }
  }
}
