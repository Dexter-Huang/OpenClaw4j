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

package com.seaskyland.llm.workflow.core.base.manager;

import static org.assertj.core.api.Assertions.assertThat;

import com.seaskyland.llm.workflow.runtime.domain.Result;
import com.seaskyland.llm.workflow.runtime.domain.mcp.McpServerDeployConfig;
import com.seaskyland.llm.workflow.runtime.enums.ErrorCode;
import com.seaskyland.llm.workflow.runtime.enums.McpInstallTypeEnum;
import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
import org.junit.jupiter.api.Test;

class MCPManagerTest {

  private final MCPManager manager = new MCPManager(null);

  @Test
  void processInstallConfigAcceptsStreamableHttpMcpEndpoint() {
    String config =
        """
        {
          "mcpServers": {
            "aio-sandbox": {
              "url": "http://localhost:8080/mcp",
              "headers": {
                "Authorization": "Bearer test-token"
              }
            }
          }
        }
        """;

    Result<String> result =
        manager.processInstallConfig(config, McpInstallTypeEnum.STREAMABLE_HTTP.name());

    assertThat(result.isSuccess()).isTrue();
    McpServerDeployConfig deployConfig =
        JsonUtils.fromJson(result.getData(), McpServerDeployConfig.class);
    assertThat(deployConfig.getRemoteAddress()).isEqualTo("http://localhost:8080");
    assertThat(deployConfig.getRemoteEndpoint()).isEqualTo("/mcp");
    assertThat(deployConfig.getRemoteHeader()).containsEntry("Authorization", "Bearer test-token");
    assertThat(JsonUtils.fromJsonToMap(deployConfig.getInstallConfig())).containsKey("mcpServers");
  }

  @Test
  void processInstallConfigKeepsSseEndpointRequirementForSseInstallType() {
    String config =
        """
        {
          "mcpServers": {
            "aio-sandbox": {
              "url": "http://localhost:8080/mcp",
              "headers": {}
            }
          }
        }
        """;

    Result<String> result = manager.processInstallConfig(config, McpInstallTypeEnum.SSE.name());

    assertThat(result.isSuccess()).isFalse();
    assertThat(result.getCode()).isEqualTo(ErrorCode.MCP_PARSE_URL_ERROR.getCode());
  }

  @Test
  void processInstallConfigStillAcceptsSseEndpointForSseInstallType() {
    String config =
        """
        {
          "mcpServers": {
            "legacy-server": {
              "url": "http://localhost:9000/sse",
              "headers": {}
            }
          }
        }
        """;

    Result<String> result = manager.processInstallConfig(config, McpInstallTypeEnum.SSE.name());

    assertThat(result.isSuccess()).isTrue();
    McpServerDeployConfig deployConfig =
        JsonUtils.fromJson(result.getData(), McpServerDeployConfig.class);
    assertThat(deployConfig.getRemoteAddress()).isEqualTo("http://localhost:9000");
    assertThat(deployConfig.getRemoteEndpoint()).isEqualTo("/sse");
  }
}
