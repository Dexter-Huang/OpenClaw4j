package com.seaskyland.llm.workflow.core.context;

import com.seaskyland.llm.workflow.core.utils.concurrent.RequestContextThreadPoolWrapper;
import com.seaskyland.llm.workflow.runtime.domain.RequestContext;
import org.junit.jupiter.api.Test;

import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertSame;

class RequestContextHolderTest {

	@Test
	void requestContextIsOnlyVisibleInsideScopedBinding() throws Exception {
		RequestContext context = requestContext("account-1");

		assertNull(RequestContextHolder.getRequestContext());

		RequestContextHolder.callWithRequestContext(context, () -> {
			assertSame(context, RequestContextHolder.getRequestContext());
			return null;
		});

		assertNull(RequestContextHolder.getRequestContext());
	}

	@Test
	void wrappedExecutorRunsTasksInsideCapturedScopedBinding() throws Exception {
		ExecutorService delegate = Executors.newSingleThreadExecutor();
		RequestContextThreadPoolWrapper executor = new RequestContextThreadPoolWrapper(delegate);
		RequestContext context = requestContext("account-2");

		try {
			RequestContextHolder.callWithRequestContext(context, () -> {
				Future<String> future = executor.submit(() -> RequestContextHolder.getRequestContext().getAccountId());
				assertEquals("account-2", future.get());
				return null;
			});

			Future<RequestContext> future = executor.submit(RequestContextHolder::getRequestContext);
			assertNull(future.get());
		}
		finally {
			executor.shutdownNow();
		}
	}

	private RequestContext requestContext(String accountId) {
		RequestContext context = new RequestContext();
		context.setAccountId(accountId);
		return context;
	}

}
