/*
 * Copyright 2026 the original author or authors.
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

package com.seaskyland.llm.workflow.runtime.domain.skill;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import lombok.Data;

/** Agent 运行时可见的 Skill 轻量索引信息，不包含 Skill 文件正文。 */
@Data
public class SkillRuntimeInfo implements Serializable {

  @JsonProperty("skill_code")
  private String skillCode;

  private String name;

  private String description;

  @JsonProperty("main_file_path")
  private String mainFilePath;
}
