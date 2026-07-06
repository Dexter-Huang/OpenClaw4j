import type { IWorkFlowNodeResultItem } from '@spark-ai/flow';

export interface IUsageSummary {
  input: number;
  output: number;
  total: number;
}

export function getUsageSummary(
  usages?: IWorkFlowNodeResultItem['usages'] | null,
): IUsageSummary;
