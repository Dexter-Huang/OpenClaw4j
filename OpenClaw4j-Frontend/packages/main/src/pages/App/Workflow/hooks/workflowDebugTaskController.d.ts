import type { IWorkFlowTaskProcess } from '@spark-ai/flow';

export interface WorkflowDebugTaskControllerOptions<TStartPayload, TStartResponse> {
  createTask: (payload: TStartPayload) => Promise<TStartResponse>;
  getTaskProcess: (payload: { task_id: string }) => Promise<IWorkFlowTaskProcess>;
  resumeTask: (payload: Record<string, unknown>) => Promise<unknown>;
  stopTask: (payload: { task_id: string }) => Promise<unknown>;
  normalizeTaskProcess?: (
    taskProcess: IWorkFlowTaskProcess,
  ) => IWorkFlowTaskProcess;
  onLoadingChange?: (loading: boolean) => void;
  onTaskCreated?: (response: TStartResponse) => void;
  onTaskIdChange?: (taskId: string | null) => void;
  onTaskProcess?: (taskProcess: IWorkFlowTaskProcess) => void;
  onTaskError?: (error: unknown) => void;
  onStopError?: (error: unknown) => void;
  schedule?: (callback: () => void, delay: number) => unknown;
  clearSchedule?: (timer: unknown) => void;
  pollInterval?: number;
  keepLoadingOnPause?: boolean;
}

export interface WorkflowDebugTaskController<TStartPayload, TStartResponse> {
  destroy: () => void;
  queryTaskStatus: () => Promise<IWorkFlowTaskProcess | null>;
  resume: (payload: Record<string, unknown>) => Promise<IWorkFlowTaskProcess | null>;
  setLatestTaskProcess: (taskProcess: IWorkFlowTaskProcess | null) => void;
  setTaskId: (taskId: string | null) => void;
  setTimer: (timer: unknown) => void;
  start: (payload: TStartPayload) => Promise<TStartResponse>;
  stop: () => Promise<void>;
}

export function createStoppedTaskProcess<T extends IWorkFlowTaskProcess | null | undefined>(
  taskProcess: T,
): T;

export function createWorkflowDebugTaskController<
  TStartPayload,
  TStartResponse extends { task_id: string },
>(
  options: WorkflowDebugTaskControllerOptions<TStartPayload, TStartResponse>,
): WorkflowDebugTaskController<TStartPayload, TStartResponse>;
