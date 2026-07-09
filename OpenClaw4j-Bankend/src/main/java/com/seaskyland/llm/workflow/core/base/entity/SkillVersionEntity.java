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

/** Skill 版本实体，隔离 Skill 元信息和可执行文件包的版本生命周期。 */
@Data
@TableName("skill_version")
public class SkillVersionEntity {

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

  private String version;

  private String description;

  @TableField("main_file_path")
  private String mainFilePath;

  private String manifest;

  @TableField("content_hash")
  private String contentHash;

  @TableField("storage_type")
  private String storageType;

  @TableField("storage_bucket")
  private String storageBucket;

  @TableField("storage_prefix")
  private String storagePrefix;

  @TableField("package_object_key")
  private String packageObjectKey;

  @TableField("file_count")
  private Integer fileCount;

  @TableField("total_size_bytes")
  private Long totalSizeBytes;

  private Integer status;

  private String creator;

  private String modifier;

  @TableField("tenant_id")
  private String tenantId;
}
