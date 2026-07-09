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

package com.seaskyland.llm.workflow.core.base.service;

import com.seaskyland.llm.workflow.core.base.entity.SkillEntity;
import com.seaskyland.llm.workflow.runtime.domain.PagingList;
import com.seaskyland.llm.workflow.runtime.domain.skill.SkillDetail;
import com.seaskyland.llm.workflow.runtime.domain.skill.SkillPackageUploadResult;
import com.seaskyland.llm.workflow.runtime.domain.skill.SkillQuery;
import com.seaskyland.llm.workflow.runtime.domain.skill.SkillRuntimeInfo;
import java.util.List;
import org.springframework.web.multipart.MultipartFile;

/** Skill 管理服务。 */
public interface SkillService {

  /**
   * 创建 Skill 包及首个版本。
   *
   * @param detail Skill 详情
   * @return Skill code
   */
  String createSkill(SkillDetail detail);

  /**
   * 上传 Skill zip 包，完成校验、解压和存储，并返回可写入版本表的包元数据。
   *
   * @param file Skill zip 包
   * @return 上传结果
   */
  SkillPackageUploadResult uploadPackage(MultipartFile file);

  /**
   * 更新 Skill 基本信息，并基于最新版本生成下一个草稿版本。版本号由后端维护，前端不需要传入。
   *
   * @param detail Skill 详情
   */
  void updateSkill(SkillDetail detail);

  /**
   * 发布 Skill 的最新草稿版本。
   *
   * @param skillCode Skill code
   */
  void publishSkill(String skillCode);

  /**
   * 软删除 Skill。
   *
   * @param skillCode Skill code
   */
  void deleteSkill(String skillCode);

  /**
   * 查询 Skill 详情。
   *
   * @param skillCode Skill code
   * @param version 指定版本，空值时使用最新非删除版本
   * @param needFiles 兼容旧参数，OSS 存储模式下不返回文件明细
   * @return Skill 详情
   */
  SkillDetail getSkill(String skillCode, String version, boolean needFiles);

  /**
   * 分页查询 Skill。
   *
   * @param query 查询条件
   * @return 分页结果
   */
  PagingList<SkillDetail> list(SkillQuery query);

  /**
   * 按 code 批量查询 Skill。
   *
   * @param query 查询条件
   * @return Skill 列表
   */
  List<SkillDetail> listByCodes(SkillQuery query);

  /**
   * 查询 Agent 运行时需要暴露给模型的 Skill 轻量索引信息。
   *
   * @param skillCodes Skill code 列表
   * @return 按传入 code 去重后的 Skill 运行时索引
   */
  List<SkillRuntimeInfo> getSkillRuntimeInfos(List<String> skillCodes);

  /**
   * 读取 Skill 包内指定相对路径文件，用于 Agent 工具按需加载 Skill 内容。
   *
   * @param skillCode Skill code
   * @param path Skill 包内相对路径，空值时读取入口文件
   * @return 文件正文
   */
  String readSkillFile(String skillCode, String path);

  /**
   * 读取一组 Skill 当前版本入口文件内容。保留该方法用于兼容旧运行时逻辑，新 Agent 链路优先使用
   * {@link #getSkillRuntimeInfos(List)} 和 {@link #readSkillFile(String, String)}。
   *
   * @param skillCodes Skill code 列表
   * @return 按传入 code 顺序返回的入口文件正文
   */
  List<String> getSkillMainFileContents(List<String> skillCodes);

  /**
   * 按 workspace 和 code 查询主表实体。
   *
   * @param workspaceId 工作空间
   * @param skillCode Skill code
   * @param status 状态过滤，空值时排除删除态
   * @return Skill 主表实体
   */
  SkillEntity getSkillByCode(String workspaceId, String skillCode, Integer status);
}
