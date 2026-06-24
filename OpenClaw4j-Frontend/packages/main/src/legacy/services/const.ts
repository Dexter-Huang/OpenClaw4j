export const API_PATH = '/api';

const trimTrailingSlash = (value: string) => value.replace(/\/+$/, '');

export const DIRECT_API_PATH =
  process.env.LEGACY_API_DIRECT === 'true' && process.env.WEB_SERVER
    ? `${trimTrailingSlash(process.env.WEB_SERVER)}/api`
    : API_PATH;
