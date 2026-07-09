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

package com.seaskyland.llm.workflow.core.base.service.impl;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.core.conditions.update.LambdaUpdateWrapper;
import com.baomidou.mybatisplus.core.metadata.IPage;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import com.seaskyland.llm.workflow.core.base.entity.SkillEntity;
import com.seaskyland.llm.workflow.core.base.entity.SkillVersionEntity;
import com.seaskyland.llm.workflow.core.base.manager.OssManager;
import com.seaskyland.llm.workflow.core.base.mapper.SkillMapper;
import com.seaskyland.llm.workflow.core.base.mapper.SkillVersionMapper;
import com.seaskyland.llm.workflow.core.base.service.SkillService;
import com.seaskyland.llm.workflow.core.config.StudioProperties;
import com.seaskyland.llm.workflow.core.context.RequestContextHolder;
import com.seaskyland.llm.workflow.core.utils.common.IdGenerator;
import com.seaskyland.llm.workflow.runtime.domain.PagingList;
import com.seaskyland.llm.workflow.runtime.domain.RequestContext;
import com.seaskyland.llm.workflow.runtime.domain.skill.SkillDetail;
import com.seaskyland.llm.workflow.runtime.domain.skill.SkillPackageUploadResult;
import com.seaskyland.llm.workflow.runtime.domain.skill.SkillQuery;
import com.seaskyland.llm.workflow.runtime.domain.skill.SkillRuntimeInfo;
import com.seaskyland.llm.workflow.runtime.enums.ErrorCode;
import com.seaskyland.llm.workflow.runtime.enums.SkillStatusEnum;
import com.seaskyland.llm.workflow.runtime.enums.UploadType;
import com.seaskyland.llm.workflow.runtime.exception.BizException;
import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
import java.io.BufferedInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.InvalidPathException;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardCopyOption;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.Date;
import java.util.HexFormat;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Optional;
import java.util.stream.Stream;
import java.util.zip.ZipEntry;
import java.util.zip.ZipInputStream;
import org.apache.commons.io.FilenameUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.CollectionUtils;
import org.springframework.web.multipart.MultipartFile;

