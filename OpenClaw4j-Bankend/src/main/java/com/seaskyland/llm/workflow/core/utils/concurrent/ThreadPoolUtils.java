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

import com.google.common.util.concurrent.ThreadFactoryBuilder;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;

/**
 * Utility class for managing thread pools in the application.
 *
 * @since 1.0.0.3
 */
public class ThreadPoolUtils {

  /** Default name for the task executor thread pool */
  private static final String DEFAULT_TASK_EXECUTOR_NAME = "default-task-executor";

  /**
   * Default task executor with queue size 1024, thread count 100-200, and fallback execution policy
   */
  public static final ExecutorService DEFAULT_TASK_EXECUTOR =
      new RequestContextThreadPoolWrapper(
          new ThreadPoolExecutor(
              100,
              200,
              120,
              TimeUnit.SECONDS,
              new LinkedBlockingQueue<>(1024),
              new ThreadFactoryBuilder()
                  .setNameFormat(DEFAULT_TASK_EXECUTOR_NAME + "-%d")
                  .setDaemon(true)
                  .build()));

  /** Thread pool names for workflow execution */
  private static final String TASK_EXECUTOR_NAME = "WorkflowTaskExecutor";

  private static final String NODE_EXECUTOR_NAME = "WorkflowNodeExecutor";

  private static final int NODE_EXECUTOR_MAX_CONCURRENCY = 200;

  /** Thread pool for workflow task execution with queue size 100 and caller-runs policy */
  public static final ExecutorService taskExecutorService =
      new RequestContextThreadPoolWrapper(
          new ThreadPoolExecutor(
              100,
              200,
              120,
              TimeUnit.SECONDS,
              new LinkedBlockingQueue<>(100),
              new ThreadFactoryBuilder()
                  .setNameFormat(TASK_EXECUTOR_NAME + "-%d")
                  .setDaemon(true)
                  .build(),
              new ThreadPoolExecutor.CallerRunsPolicy()));

  /**
   * Workflow 节点执行池。
   *
   * <p>节点执行经常包含 LLM、HTTP、DB、sleep 等阻塞等待，使用虚拟线程减少平台线程占用；同时保留显式并发上限，避免
   * 无限制提交把外部依赖打满。
   */
  public static final ExecutorService nodeExecutorService =
      new RequestContextThreadPoolWrapper(
          new BoundedExecutorService(
              Executors.newThreadPerTaskExecutor(
                  Thread.ofVirtual().name(NODE_EXECUTOR_NAME + "-", 0).factory()),
              NODE_EXECUTOR_MAX_CONCURRENCY));

  /** Thread pool for plugin execution with queue size 50 and thread count 40-50 */
  public static final String TOOL_TASK_EXECUTOR_NAME = "tool-task-executor";

  public static final ExecutorService TOOL_TASK_EXECUTOR =
      new ThreadPoolExecutor(
          40,
          50,
          120,
          TimeUnit.SECONDS,
          new LinkedBlockingQueue<>(50),
          new ThreadFactoryBuilder()
              .setNameFormat(TOOL_TASK_EXECUTOR_NAME + "-%d")
              .setDaemon(true)
              .build());
}
