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

package com.seaskyland.llm.workflow.core.config;

import lombok.Data;
import org.springframework.boot.context.properties.ConfigurationProperties;

/** 工作流脚本沙箱 HTTP 服务配置。 */
@ConfigurationProperties(prefix = "sandbox")
@Data
public class SandboxProperties {

  /** 是否启用外部 Rust sandbox 服务执行 Python 和 JavaScript。 */
  private boolean enabled = true;

  /** Rust sandbox 服务基础地址。 */
  private String baseUrl = "http://127.0.0.1:9010";

  /** 单次脚本执行 HTTP 和沙箱默认超时时间。 */
  private Integer timeoutMs = 30000;
}