/** Skill 管理服务实现。 */
@Service
public class SkillServiceImpl extends ServiceImpl<SkillMapper, SkillEntity>
    implements SkillService {

  private static final String DEFAULT_VERSION = "1";
  private static final String DEFAULT_MAIN_FILE = "SKILL.md";
  private static final String DEFAULT_SOURCE = "CUSTOMER";
  private static final String SKILL_STORAGE_ROOT = "skills";
  private static final String SKILL_PACKAGE_DIR = "packages";
  private static final String PACKAGE_FILE_NAME = "package.zip";
  private static final String MANIFEST_FILE_NAME = "manifest.json";
  private static final int MAX_SKILL_FILE_COUNT = 1000;
  private static final long MAX_SKILL_PACKAGE_BYTES = 100L * 1024L * 1024L;

  private final SkillVersionMapper skillVersionMapper;
  private final StudioProperties studioProperties;
  private final OssManager ossManager;

  public SkillServiceImpl(
      SkillVersionMapper skillVersionMapper, StudioProperties studioProperties, OssManager ossManager) {
    this.skillVersionMapper = skillVersionMapper;
    this.studioProperties = studioProperties;
    this.ossManager = ossManager;
  }

  @Override
  @Transactional(rollbackFor = Exception.class)
  public String createSkill(SkillDetail detail) {
    try {
      RequestContext context = RequestContextHolder.getRequestContext();
      Date now = new Date();
      String skillCode = IdGenerator.idStr();
      String mainFilePath = normalizeMainFilePath(detail.getMainFilePath());

      SkillEntity entity = new SkillEntity();
      entity.setSkillCode(skillCode);
      entity.setWorkspaceId(context.getWorkspaceId());
      entity.setAccountId(context.getAccountId());
      entity.setName(detail.getName());
      entity.setDescription(detail.getDescription());
      entity.setSource(StringUtils.defaultIfBlank(detail.getSource(), DEFAULT_SOURCE));
      entity.setStatus(defaultStatus(detail.getStatus()));
      entity.setTags(detail.getTags());
      entity.setCreator(context.getAccountId());
      entity.setModifier(context.getAccountId());
      entity.setGmtCreate(now);
      entity.setGmtModified(now);
      this.save(entity);

      saveVersionSnapshot(
          skillCode, DEFAULT_VERSION, mainFilePath, detail, context, now, entity.getStatus());
      return skillCode;
    } catch (BizException e) {
      throw e;
    } catch (Exception e) {
      throw new BizException(ErrorCode.CREATE_SKILL_ERROR.toError(), e);
    }
  }

  @Override
  public SkillPackageUploadResult uploadPackage(MultipartFile file) {
    if (file == null || file.isEmpty()) {
      throw new BizException(ErrorCode.MISSING_PARAMS.toError("file"));
    }
    String extension = StringUtils.lowerCase(FilenameUtils.getExtension(file.getOriginalFilename()));
    if (!"zip".equals(extension)) {
      throw new BizException(
          ErrorCode.INVALID_PARAMS.toError("file", "skill package must be a zip file"));
    }
    if (file.getSize() > MAX_SKILL_PACKAGE_BYTES) {
      throw new BizException(
          ErrorCode.INVALID_PARAMS.toError("file", "skill package must be smaller than 100MB"));
    }

    Path tempRoot = null;
    try {
      RequestContext context = RequestContextHolder.getRequestContext();
      String storageType = normalizeStorageType(null);
      String packageId = IdGenerator.uuid32();
      String storagePrefix =
          normalizeStoragePrefix(
              SKILL_STORAGE_ROOT
                  + "/"
                  + context.getWorkspaceId()
                  + "/"
                  + SKILL_PACKAGE_DIR
                  + "/"
                  + packageId
                  + "/");
      String packageObjectKey = storagePrefix + PACKAGE_FILE_NAME;

      tempRoot = Files.createTempDirectory("skill-package-");
      Path zipFile = tempRoot.resolve(PACKAGE_FILE_NAME);
      Path extractRoot = tempRoot.resolve("content");
      Files.createDirectories(extractRoot);
      file.transferTo(zipFile);

      SkillPackageStats stats = unzipAndValidateSkillPackage(zipFile, extractRoot);
      if (UploadType.OSS.getValue().equals(storageType)) {
        uploadSkillPackageToOss(storagePrefix, packageObjectKey, zipFile, extractRoot);
      } else {
        storeSkillPackageToLocal(storagePrefix, packageObjectKey, zipFile, extractRoot);
      }

      SkillPackageUploadResult result = new SkillPackageUploadResult();
      result.setStorageType(storageType);
      result.setStorageBucket(resolveStorageBucket(storageType, null));
      result.setStoragePrefix(storagePrefix);
      result.setPackageObjectKey(packageObjectKey);
      result.setMainFilePath(DEFAULT_MAIN_FILE);
      result.setManifest(stats.manifest());
      result.setContentHash(sha256(zipFile));
      result.setFileCount(stats.fileCount());
      result.setTotalSizeBytes(stats.totalSizeBytes());
      return result;
    } catch (BizException e) {
      throw e;
    } catch (Exception e) {
      throw new BizException(ErrorCode.INVALID_PARAMS.toError("file", e.getMessage()), e);
    } finally {
      deleteTempDirectory(tempRoot);
    }
  }

  @Override
  @Transactional(rollbackFor = Exception.class)
  public void updateSkill(SkillDetail detail) {
    try {
      RequestContext context = RequestContextHolder.getRequestContext();
      SkillEntity entity = getSkillByCode(context.getWorkspaceId(), detail.getSkillCode(), null);
      if (entity == null) {
        throw new BizException(ErrorCode.SKILL_NOT_FOUND.toError());
      }

      Date now = new Date();
      SkillVersionEntity targetVersion = resolveEditableVersion(entity, context, now);
      String mainFilePath = normalizeMainFilePath(detail.getMainFilePath());

      entity.setName(detail.getName());
      entity.setDescription(detail.getDescription());
      entity.setSource(StringUtils.defaultIfBlank(detail.getSource(), entity.getSource()));
      entity.setTags(detail.getTags());
      if (SkillStatusEnum.Published.getCode().equals(entity.getStatus())) {
        entity.setStatus(SkillStatusEnum.PublishedEditing.getCode());
      }
      entity.setModifier(context.getAccountId());
      entity.setGmtModified(now);
      this.updateById(entity);

      saveVersionSnapshot(
          detail.getSkillCode(),
          targetVersion.getVersion(),
          mainFilePath,
          detail,
          context,
          now,
          SkillStatusEnum.Draft.getCode());
    } catch (BizException e) {
      throw e;
    } catch (Exception e) {
      throw new BizException(ErrorCode.UPDATE_SKILL_ERROR.toError(), e);
    }
  }

  @Override
  @Transactional(rollbackFor = Exception.class)
  public void publishSkill(String skillCode) {
    try {
      RequestContext context = RequestContextHolder.getRequestContext();
      SkillEntity entity = getSkillByCode(context.getWorkspaceId(), skillCode, null);
      if (entity == null) {
        throw new BizException(ErrorCode.SKILL_NOT_FOUND.toError());
      }
      SkillVersionEntity latestVersion = getLatestVersion(context.getWorkspaceId(), skillCode);
      if (latestVersion == null) {
        throw new BizException(ErrorCode.SKILL_NOT_FOUND.toError());
      }
      Date now = new Date();
      latestVersion.setStatus(SkillStatusEnum.Published.getCode());
      latestVersion.setModifier(context.getAccountId());
      latestVersion.setGmtModified(now);
      skillVersionMapper.updateById(latestVersion);

      entity.setStatus(SkillStatusEnum.Published.getCode());
      entity.setModifier(context.getAccountId());
      entity.setGmtModified(now);
      this.updateById(entity);
    } catch (BizException e) {
      throw e;
    } catch (Exception e) {
      throw new BizException(ErrorCode.UPDATE_SKILL_ERROR.toError(), e);
    }
  }

  @Override
  @Transactional(rollbackFor = Exception.class)
  public void deleteSkill(String skillCode) {
    try {
      RequestContext context = RequestContextHolder.getRequestContext();
      SkillEntity entity = getSkillByCode(context.getWorkspaceId(), skillCode, null);
      if (entity == null) {
        throw new BizException(ErrorCode.SKILL_NOT_FOUND.toError());
      }
      Date now = new Date();
      entity.setStatus(SkillStatusEnum.Deleted.getCode());
      entity.setGmtModified(now);
      entity.setModifier(context.getAccountId());
      this.updateById(entity);

      LambdaUpdateWrapper<SkillVersionEntity> versionUpdate = new LambdaUpdateWrapper<>();
      versionUpdate
          .eq(SkillVersionEntity::getWorkspaceId, context.getWorkspaceId())
          .eq(SkillVersionEntity::getSkillCode, skillCode)
          .set(SkillVersionEntity::getStatus, SkillStatusEnum.Deleted.getCode())
          .set(SkillVersionEntity::getGmtModified, now)
          .set(SkillVersionEntity::getModifier, context.getAccountId());
      skillVersionMapper.update(null, versionUpdate);
    } catch (BizException e) {
      throw e;
    } catch (Exception e) {
      throw new BizException(ErrorCode.DELETE_SKILL_ERROR.toError(), e);
    }
  }

  @Override
  public SkillDetail getSkill(String skillCode, String version, boolean needFiles) {
    RequestContext context = RequestContextHolder.getRequestContext();
    SkillEntity entity = getSkillByCode(context.getWorkspaceId(), skillCode, null);
    if (entity == null) {
      throw new BizException(ErrorCode.SKILL_NOT_FOUND.toError());
    }
    SkillVersionEntity versionEntity =
        StringUtils.isBlank(version)
            ? getLatestVersion(context.getWorkspaceId(), skillCode)
            : getVersion(context.getWorkspaceId(), skillCode, version);
    if (versionEntity == null) {
      throw new BizException(ErrorCode.SKILL_NOT_FOUND.toError());
    }
    return toSkillDetail(entity, versionEntity);
  }

  @Override
  public PagingList<SkillDetail> list(SkillQuery query) {
    RequestContext context = RequestContextHolder.getRequestContext();
    Page<SkillEntity> page = new Page<>(query.getCurrent(), query.getSize());
    LambdaQueryWrapper<SkillEntity> queryWrapper = new LambdaQueryWrapper<>();
    queryWrapper.eq(SkillEntity::getWorkspaceId, context.getWorkspaceId());
    if (StringUtils.isNotBlank(query.getName())) {
      queryWrapper.like(SkillEntity::getName, query.getName());
    }
    if (query.getStatus() != null) {
      queryWrapper.eq(SkillEntity::getStatus, query.getStatus());
    } else {
      queryWrapper.ne(SkillEntity::getStatus, SkillStatusEnum.Deleted.getCode());
    }
    queryWrapper.orderByDesc(SkillEntity::getId);
    IPage<SkillEntity> pageResult = this.page(page, queryWrapper);
    List<SkillDetail> details = new ArrayList<>();
    if (!CollectionUtils.isEmpty(pageResult.getRecords())) {
      for (SkillEntity entity : pageResult.getRecords()) {
        SkillVersionEntity versionEntity = getLatestVersion(context.getWorkspaceId(), entity.getSkillCode());
        details.add(toSkillDetail(entity, versionEntity));
      }
    }
    return new PagingList<>(query.getCurrent(), query.getSize(), pageResult.getTotal(), details);
  }

  @Override
  public List<SkillDetail> listByCodes(SkillQuery query) {
    RequestContext context = RequestContextHolder.getRequestContext();
    LambdaQueryWrapper<SkillEntity> queryWrapper = new LambdaQueryWrapper<>();
    queryWrapper
        .eq(SkillEntity::getWorkspaceId, context.getWorkspaceId())
        .ne(SkillEntity::getStatus, SkillStatusEnum.Deleted.getCode());
    if (!CollectionUtils.isEmpty(query.getSkillCodes())) {
      queryWrapper.in(SkillEntity::getSkillCode, query.getSkillCodes());
    }
    queryWrapper.orderByDesc(SkillEntity::getGmtModified);
    List<SkillEntity> entities = this.list(queryWrapper);
    List<SkillDetail> details = new ArrayList<>();
    if (!CollectionUtils.isEmpty(entities)) {
      for (SkillEntity entity : entities) {
        SkillVersionEntity versionEntity = getLatestVersion(context.getWorkspaceId(), entity.getSkillCode());
        details.add(toSkillDetail(entity, versionEntity));
      }
    }
    return details;
  }

  @Override
  public List<SkillRuntimeInfo> getSkillRuntimeInfos(List<String> skillCodes) {
    if (CollectionUtils.isEmpty(skillCodes)) {
      return List.of();
    }

    RequestContext context = RequestContextHolder.getRequestContext();
    List<SkillRuntimeInfo> runtimeInfos = new ArrayList<>();
    for (String skillCode : new LinkedHashSet<>(skillCodes)) {
      if (StringUtils.isBlank(skillCode)) {
        continue;
      }
      SkillEntity entity = getSkillByCode(context.getWorkspaceId(), skillCode, null);
      if (entity == null) {
        throw new BizException(ErrorCode.SKILL_NOT_FOUND.toError());
      }
      SkillVersionEntity versionEntity = getLatestVersion(context.getWorkspaceId(), skillCode);
      if (versionEntity == null) {
        throw new BizException(ErrorCode.SKILL_NOT_FOUND.toError());
      }

      SkillRuntimeInfo runtimeInfo = new SkillRuntimeInfo();
      runtimeInfo.setSkillCode(entity.getSkillCode());
      runtimeInfo.setName(entity.getName());
      runtimeInfo.setDescription(entity.getDescription());
      runtimeInfo.setMainFilePath(normalizeMainFilePath(versionEntity.getMainFilePath()));
      runtimeInfos.add(runtimeInfo);
    }
    return runtimeInfos;
  }

  @Override
  public String readSkillFile(String skillCode, String path) {
    if (StringUtils.isBlank(skillCode)) {
      throw new BizException(ErrorCode.INVALID_PARAMS.toError("skill_code", "skill code is required"));
    }

    RequestContext context = RequestContextHolder.getRequestContext();
    SkillEntity entity = getSkillByCode(context.getWorkspaceId(), skillCode, null);
    if (entity == null) {
      throw new BizException(ErrorCode.SKILL_NOT_FOUND.toError());
    }
    SkillVersionEntity versionEntity = getLatestVersion(context.getWorkspaceId(), skillCode);
    if (versionEntity == null) {
      throw new BizException(ErrorCode.SKILL_NOT_FOUND.toError());
    }
    String filePath =
        StringUtils.isBlank(path) ? normalizeMainFilePath(versionEntity.getMainFilePath()) : normalizeRelativePath(path);
    return readSkillFile(versionEntity, filePath);
  }

  @Override
  public List<String> getSkillMainFileContents(List<String> skillCodes) {
    if (CollectionUtils.isEmpty(skillCodes)) {
      return List.of();
    }

    RequestContext context = RequestContextHolder.getRequestContext();
    List<String> contents = new ArrayList<>();
    for (String skillCode : new LinkedHashSet<>(skillCodes)) {
      if (StringUtils.isBlank(skillCode)) {
        continue;
      }
      SkillEntity entity = getSkillByCode(context.getWorkspaceId(), skillCode, null);
      if (entity == null) {
        throw new BizException(ErrorCode.SKILL_NOT_FOUND.toError());
      }
      SkillVersionEntity versionEntity = getLatestVersion(context.getWorkspaceId(), skillCode);
      if (versionEntity == null) {
        throw new BizException(ErrorCode.SKILL_NOT_FOUND.toError());
      }
      String content = readSkillFile(versionEntity, normalizeMainFilePath(versionEntity.getMainFilePath()));
      if (StringUtils.isNotBlank(content)) {
        contents.add(content);
      }
    }
    return contents;
  }

  @Override
  public SkillEntity getSkillByCode(String workspaceId, String skillCode, Integer status) {
    LambdaQueryWrapper<SkillEntity> queryWrapper = new LambdaQueryWrapper<>();
    queryWrapper
        .eq(SkillEntity::getWorkspaceId, workspaceId)
        .eq(SkillEntity::getSkillCode, skillCode);
    if (status != null) {
      queryWrapper.eq(SkillEntity::getStatus, status);
    } else {
      queryWrapper.ne(SkillEntity::getStatus, SkillStatusEnum.Deleted.getCode());
    }
    Optional<SkillEntity> entityOptional = this.getOneOpt(queryWrapper);
    return entityOptional.orElse(null);
  }

  private SkillVersionEntity resolveEditableVersion(
      SkillEntity entity, RequestContext context, Date now) {
    SkillVersionEntity latestVersion = getLatestVersion(context.getWorkspaceId(), entity.getSkillCode());
    if (latestVersion == null) {
      throw new BizException(ErrorCode.SKILL_NOT_FOUND.toError());
    }

    // Skill 包版本由后端统一维护：每次保存都基于最新版本复制并递增，避免前端手填版本造成覆盖。
    SkillVersionEntity draftVersion = copyVersion(latestVersion);
    draftVersion.setVersion(nextVersion(latestVersion.getVersion()));
    draftVersion.setStatus(SkillStatusEnum.Draft.getCode());
    draftVersion.setCreator(context.getAccountId());
    draftVersion.setModifier(context.getAccountId());
    draftVersion.setGmtCreate(now);
    draftVersion.setGmtModified(now);
    skillVersionMapper.insert(draftVersion);
    return draftVersion;
  }

  private void saveVersionSnapshot(
      String skillCode,
      String version,
      String mainFilePath,
      SkillDetail detail,
      RequestContext context,
      Date now,
      Integer versionStatus) {
    SkillVersionEntity versionEntity = getVersion(context.getWorkspaceId(), skillCode, version);
    if (versionEntity == null) {
      versionEntity = new SkillVersionEntity();
      versionEntity.setSkillCode(skillCode);
      versionEntity.setWorkspaceId(context.getWorkspaceId());
      versionEntity.setVersion(version);
      versionEntity.setCreator(context.getAccountId());
      versionEntity.setGmtCreate(now);
    }

    String storageType = normalizeStorageType(detail.getStorageType());
    String storagePrefix =
        StringUtils.defaultIfBlank(
            detail.getStoragePrefix(),
            buildDefaultStoragePrefix(context.getWorkspaceId(), skillCode, version));
    String contentHash =
        StringUtils.defaultIfBlank(
            detail.getContentHash(),
            calculateContentHash(detail, storageType, storagePrefix, mainFilePath));

    versionEntity.setDescription(detail.getDescription());
    versionEntity.setMainFilePath(mainFilePath);
    versionEntity.setManifest(detail.getManifest());
    versionEntity.setContentHash(contentHash);
    versionEntity.setStorageType(storageType);
    versionEntity.setStorageBucket(resolveStorageBucket(storageType, detail.getStorageBucket()));
    versionEntity.setStoragePrefix(normalizeStoragePrefix(storagePrefix));
    versionEntity.setPackageObjectKey(detail.getPackageObjectKey());
    versionEntity.setFileCount(detail.getFileCount());
    versionEntity.setTotalSizeBytes(detail.getTotalSizeBytes());
    versionEntity.setStatus(versionStatus);
    versionEntity.setModifier(context.getAccountId());
    versionEntity.setGmtModified(now);
    if (versionEntity.getId() == null) {
      skillVersionMapper.insert(versionEntity);
    } else {
      skillVersionMapper.updateById(versionEntity);
    }
  }

  private SkillVersionEntity getVersion(String workspaceId, String skillCode, String version) {
    if (StringUtils.isBlank(skillCode) || StringUtils.isBlank(version)) {
      return null;
    }
    LambdaQueryWrapper<SkillVersionEntity> queryWrapper = new LambdaQueryWrapper<>();
    queryWrapper
        .eq(SkillVersionEntity::getWorkspaceId, workspaceId)
        .eq(SkillVersionEntity::getSkillCode, skillCode)
        .eq(SkillVersionEntity::getVersion, version)
        .ne(SkillVersionEntity::getStatus, SkillStatusEnum.Deleted.getCode());
    return skillVersionMapper.selectOne(queryWrapper);
  }

  private SkillVersionEntity getLatestVersion(String workspaceId, String skillCode) {
    LambdaQueryWrapper<SkillVersionEntity> queryWrapper = new LambdaQueryWrapper<>();
    queryWrapper
        .eq(SkillVersionEntity::getWorkspaceId, workspaceId)
        .eq(SkillVersionEntity::getSkillCode, skillCode)
        .ne(SkillVersionEntity::getStatus, SkillStatusEnum.Deleted.getCode())
        .orderByDesc(SkillVersionEntity::getId)
        .last("LIMIT 1");
    return skillVersionMapper.selectOne(queryWrapper);
  }

  private SkillDetail toSkillDetail(SkillEntity entity, SkillVersionEntity versionEntity) {
    SkillDetail detail = new SkillDetail();
    detail.setSkillCode(entity.getSkillCode());
    detail.setName(entity.getName());
    detail.setDescription(entity.getDescription());
    detail.setSource(entity.getSource());
    detail.setStatus(entity.getStatus());
    detail.setTags(entity.getTags());
    detail.setGmtModified(entity.getGmtModified());
    if (versionEntity != null) {
      detail.setCurrentVersion(versionEntity.getVersion());
      detail.setVersion(versionEntity.getVersion());
      detail.setMainFilePath(versionEntity.getMainFilePath());
      detail.setManifest(versionEntity.getManifest());
      detail.setContentHash(versionEntity.getContentHash());
      detail.setStorageType(versionEntity.getStorageType());
      detail.setStorageBucket(versionEntity.getStorageBucket());
      detail.setStoragePrefix(versionEntity.getStoragePrefix());
      detail.setPackageObjectKey(versionEntity.getPackageObjectKey());
      detail.setFileCount(versionEntity.getFileCount());
      detail.setTotalSizeBytes(versionEntity.getTotalSizeBytes());
    }
    return detail;
  }

  private SkillVersionEntity copyVersion(SkillVersionEntity source) {
    SkillVersionEntity target = new SkillVersionEntity();
    target.setSkillCode(source.getSkillCode());
    target.setWorkspaceId(source.getWorkspaceId());
    target.setDescription(source.getDescription());
    target.setMainFilePath(source.getMainFilePath());
    target.setManifest(source.getManifest());
    target.setContentHash(source.getContentHash());
    target.setStorageType(source.getStorageType());
    target.setStorageBucket(source.getStorageBucket());
    target.setStoragePrefix(source.getStoragePrefix());
    target.setPackageObjectKey(source.getPackageObjectKey());
    target.setFileCount(source.getFileCount());
    target.setTotalSizeBytes(source.getTotalSizeBytes());
    target.setTenantId(source.getTenantId());
    return target;
  }

  private String nextVersion(String version) {
    try {
      return String.valueOf(Integer.parseInt(version) + 1);
    } catch (NumberFormatException e) {
      throw new BizException(
          ErrorCode.INVALID_PARAMS.toError("version", "skill version must be numeric"), e);
    }
  }

  private String normalizeMainFilePath(String mainFilePath) {
    return normalizeRelativePath(StringUtils.defaultIfBlank(mainFilePath, DEFAULT_MAIN_FILE));
  }

  private String normalizeStoragePrefix(String storagePrefix) {
    String normalizedPrefix = normalizeRelativePath(storagePrefix);
    return normalizedPrefix.endsWith("/") ? normalizedPrefix : normalizedPrefix + "/";
  }

  private String normalizeRelativePath(String path) {
    if (StringUtils.isBlank(path)) {
      throw new BizException(ErrorCode.INVALID_PARAMS.toError("path", "path is required"));
    }
    String normalizedPath = path.trim().replace('\\', '/');
    if (normalizedPath.startsWith("/") || normalizedPath.contains("../") || normalizedPath.equals("..")) {
      throw new BizException(
          ErrorCode.INVALID_PARAMS.toError("path", "path must stay inside skill package"));
    }
    try {
      if (Paths.get(normalizedPath).isAbsolute()) {
        throw new BizException(ErrorCode.INVALID_PARAMS.toError("path", "path must be relative"));
      }
    } catch (InvalidPathException e) {
      throw new BizException(ErrorCode.INVALID_PARAMS.toError("path", e.getMessage()), e);
    }
    return normalizedPath;
  }

  private String normalizeStorageType(String storageType) {
    return UploadType.fromValue(StringUtils.defaultIfBlank(storageType, studioProperties.getUploadMethod()))
        .getValue();
  }

  private String resolveStorageBucket(String storageType, String storageBucket) {
    if (StringUtils.isNotBlank(storageBucket)) {
      return storageBucket;
    }
    if (UploadType.OSS.getValue().equals(storageType) && studioProperties.getOss() != null) {
      return studioProperties.getOss().getBucket();
    }
    return null;
  }

  private Integer defaultStatus(Integer status) {
    return status == null ? SkillStatusEnum.Draft.getCode() : status;
  }

  private String buildDefaultStoragePrefix(String workspaceId, String skillCode, String version) {
    return SKILL_STORAGE_ROOT + "/" + workspaceId + "/" + skillCode + "/" + version + "/";
  }

  private SkillPackageStats unzipAndValidateSkillPackage(Path zipFile, Path extractRoot)
      throws IOException {
    int fileCount = 0;
    long totalSizeBytes = 0L;
    boolean hasMainFile = false;
    Path manifestPath = null;

    try (InputStream inputStream = new BufferedInputStream(Files.newInputStream(zipFile));
        ZipInputStream zipInputStream = new ZipInputStream(inputStream, StandardCharsets.UTF_8)) {
      ZipEntry entry;
      while ((entry = zipInputStream.getNextEntry()) != null) {
        String entryPath = normalizeZipEntryPath(entry.getName());
        if (entry.isDirectory()) {
          zipInputStream.closeEntry();
          continue;
        }

        fileCount++;
        if (fileCount > MAX_SKILL_FILE_COUNT) {
          throw new BizException(
              ErrorCode.INVALID_PARAMS.toError(
                  "file", "skill package contains too many files"));
        }

        Path targetPath = extractRoot.resolve(entryPath).normalize();
        if (!targetPath.startsWith(extractRoot)) {
          throw new BizException(
              ErrorCode.INVALID_PARAMS.toError("file", "skill package path is invalid"));
        }
        Files.createDirectories(targetPath.getParent());

        long written = copyZipEntry(zipInputStream, targetPath);
        totalSizeBytes += written;
        if (totalSizeBytes > MAX_SKILL_PACKAGE_BYTES) {
          throw new BizException(
              ErrorCode.INVALID_PARAMS.toError(
                  "file", "skill package uncompressed size must be smaller than 100MB"));
        }

        if (DEFAULT_MAIN_FILE.equals(entryPath)) {
          hasMainFile = true;
        }
        if (MANIFEST_FILE_NAME.equals(entryPath)) {
          manifestPath = targetPath;
        }
        zipInputStream.closeEntry();
      }
    }

    if (fileCount == 0) {
      throw new BizException(ErrorCode.INVALID_PARAMS.toError("file", "skill package is empty"));
    }
    if (!hasMainFile) {
      throw new BizException(
          ErrorCode.INVALID_PARAMS.toError("file", "skill package must contain SKILL.md"));
    }

    String manifest = "{}";
    if (manifestPath != null) {
      manifest = Files.readString(manifestPath, StandardCharsets.UTF_8);
      if (!JsonUtils.isValidJson(manifest)) {
        throw new BizException(
            ErrorCode.INVALID_PARAMS.toError("file", "manifest.json must be valid JSON"));
      }
    }
    return new SkillPackageStats(fileCount, totalSizeBytes, manifest);
  }

  private long copyZipEntry(ZipInputStream zipInputStream, Path targetPath) throws IOException {
    long written = 0L;
    byte[] buffer = new byte[8192];
    try (OutputStream outputStream = Files.newOutputStream(targetPath)) {
      int length;
      while ((length = zipInputStream.read(buffer)) >= 0) {
        outputStream.write(buffer, 0, length);
        written += length;
      }
    }
    return written;
  }

  private String normalizeZipEntryPath(String entryName) {
    String normalizedPath = normalizeRelativePath(entryName).replace('\\', '/');
    while (normalizedPath.startsWith("./")) {
      normalizedPath = normalizedPath.substring(2);
    }
    if (StringUtils.isBlank(normalizedPath)) {
      throw new BizException(ErrorCode.INVALID_PARAMS.toError("file", "zip entry path is empty"));
    }
    return normalizedPath;
  }

  private void storeSkillPackageToLocal(
      String storagePrefix, String packageObjectKey, Path zipFile, Path extractRoot)
      throws IOException {
    Path storageRoot = Paths.get(studioProperties.getStoragePath()).toAbsolutePath().normalize();
    Path packageRoot = storageRoot.resolve(storagePrefix).normalize();
    if (!packageRoot.startsWith(storageRoot)) {
      throw new BizException(
          ErrorCode.INVALID_PARAMS.toError("storagePrefix", "storage prefix is invalid"));
    }
    Files.createDirectories(packageRoot);
    copyLocalFile(zipFile, storageRoot.resolve(packageObjectKey).normalize(), storageRoot);

    try (Stream<Path> paths = Files.walk(extractRoot)) {
      List<Path> files = paths.filter(Files::isRegularFile).toList();
      for (Path source : files) {
        String relativePath = toRelativeObjectPath(extractRoot, source);
        copyLocalFile(source, packageRoot.resolve(relativePath).normalize(), storageRoot);
      }
    }
  }

  private void copyLocalFile(Path source, Path target, Path storageRoot) throws IOException {
    if (!target.startsWith(storageRoot)) {
      throw new BizException(ErrorCode.INVALID_PARAMS.toError("file", "target path is invalid"));
    }
    Files.createDirectories(target.getParent());
    Files.copy(source, target, StandardCopyOption.REPLACE_EXISTING);
  }

  private void uploadSkillPackageToOss(
      String storagePrefix, String packageObjectKey, Path zipFile, Path extractRoot)
      throws IOException {
    ossManager.uploadFileToObject(packageObjectKey, zipFile.toString());
    try (Stream<Path> paths = Files.walk(extractRoot)) {
      List<Path> files = paths.filter(Files::isRegularFile).toList();
      for (Path source : files) {
        String objectKey = storagePrefix + toRelativeObjectPath(extractRoot, source);
        ossManager.uploadFileToObject(objectKey, source.toString());
      }
    }
  }

  private String readSkillFile(SkillVersionEntity versionEntity, String relativePath) {
    String objectKey =
        normalizeStoragePrefix(versionEntity.getStoragePrefix())
            + normalizeRelativePath(relativePath);
    if (UploadType.OSS.getValue().equals(versionEntity.getStorageType())) {
      return ossManager.downloadFileAsString(objectKey);
    }

    Path storageRoot = Paths.get(studioProperties.getStoragePath()).toAbsolutePath().normalize();
    Path mainFile = storageRoot.resolve(objectKey).normalize();
    if (!mainFile.startsWith(storageRoot)) {
      throw new BizException(
          ErrorCode.INVALID_PARAMS.toError("skill", "skill main file path is invalid"));
    }
    try {
      return Files.readString(mainFile, StandardCharsets.UTF_8);
    } catch (IOException e) {
      throw new BizException(ErrorCode.SKILL_NOT_FOUND.toError(), e);
    }
  }

  private String toRelativeObjectPath(Path root, Path file) {
    return root.relativize(file).toString().replace('\\', '/');
  }

  private String sha256(Path file) throws IOException {
    try {
      MessageDigest digest = MessageDigest.getInstance("SHA-256");
      byte[] buffer = new byte[8192];
      try (InputStream inputStream = Files.newInputStream(file)) {
        int length;
        while ((length = inputStream.read(buffer)) >= 0) {
          digest.update(buffer, 0, length);
        }
      }
      return HexFormat.of().formatHex(digest.digest());
    } catch (NoSuchAlgorithmException e) {
      throw new IllegalStateException("SHA-256 algorithm is not available", e);
    }
  }

  private void deleteTempDirectory(Path tempRoot) {
    if (tempRoot == null) {
      return;
    }
    Path safeTempRoot = tempRoot.toAbsolutePath().normalize();
    try (Stream<Path> paths = Files.walk(safeTempRoot)) {
      List<Path> orderedPaths =
          paths.sorted(Comparator.reverseOrder()).filter(path -> path.startsWith(safeTempRoot)).toList();
      for (Path path : orderedPaths) {
        Files.deleteIfExists(path);
      }
    } catch (IOException e) {
      // 临时目录清理失败不影响上传结果；下次系统临时目录清理会兜底。
    }
  }

  private String calculateContentHash(
      SkillDetail detail, String storageType, String storagePrefix, String mainFilePath) {
    return sha256(
        StringUtils.defaultString(detail.getManifest())
            + "\n"
            + storageType
            + "\n"
            + storagePrefix
            + "\n"
            + mainFilePath
            + "\n"
            + StringUtils.defaultString(detail.getPackageObjectKey())
            + "\n"
            + JsonUtils.toJson(detail));
  }

  private String sha256(String data) {
    try {
      MessageDigest digest = MessageDigest.getInstance("SHA-256");
      return HexFormat.of().formatHex(digest.digest(data.getBytes(StandardCharsets.UTF_8)));
    } catch (NoSuchAlgorithmException e) {
      throw new IllegalStateException("SHA-256 algorithm is not available", e);
    }
  }

  private record SkillPackageStats(Integer fileCount, Long totalSizeBytes, String manifest) {}
}
