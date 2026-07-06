import { IWorkFlowNodeResultItem } from '@spark-flow/types/work-flow';

export interface ITokenDetail {
  id: string;
  name: string;
  type: string;
  input: number;
  output: number;
}

export function getTaskTokenDetails(
  nodeResults?: IWorkFlowNodeResultItem[] | null,
): ITokenDetail[];
