/// *
// * Copyright 2025 the original author or authors.
// *
// * Licensed under the Apache License, Version 2.0 (the "License");
// * you may not use this file except in compliance with the License.
// * You may obtain a copy of the License at
// *
// *      https://www.apache.org/licenses/LICENSE-2.0
// *
// * Unless required by applicable law or agreed to in writing, software
// * distributed under the License is distributed on an "AS IS" BASIS,
// * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// * See the License for the specific language governing permissions and
// * limitations under the License.
// */
//
// package com.seaskyland.llm.workflow.admin.interceptor;
//
// import com.seaskyland.llm.workflow.runtime.enums.AccountType;
// import com.seaskyland.llm.workflow.runtime.enums.ErrorCode;
// import com.seaskyland.llm.workflow.runtime.domain.RequestContext;
// import com.seaskyland.llm.workflow.runtime.domain.Result;
// import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
// import com.seaskyland.llm.workflow.core.context.RequestContextHolder;
// import com.seaskyland.llm.workflow.core.utils.LogUtils;
// import com.seaskyland.llm.workflow.core.utils.common.IdGenerator;
// import jakarta.servlet.http.HttpServletRequest;
// import jakarta.servlet.http.HttpServletResponse;
// import lombok.RequiredArgsConstructor;
// import lombok.extern.slf4j.Slf4j;
// import org.jetbrains.annotations.NotNull;
// import org.springframework.stereotype.Component;
// import org.springframework.web.bind.annotation.RequestMethod;
// import org.springframework.web.servlet.HandlerInterceptor;
//
// import java.io.IOException;
//
/// **
// * Interceptor for handling token-based authentication for console API access. Validates
// * access tokens and sets up request context for authenticated users.
// *
// * @since 1.0.0.3
// */
// @Slf4j
// @Component
// @RequiredArgsConstructor
// public class TokenFakeAuthInterceptor implements HandlerInterceptor {
//
//	/**
//	 * Intercepts requests to validate authentication token and set up request context.
//	 * Returns true if authentication is successful, false otherwise.
//	 */
//	@Override
//	public boolean preHandle(@NotNull HttpServletRequest request, @NotNull HttpServletResponse
// response,
//			@NotNull Object handler) {
//		if (RequestMethod.OPTIONS.name().equals(request.getMethod().toUpperCase())) {
//			return true;
//		}
//		RequestContext context = new RequestContext();
//		context.setRequestId(IdGenerator.uuid());
//		context.setAccountId("10000");
//		context.setUsername("saa");
//		context.setWorkspaceId("1");
//		context.setAccountType(AccountType.ADMIN);
//		context.setCallerIp(request.getRemoteAddr());
//		context.setStartTime(System.currentTimeMillis());
//
//		RequestContextHolder.setRequestContext(context);
//		return true;
//	}
//
//	/**
//	 * Sends an unauthorized error response with the specified error code.
//	 * @param start Start time of the request for monitoring
//	 * @param response HTTP response object
//	 * @param errorCode Error code to be returned
//	 */
//	public void returnAuthError(long start, HttpServletResponse response, ErrorCode errorCode) {
//		response.setContentType("application/json;charset=UTF-8");
//		response.setStatus(HttpServletResponse.SC_UNAUTHORIZED);
//		Result<String> result = Result.error(IdGenerator.uuid(), errorCode);
//
//		LogUtils.monitor("AuthInterceptor", "TokenAuth", start, "unauthorized", "", result);
//
//		try {
//			response.getWriter().write(JsonUtils.toJson(result));
//		}
//		catch (IOException e) {
//			LogUtils.error("failed to unauthorized message: {}", result, e);
//		}
//	}
//
// }
