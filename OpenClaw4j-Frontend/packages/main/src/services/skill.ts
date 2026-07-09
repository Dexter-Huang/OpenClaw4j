import { request } from '@/request';
import { IApiResponse } from '@/types/common';
import type {
  ICreateSkillParams,
  IGetSkillParams,
  IListSkillsByCodesParams,
  IListSkillsParams,
  ISkill,
  ISkillPackageUploadResult,
  ISkillPagingList,
  IUpdateSkillParams,
} from '@/types/skill';

export async function createSkill(
  params: ICreateSkillParams,
): Promise<IApiResponse<string>> {
  const response = await request({
    url: '/console/v1/skills',
    method: 'POST',
    data: params,
  });
  return response.data as IApiResponse<string>;
}

export async function uploadSkillPackage(
  file: File,
): Promise<IApiResponse<ISkillPackageUploadResult>> {
  const formData = new FormData();
  formData.append('file', file);
  const response = await request({
    url: '/console/v1/skills/package',
    method: 'POST',
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  });
  return response.data as IApiResponse<ISkillPackageUploadResult>;
}

export async function updateSkill(
  params: IUpdateSkillParams,
): Promise<IApiResponse<string>> {
  const response = await request({
    url: '/console/v1/skills',
    method: 'PUT',
    data: params,
  });
  return response.data as IApiResponse<string>;
}

export async function deleteSkill(
  skillCode: string,
): Promise<IApiResponse<null>> {
  const response = await request({
    url: `/console/v1/skills/${skillCode}`,
    method: 'DELETE',
  });
  return response.data as IApiResponse<null>;
}

export async function publishSkill(
  skillCode: string,
): Promise<IApiResponse<null>> {
  const response = await request({
    url: `/console/v1/skills/${skillCode}/publish`,
    method: 'POST',
  });
  return response.data as IApiResponse<null>;
}

export async function getSkill(
  params: IGetSkillParams,
): Promise<IApiResponse<ISkill>> {
  const response = await request({
    url: `/console/v1/skills/${params.skill_code}`,
    method: 'GET',
    params,
  });
  return response.data as IApiResponse<ISkill>;
}

export async function listSkills(
  params: IListSkillsParams,
): Promise<IApiResponse<ISkillPagingList>> {
  const response = await request({
    url: '/console/v1/skills',
    method: 'GET',
    params,
  });
  return response.data as IApiResponse<ISkillPagingList>;
}

export async function listSkillsByCodes(
  params: IListSkillsByCodesParams,
): Promise<IApiResponse<ISkill[]>> {
  const response = await request({
    url: '/console/v1/skills/query-by-codes',
    method: 'POST',
    data: params,
  });
  return response.data as IApiResponse<ISkill[]>;
}
