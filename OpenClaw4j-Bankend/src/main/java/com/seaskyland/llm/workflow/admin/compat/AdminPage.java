package com.seaskyland.llm.workflow.admin.compat;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.List;

@Data
@NoArgsConstructor
@AllArgsConstructor
public class AdminPage<T> {

	private Long totalCount;

	private Long totalPage;

	private Long pageNumber;

	private Long pageSize;

	private List<T> pageItems;

	public static <T> AdminPage<T> of(long totalCount, long pageNumber, long pageSize, List<T> pageItems) {
		long safePageSize = pageSize <= 0 ? 10 : pageSize;
		long totalPage = totalCount == 0 ? 0 : (totalCount + safePageSize - 1) / safePageSize;
		return new AdminPage<>(totalCount, totalPage, pageNumber <= 0 ? 1 : pageNumber, safePageSize, pageItems);
	}

}
