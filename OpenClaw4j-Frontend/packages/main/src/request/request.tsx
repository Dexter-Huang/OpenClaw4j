import $i18n from '@/i18n';
import { notification, Space } from 'antd';
import axios, { AxiosRequestConfig, AxiosResponse } from 'axios';
import styles from './index.module.less';
import { session } from './session';
export { session };

// Base URL management
let _baseURL = '';
export const baseURL = {
  // Set the base URL
  set(v: string) {
    _baseURL = v;
  },
  // Get the current base URL
  get() {
    return _baseURL;
  },
};

// Initialize base URL from environment variable
baseURL.set(process.env.WEB_SERVER || '');

// Create axios instance with default configuration
export const instance = axios.create({
  baseURL: baseURL.get(),
  timeout: 60 * 1000, // 60 seconds timeout
  headers: {},
});

/**
 * Get base headers including authorization and language
 * @returns Promise with headers object
 */
async function getBaseHeader() {
  const header: {
    // Authorization?: string;
    'Accept-Language'?: string;
    'X-SAA-TOKEN'?: string;
  } = {
    'Accept-Language': $i18n.getCurrentLanguage(),
  };

  // Get session token if available
  const token = await session.asyncGet();

  if (token) {
    // header.Authorization = 'Bearer ' + token;
    header['X-SAA-TOKEN'] = 'Bearer ' + token;
  }
  return header;
}

/**
 * Show error notification
 * @param error Error object containing response data
 */
function notificationError(error: any) {
  notification.error({
    className: styles.error,
    message: error.response?.data?.code || error.message,
    description: (
      <Space direction="vertical">
        <div>{error.response?.data?.message}</div>
        <div>{error.response?.data?.request_id}</div>
      </Space>
    ),
  });
}

/**
 * Main fetch function with error handling
 * @param config Axios request configuration with optional autoMsg flag
 * @returns Promise with response data
 */
export default async function fetch<T = any>(
  config: AxiosRequestConfig & { autoMsg?: boolean },
): Promise<AxiosResponse<T>>;
export default async function fetch<T = any>(
  url: string,
  config?: AxiosRequestConfig & { autoMsg?: boolean },
): Promise<AxiosResponse<T>>;
export default async function fetch<T = any>(
  urlOrConfig: string | (AxiosRequestConfig & { autoMsg?: boolean }),
  config: AxiosRequestConfig & { autoMsg?: boolean } = {},
): Promise<AxiosResponse<T>> {
  const requestConfig =
    typeof urlOrConfig === 'string'
      ? { ...config, url: urlOrConfig }
      : urlOrConfig;
  const { headers = {}, autoMsg = true } = requestConfig;

  // Internal request function with merged headers
  async function request() {
    return instance.request({
      ...requestConfig,
      headers: {
        ...headers,
        ...(await getBaseHeader()),
      },
    });
  }

  return await request().catch((error) => {
    console.error('Request failed:', error);
    autoMsg && notificationError(error);
    throw error;
  });
}
