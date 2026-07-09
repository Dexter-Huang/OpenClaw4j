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

package com.seaskyland.llm.workflow.admin.controller;

import com.seaskyland.llm.workflow.admin.annotation.ApiModelAttribute;
import com.seaskyland.llm.workflow.core.base.service.SkillService;
import com.seaskyland.llm.workflow.core.context.RequestContextHolder;
import com.seaskyland.llm.workflow.runtime.domain.PagingList;
import com.seaskyland.llm.workflow.runtime.domain.RequestContext;
import com.seaskyland.llm.workflow.runtime.domain.Result;
import com.seaskyland.llm.workflow.runtime.domain.skill.SkillDetail;
import com.seaskyland.llm.workflow.runtime.domain.skill.SkillPackageUploadResult;
import com.seaskyland.llm.workflow.runtime.domain.skill.SkillQuery;
import com.seaskyland.llm.workflow.runtime.enums.ErrorCode;
import com.seaskyland.llm.workflow.runtime.exception.BizException;
import io.swagger.v3.oas.annotations.tags.Tag;
import java.util.List;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

/** Skill 管理接口。 */
@Slf4j
@RestController
@Tag(name = "skill")
@RequestMapping("/console/v1/skills")
public class SkillController {

  private final SkillService skillService;

  public SkillController(SkillService skillService) {
    this.skillService = skillService;
  }

  @PostMapping()
  public Result<String> createSkill(@RequestBody SkillDetail detail) {
    RequestContext context = RequestContextHolder.getRequestContext();
    validateWriteDetail(detail, false);
    String skillCode = skillService.createSkill(detail);
    return Result.success(context.getRequestId(), skillCode);
  }

  @PostMapping(value = "/package", consumes = "multipart/form-data")
  public Result<SkillPackageUploadResult> uploadPackage(@RequestParam("file") MultipartFile file) {
    RequestContext context = RequestContextHolder.getRequestContext();
    SkillPackageUploadResult result = skillService.uploadPackage(file);
    return Result.success(context.getRequestId(), result);
  }

  @PutMapping()
  public Result<String> updateSkill(@RequestBody SkillDetail detail) {
    RequestContext context = RequestContextHolder.getRequestContext();
    validateWriteDetail(detail, true);
    skillService.updateSkill(detail);
    return Result.success(context.getRequestId(), null);
  }

  @DeleteMapping("/{skillCode}")
  public Result<Void> deleteSkill(@PathVariable("skillCode") String skillCode) {
    RequestContext context = RequestContextHolder.getRequestContext();
    if (StringUtils.isBlank(skillCode)) {
      throw new BizException(ErrorCode.MISSING_PARAMS.toError("skillCode"));
    }
    skillService.deleteSkill(skillCode);
    return Result.success(context.getRequestId(), null);
  }

  @PostMapping("/{skillCode}/publish")
  public Result<Void> publishSkill(@PathVariable("skillCode") String skillCode) {
    RequestContext context = RequestContextHolder.getRequestContext();
    if (StringUtils.isBlank(skillCode)) {
      throw new BizException(ErrorCode.MISSING_PARAMS.toError("skillCode"));
    }
    skillService.publishSkill(skillCode);
    return Result.success(context.getRequestId(), null);
  }

  @GetMapping("/{skillCode}")
  public Result<SkillDetail> getSkill(
      @PathVariable("skillCode") String skillCode,
      @RequestParam(value = "version", required = false) String version,
      @RequestParam(value = "need_files", required = false, defaultValue = "false")
          Boolean needFiles) {
    RequestContext context = RequestContextHolder.getRequestContext();
    if (StringUtils.isBlank(skillCode)) {
      throw new BizException(ErrorCode.MISSING_PARAMS.toError("skillCode"));
    }
    SkillDetail detail = skillService.getSkill(skillCode, version, Boolean.TRUE.equals(needFiles));
    return Result.success(context.getRequestId(), detail);
  }

  @GetMapping()
  public Result<PagingList<SkillDetail>> listSkills(@ApiModelAttribute SkillQuery query) {
    RequestContext context = RequestContextHolder.getRequestContext();
    PagingList<SkillDetail> skills = skillService.list(query);
    return Result.success(context.getRequestId(), skills);
  }

  @PostMapping("/query-by-codes")
  public Result<List<SkillDetail>> listSkillsByCodes(@RequestBody SkillQuery query) {
    RequestContext context = RequestContextHolder.getRequestContext();
    List<SkillDetail> skills = skillService.listByCodes(query);
    return Result.success(context.getRequestId(), skills);
  }

  private void validateWriteDetail(SkillDetail detail, boolean requireSkillCode) {
    if (detail == null) {
      throw new BizException(ErrorCode.MISSING_PARAMS.toError("skill"));
    }
    if (requireSkillCode && StringUtils.isBlank(detail.getSkillCode())) {
      throw new BizException(ErrorCode.MISSING_PARAMS.toError("skillCode"));
    }
    if (StringUtils.isBlank(detail.getName())) {
      throw new BizException(ErrorCode.MISSING_PARAMS.toError("skillName"));
    }
  }
}
