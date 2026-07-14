export interface IApiResponse<T = any> {
  data: T;
  request_id: string;
}

export interface IPageParams {
  current?: number;
  page?: number;
  pageSize?: number;
  size?: number;
}

export interface IPageResult<T = any> {
  records: T[];
  total: number;
  current?: number;
  page?: number;
  pageSize?: number;
  size?: number;
}
