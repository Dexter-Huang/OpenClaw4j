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

package com.seaskyland.llm.workflow.core.workflow.processor.support;

import com.seaskyland.llm.workflow.core.base.manager.FileManager;
import com.seaskyland.llm.workflow.core.config.StudioProperties;
import com.seaskyland.llm.workflow.core.utils.common.VariableUtils;
import com.seaskyland.llm.workflow.core.workflow.WorkflowContext;
import com.seaskyland.llm.workflow.runtime.domain.file.File;
import com.seaskyland.llm.workflow.runtime.domain.workflow.Node;
import com.seaskyland.llm.workflow.runtime.domain.workflow.inner.ModelConfig;
import com.seaskyland.llm.workflow.runtime.enums.ErrorCode;
import com.seaskyland.llm.workflow.runtime.exception.BizException;
import java.net.URL;
import java.util.List;
import java.util.Objects;
import java.util.stream.Collectors;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.BooleanUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.ai.chat.messages.UserMessage;
import org.springframework.ai.content.Media;
import org.springframework.core.io.FileUrlResource;
import org.springframework.http.MediaType;
import org.springframework.util.MimeType;

/** 构造带视觉媒体的模型用户消息，供 LLM、意图分类等多模态节点复用。 */
@Slf4j
public final class VisionUserMessageFactory {

  private VisionUserMessageFactory() {}

  public static boolean hasEnabledVisionParams(ModelConfig modelConfig) {
    if (modelConfig == null || modelConfig.getVisionConfig() == null) {
      return false;
    }
    return BooleanUtils.isTrue(modelConfig.getVisionConfig().getEnable())
        && CollectionUtils.isNotEmpty(modelConfig.getVisionConfig().getParams());
  }

  public static UserMessage constructUserMessage(
      Node node,
      ModelConfig modelConfig,
      String userPrompt,
      WorkflowContext context,
      StudioProperties studioProperties,
      FileManager fileManager) {
    if (!hasEnabledVisionParams(modelConfig)) {
      return new UserMessage(userPrompt);
    }

    Object value =
        VariableUtils.getValueFromContext(modelConfig.getVisionConfig().getParams().get(0), context);
    if (value == null) {
      return new UserMessage(userPrompt);
    } else if (value instanceof File) {
      Media media = constructMedia(value, studioProperties, fileManager);
      if (media == null) {
        return new UserMessage(userPrompt);
      }
      return UserMessage.builder().text(userPrompt).media(media).build();
    } else if (value instanceof List) {
      List<Media> mediaList =
          ((List<?>) value)
              .stream()
                  .map(item -> constructMedia(item, studioProperties, fileManager))
                  .filter(Objects::nonNull)
                  .collect(Collectors.toList());
      if (CollectionUtils.isEmpty(mediaList)) {
        return new UserMessage(userPrompt);
      }
      return UserMessage.builder().text(userPrompt).media(mediaList).build();
    }

    throw new BizException(
        ErrorCode.WORKFLOW_CONFIG_INVALID.toError(
            node.getName() + " vision param is not File or List<File>"));
  }

  private static Media constructMedia(
      Object value, StudioProperties studioProperties, FileManager fileManager) {
    if (value == null) {
      return null;
    }
    if (!(value instanceof File)) {
      throw new BizException(ErrorCode.WORKFLOW_CONFIG_INVALID.toError("object is not File"));
    }
    File file = (File) value;
    String url = file.getUrl();
    String mimeType = file.getMimeType();
    if (StringUtils.isBlank(url)) {
      return null;
    }

    String source = file.getSource() == null ? File.SourceEnum.localFile.name() : file.getSource();
    try {
      if (File.SourceEnum.localFile.name().equals(source)) {
        String storagePath = studioProperties.getStoragePath();
        return new Media(
            MimeType.valueOf(mimeType),
            new FileUrlResource(storagePath + java.io.File.separator + url));
      }

      MediaType mediaType = fileManager.getMediaTypeFromUrl(url);
      return Media.builder()
          .mimeType(MimeType.valueOf(mediaType.toString()))
          .data(new URL(url))
          .build();
    } catch (Exception e) {
      log.error("Error processing local image: {}", url, e);
      throw new BizException(
          ErrorCode.WORKFLOW_EXECUTE_ERROR.toError(
              "Failed to process local image: " + e.getMessage()));
    }
  }
}
