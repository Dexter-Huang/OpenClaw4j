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

package com.seaskyland.llm.workflow.core.base.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import java.util.Date;
import lombok.Data;

/** Skill 主表实体，保存可被 Agent 选择的 Skill 包入口信息。 */
@Data
@TableName("skill")
public class SkillEntity {

  @TableId(value = "id", type = IdType.AUTO)
  private Long id;

  @TableField("gmt_create")
  private Date gmtCreate;

  @TableField("gmt_modified")
  private Date gmtModified;

  @TableField("skill_code")
  private String skillCode;

  @TableField("workspace_id")
  private String workspaceId;

  @TableField("account_id")
  private String accountId;

  private String name;

  private String description;

  private String source;

  private Integer status;

  private String tags;

  private String creator;

  private String modifier;

  @TableField("tenant_id")
  private String tenantId;
}
