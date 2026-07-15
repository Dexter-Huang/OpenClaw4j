import {
  getWorkFlowTaskProcess,
  resumeWorkFlowTask,
  stopWorkFlowTask,
} from '@/services/workflow';
import type { IWorkFlowTaskProcess } from '@spark-ai/flow';
import { useCallback, useEffect, useRef, useState } from 'react';
import { createWorkflowDebugTaskController } from './workflowDebugTaskController';
import type { WorkflowDebugTaskController } from './workflowDebugTaskController';

interface IWorkflowDebugTaskStartResponse {
  task_id: string;
}

interface IUseWorkflowDebugTaskOptions<
  TStartPayload,
  TStartResponse extends IWorkflowDebugTaskStartResponse,
> {
  createTask: (payload: TStartPayload) => Promise<TStartResponse>;
  normalizeTaskProcess?: (
    taskProcess: IWorkFlowTaskProcess,
  ) => IWorkFlowTaskProcess;
  onTaskCreated?: (response: TStartResponse) => void;
  onTaskProcess?: (taskProcess: IWorkFlowTaskProcess) => void;
  onTaskError?: (error: unknown) => void;
  onStopError?: (error: unknown) => void;
  keepLoadingOnPause?: boolean;
}

export function useWorkflowDebugTask<
  TStartPayload,
  TStartResponse extends IWorkflowDebugTaskStartResponse = IWorkflowDebugTaskStartResponse,
>(options: IUseWorkflowDebugTaskOptions<TStartPayload, TStartResponse>) {
  const optionsRef = useRef(options);
  const [loading, setLoading] = useState(false);
  const [taskInfo, setTaskInfo] = useState<IWorkFlowTaskProcess | null>(null);
  const [taskId, setTaskId] = useState<string | null>(null);

  optionsRef.current = options;

  const controllerRef =
    useRef<WorkflowDebugTaskController<TStartPayload, TStartResponse> | null>(
      null,
    );

  if (!controllerRef.current) {
    controllerRef.current = createWorkflowDebugTaskController<
      TStartPayload,
      TStartResponse
    >({
      createTask: (payload) => optionsRef.current.createTask(payload),
      getTaskProcess: getWorkFlowTaskProcess,
      resumeTask: resumeWorkFlowTask as (
        payload: Record<string, unknown>,
      ) => Promise<unknown>,
      stopTask: stopWorkFlowTask,
      normalizeTaskProcess: (taskProcess) =>
        optionsRef.current.normalizeTaskProcess?.(taskProcess) || taskProcess,
      onLoadingChange: setLoading,
      onTaskCreated: (response) => {
        optionsRef.current.onTaskCreated?.(response);
      },
      onTaskIdChange: setTaskId,
      onTaskProcess: (taskProcess) => {
        setTaskInfo(taskProcess);
        optionsRef.current.onTaskProcess?.(taskProcess);
      },
      onTaskError: (error) => {
        optionsRef.current.onTaskError?.(error);
      },
      onStopError: (error) => {
        optionsRef.current.onStopError?.(error);
      },
      keepLoadingOnPause: options.keepLoadingOnPause,
    });
  }

  useEffect(() => {
    return () => {
      controllerRef.current?.destroy();
    };
  }, []);

  const start = useCallback((payload: TStartPayload) => {
    return controllerRef.current!.start(payload);
  }, []);

  const resume = useCallback((payload: Record<string, unknown>) => {
    return controllerRef.current!.resume(payload);
  }, []);

  const stop = useCallback(() => {
    return controllerRef.current!.stop();
  }, []);

  const queryTaskStatus = useCallback(() => {
    return controllerRef.current!.queryTaskStatus();
  }, []);

  return {
    loading,
    queryTaskStatus,
    resume,
    start,
    stop,
    taskId,
    taskInfo,
  };
}
