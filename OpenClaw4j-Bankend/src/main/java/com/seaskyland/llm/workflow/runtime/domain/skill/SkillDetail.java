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

package com.seaskyland.llm.workflow.runtime.domain.skill;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.Date;
import lombok.Data;

/** Skill 管理端详情。 */
@Data
public class SkillDetail implements Serializable {

  @JsonProperty("skill_code")
  private String skillCode;

  private String name;

  private String description;

  private String source;

  private Integer status;

  @JsonProperty("current_version")
  private String currentVersion;

  private String version;

  private String tags;

  @JsonProperty("main_file_path")
  private String mainFilePath;

  private String manifest;

  @JsonProperty("content_hash")
  private String contentHash;

  @JsonProperty("storage_type")
  private String storageType;

  @JsonProperty("storage_bucket")
  private String storageBucket;

  @JsonProperty("storage_prefix")
  private String storagePrefix;

  @JsonProperty("package_object_key")
  private String packageObjectKey;

  @JsonProperty("file_count")
  private Integer fileCount;

  @JsonProperty("total_size_bytes")
  private Long totalSizeBytes;

  @JsonProperty("need_files")
  private Boolean needFiles = false;

  @JsonProperty("gmt_modified")
  private Date gmtModified;
}
