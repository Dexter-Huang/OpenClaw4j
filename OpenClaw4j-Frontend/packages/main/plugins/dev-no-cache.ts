import type { IApi } from 'umi';

const NO_STORE_HEADERS = {
  'Cache-Control': 'no-store, no-cache, must-revalidate, proxy-revalidate',
  Pragma: 'no-cache',
  Expires: '0',
};

export default (api: IApi) => {
  api.addBeforeMiddlewares(() => [
    (_req, res, next) => {
      Object.entries(NO_STORE_HEADERS).forEach(([key, value]) => {
        res.setHeader(key, value);
      });
      next();
    },
  ]);
};
