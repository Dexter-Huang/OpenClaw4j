import $i18n from '@/i18n';
import { IPagingList } from '@/types/mcp';

export enum SkillStatus {
  DELETED = 0,
  DRAFT = 1,
  PUBLISHED = 2,
  PUBLISHED_EDITING = 3,
}

export const SkillStatusMap = {
  [SkillStatus.DRAFT]: {
    color: 'default',
    text: $i18n.get({ id: 'main.types.skill.draft', dm: '草稿' }),
  },
  [SkillStatus.PUBLISHED]: {
    color: 'success',
    text: $i18n.get({ id: 'main.types.skill.published', dm: '已发布' }),
  },
  [SkillStatus.PUBLISHED_EDITING]: {
    color: 'processing',
    text: $i18n.get({
      id: 'main.types.skill.publishedEditing',
      dm: '已发布编辑中',
    }),
  },
  [SkillStatus.DELETED]: {
    color: 'error',
    text: $i18n.get({ id: 'main.types.skill.deleted', dm: '已删除' }),
  },
};

export interface ISkill {
  skill_code: string;
  name: string;
  description?: string;
  source?: string;
  status: SkillStatus;
  current_version?: string;
  version?: string;
  tags?: string;
  main_file_path?: string;
  manifest?: string;
  content_hash?: string;
  storage_type?: 'file' | 'oss';
  storage_bucket?: string;
  storage_prefix?: string;
  package_object_key?: string;
  file_count?: number;
  total_size_bytes?: number;
  need_files?: boolean;
  gmt_modified?: string;
}

export interface ISkillPackageUploadResult {
  storage_type?: 'file' | 'oss';
  storage_bucket?: string;
  storage_prefix?: string;
  package_object_key: string;
  main_file_path?: string;
  manifest?: string;
  content_hash?: string;
  file_count?: number;
  total_size_bytes?: number;
}

export interface ICreateSkillParams {
  name: string;
  description?: string;
  version?: string;
  tags?: string;
  main_file_path?: string;
  manifest?: string;
  storage_type?: 'file' | 'oss';
  storage_bucket?: string;
  storage_prefix?: string;
  package_object_key?: string;
  content_hash?: string;
  file_count?: number;
  total_size_bytes?: number;
  status?: SkillStatus;
}

export interface IUpdateSkillParams extends ICreateSkillParams {
  skill_code: string;
}

export interface IGetSkillParams {
  skill_code: string;
  version?: string;
  need_files?: boolean;
}

export interface IListSkillsParams {
  current?: number;
  size?: number;
  total?: number;
  status?: SkillStatus;
  need_files?: boolean;
  name?: string;
}

export interface IListSkillsByCodesParams {
  skill_codes: string[];
  need_files: boolean;
}

export type ISkillPagingList = IPagingList<ISkill>;
