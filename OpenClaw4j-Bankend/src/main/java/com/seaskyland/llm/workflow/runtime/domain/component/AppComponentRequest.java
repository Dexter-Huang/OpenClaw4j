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

package com.seaskyland.llm.workflow.runtime.domain.component;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.seaskyland.llm.workflow.runtime.domain.BaseQuery;
import com.seaskyland.llm.workflow.runtime.domain.chat.ChatMessage;
import java.io.Serializable;
import java.util.List;
import java.util.Map;
import lombok.Data;
import lombok.EqualsAndHashCode;

/**
 * Request model for application component operations
 *
 * @author guning.lt
 * @since 1.0.0.3
 */
@Data
@EqualsAndHashCode(callSuper = true)
public class AppComponentRequest extends BaseQuery implements Serializable {

  /** Component identifier code */
  private String code;

  /** Type of the component */
  private String type;

  /** Business variables for component input parameters */
  @JsonProperty("biz_vars")
  private Map<String, Object> bizVars;

  /** Flag indicating if streaming mode is enabled */
  @JsonProperty("stream_mode")
  private Boolean streamMode = true;

  /** List of chat messages */
  @JsonProperty("messages")
  private List<ChatMessage> messages;
}
