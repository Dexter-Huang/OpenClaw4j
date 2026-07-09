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

package com.seaskyland.llm.workflow.runtime.enums;

import lombok.Getter;

/** Skill 状态枚举，与应用/工作流的草稿、发布、编辑中状态语义保持一致。 */
@Getter
public enum SkillStatusEnum {

  /** 已删除 */
  Deleted(0, "deleted"),

  /** 草稿 */
  Draft(1, "draft"),

  /** 已发布 */
  Published(2, "published"),

  /** 已发布后继续编辑 */
  PublishedEditing(3, "published_editing");

  private final Integer code;

  private final String status;

  SkillStatusEnum(Integer code, String status) {
    this.code = code;
    this.status = status;
  }
}
