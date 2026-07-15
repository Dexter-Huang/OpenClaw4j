/* eslint-env jest */

const {
  createStoppedTaskProcess,
  createWorkflowDebugTaskController,
} = require('./workflowDebugTaskController');

function createDeferred() {
  const deferred = {};
  deferred.promise = new Promise((resolve, reject) => {
    deferred.resolve = resolve;
    deferred.reject = reject;
  });
  return deferred;
}

describe('workflow debug task controller', () => {
  it('marks executing and paused task results as stopped', () => {
    expect(
      createStoppedTaskProcess({
        task_status: 'executing',
        node_results: [
          { node_id: 'node-1', node_status: 'executing' },
          { node_id: 'node-2', node_status: 'pause' },
          { node_id: 'node-3', node_status: 'success' },
        ],
        task_results: [
          { node_id: 'node-1', node_status: 'executing' },
          { node_id: 'node-2', node_status: 'pause' },
        ],
      }),
    ).toEqual({
      task_status: 'stop',
      node_results: [
        { node_id: 'node-1', node_status: 'stop' },
        { node_id: 'node-2', node_status: 'stop' },
        { node_id: 'node-3', node_status: 'success' },
      ],
      task_results: [
        { node_id: 'node-1', node_status: 'stop' },
        { node_id: 'node-2', node_status: 'stop' },
      ],
    });
  });

  it('polls executing tasks and stops loading when the task finishes', async () => {
    const loadingChanges = [];
    const processUpdates = [];
    const scheduledCallbacks = [];
    const controller = createWorkflowDebugTaskController({
      createTask: jest.fn().mockResolvedValue({ task_id: 'task-1' }),
      getTaskProcess: jest
        .fn()
        .mockResolvedValueOnce({
          task_status: 'executing',
          node_results: [],
          task_results: [],
        })
        .mockResolvedValueOnce({
          task_status: 'success',
          node_results: [],
          task_results: [],
        }),
      resumeTask: jest.fn(),
      stopTask: jest.fn(),
      onLoadingChange: (loading) => loadingChanges.push(loading),
      onTaskProcess: (process) => processUpdates.push(process.task_status),
      schedule: (callback) => {
        scheduledCallbacks.push(callback);
        return `timer-${scheduledCallbacks.length}`;
      },
      clearSchedule: jest.fn(),
    });

    await controller.start({ app_id: 'app-1', inputs: [] });
    await scheduledCallbacks[0]();

    expect(loadingChanges).toEqual([true, false]);
    expect(processUpdates).toEqual(['executing', 'success']);
  });

  it('calls the backend stop endpoint, clears polling, and emits a stopped process', async () => {
    const stopTask = jest.fn().mockResolvedValue(true);
    const clearSchedule = jest.fn();
    const processUpdates = [];
    const controller = createWorkflowDebugTaskController({
      createTask: jest.fn(),
      getTaskProcess: jest.fn(),
      resumeTask: jest.fn(),
      stopTask,
      onLoadingChange: jest.fn(),
      onTaskProcess: (process) => processUpdates.push(process),
      schedule: jest.fn(),
      clearSchedule,
    });

    controller.setTaskId('task-1');
    controller.setLatestTaskProcess({
      task_status: 'executing',
      node_results: [{ node_id: 'node-1', node_status: 'executing' }],
      task_results: [{ node_id: 'node-1', node_status: 'pause' }],
    });
    controller.setTimer('timer-1');

    await controller.stop();

    expect(stopTask).toHaveBeenCalledWith({ task_id: 'task-1' });
    expect(clearSchedule).toHaveBeenCalledWith('timer-1');
    expect(processUpdates).toEqual([
      {
        task_status: 'stop',
        node_results: [{ node_id: 'node-1', node_status: 'stop' }],
        task_results: [{ node_id: 'node-1', node_status: 'stop' }],
      },
    ]);
  });

  it('stops loading and reports the error when task creation fails', async () => {
    const error = new Error('create failed');
    const loadingChanges = [];
    const taskErrors = [];
    const controller = createWorkflowDebugTaskController({
      createTask: jest.fn().mockRejectedValue(error),
      getTaskProcess: jest.fn(),
      resumeTask: jest.fn(),
      stopTask: jest.fn(),
      onLoadingChange: (loading) => loadingChanges.push(loading),
      onTaskError: (receivedError) => taskErrors.push(receivedError),
    });

    await expect(controller.start({ app_id: 'app-1', inputs: [] })).rejects.toBe(
      error,
    );

    expect(loadingChanges).toEqual([true, false]);
    expect(taskErrors).toEqual([error]);
  });

  it('stops loading and reports the error when polling fails', async () => {
    const error = new Error('poll failed');
    const loadingChanges = [];
    const taskErrors = [];
    const controller = createWorkflowDebugTaskController({
      createTask: jest.fn().mockResolvedValue({ task_id: 'task-1' }),
      getTaskProcess: jest.fn().mockRejectedValue(error),
      resumeTask: jest.fn(),
      stopTask: jest.fn(),
      onLoadingChange: (loading) => loadingChanges.push(loading),
      onTaskError: (receivedError) => taskErrors.push(receivedError),
    });

    await expect(controller.start({ app_id: 'app-1', inputs: [] })).rejects.toBe(
      error,
    );

    expect(loadingChanges).toEqual([true, false]);
    expect(taskErrors).toEqual([error]);
  });

  it('ignores a stale polling result after a newer task starts', async () => {
    const firstPoll = createDeferred();
    const processUpdates = [];
    const controller = createWorkflowDebugTaskController({
      createTask: jest
        .fn()
        .mockResolvedValueOnce({ task_id: 'task-1' })
        .mockResolvedValueOnce({ task_id: 'task-2' }),
      getTaskProcess: jest
        .fn()
        .mockReturnValueOnce(firstPoll.promise)
        .mockResolvedValueOnce({
          task_status: 'success',
          node_results: [{ node_id: 'node-2' }],
          task_results: [],
        }),
      resumeTask: jest.fn(),
      stopTask: jest.fn(),
      onLoadingChange: jest.fn(),
      onTaskProcess: (process) => processUpdates.push(process),
    });

    const firstStart = controller.start({ app_id: 'app-1', inputs: [] });
    await Promise.resolve();
    await Promise.resolve();
    await controller.start({ app_id: 'app-1', inputs: [] });

    firstPoll.resolve({
      task_status: 'success',
      node_results: [{ node_id: 'node-1' }],
      task_results: [],
    });
    await firstStart;

    expect(processUpdates).toEqual([
      {
        task_status: 'success',
        node_results: [{ node_id: 'node-2' }],
        task_results: [],
      },
    ]);
  });

  it('ignores an in-flight polling result after the task is stopped', async () => {
    const poll = createDeferred();
    const processUpdates = [];
    const controller = createWorkflowDebugTaskController({
      createTask: jest.fn(),
      getTaskProcess: jest.fn().mockReturnValue(poll.promise),
      resumeTask: jest.fn(),
      stopTask: jest.fn().mockResolvedValue(true),
      onLoadingChange: jest.fn(),
      onTaskProcess: (process) => processUpdates.push(process),
    });

    controller.setTaskId('task-1');
    controller.setLatestTaskProcess({
      task_status: 'executing',
      node_results: [{ node_id: 'node-1', node_status: 'executing' }],
      task_results: [],
    });

    const pendingQuery = controller.queryTaskStatus();
    await Promise.resolve();
    await controller.stop();
    poll.resolve({
      task_status: 'success',
      node_results: [{ node_id: 'node-1', node_status: 'success' }],
      task_results: [],
    });
    await pendingQuery;

    expect(processUpdates).toEqual([
      {
        task_status: 'stop',
        node_results: [{ node_id: 'node-1', node_status: 'stop' }],
        task_results: [],
      },
    ]);
  });
});
