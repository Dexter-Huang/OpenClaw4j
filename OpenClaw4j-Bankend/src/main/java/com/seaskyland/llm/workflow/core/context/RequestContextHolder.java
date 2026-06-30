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

package com.seaskyland.llm.workflow.core.context;

import com.seaskyland.llm.workflow.runtime.domain.RequestContext;

import java.util.concurrent.Callable;

/**
 * A holder class for managing request context in a lexically scoped manner.
 *
 * @since 1.0.0.3
 */
public class RequestContextHolder {

	/**
	 * Scoped request context for the current execution path.
	 */
	private static final ScopedValue<RequestContext> REQUEST_CONTEXT = ScopedValue.newInstance();

	/**
	 * Runs the operation with the request context bound to the current lexical scope.
	 * @param requestContext the context to bind
	 * @param operation the operation to run
	 */
	public static void runWithRequestContext(RequestContext requestContext, ScopedOperation operation) throws Exception {
		callWithRequestContext(requestContext, () -> {
			operation.run();
			return null;
		});
	}

	/**
	 * Calls the operation with the request context bound to the current lexical scope.
	 * @param requestContext the context to bind
	 * @param operation the operation to call
	 * @return the operation result
	 */
	public static <T> T callWithRequestContext(RequestContext requestContext, Callable<T> operation) throws Exception {
		if (requestContext == null) {
			return operation.call();
		}
		return ScopedValue.where(REQUEST_CONTEXT, requestContext).call(() -> operation.call());
	}

	/**
	 * Gets the request context for the current lexical scope.
	 * @return the current request context, or null when no context is bound
	 */
	public static RequestContext getRequestContext() {
		return REQUEST_CONTEXT.isBound() ? REQUEST_CONTEXT.get() : null;
	}

	/**
	 * Scoped operation that may throw checked exceptions.
	 */
	@FunctionalInterface
	public interface ScopedOperation {

		void run() throws Exception;

	}

}
