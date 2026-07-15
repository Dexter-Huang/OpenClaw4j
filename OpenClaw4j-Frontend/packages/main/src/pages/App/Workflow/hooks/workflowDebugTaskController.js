function noop() {}

function markStoppedResults(results) {
  if (!Array.isArray(results)) return results;

  return results.map((item) => {
    if (!['executing', 'pause'].includes(item.node_status)) return item;
    return {
      ...item,
      node_status: 'stop',
    };
  });
}

function createStoppedTaskProcess(taskProcess) {
  if (!taskProcess) return taskProcess;

  return {
    ...taskProcess,
    task_status: 'stop',
    node_results: markStoppedResults(taskProcess.node_results),
    task_results: markStoppedResults(taskProcess.task_results),
  };
}

function createWorkflowDebugTaskController(options) {
  const {
    createTask,
    getTaskProcess,
    resumeTask,
    stopTask,
    normalizeTaskProcess = (process) => process,
    onLoadingChange = noop,
    onTaskCreated = noop,
    onTaskIdChange = noop,
    onTaskProcess = noop,
    onTaskError = noop,
    onStopError = noop,
    schedule = setTimeout,
    clearSchedule = clearTimeout,
    pollInterval = 500,
    keepLoadingOnPause = false,
  } = options;

  let timer = null;
  let taskId = null;
  let latestTaskProcess = null;
  let destroyed = false;
  let taskVersion = 0;
  const reportedTaskErrors = new Set();

  const clearTimer = () => {
    if (!timer) return;
    clearSchedule(timer);
    timer = null;
  };

  const setLoading = (loading) => {
    onLoadingChange(loading);
  };

  const setTaskId = (nextTaskId) => {
    taskId = nextTaskId || null;
    onTaskIdChange(taskId);
  };

  const setLatestTaskProcess = (taskProcess) => {
    latestTaskProcess = taskProcess || null;
  };

  const emitTaskProcess = (taskProcess) => {
    const nextTaskProcess = normalizeTaskProcess(taskProcess);
    setLatestTaskProcess(nextTaskProcess);
    onTaskProcess(nextTaskProcess);
    return nextTaskProcess;
  };

  const isCurrentTaskVersion = (version) => !destroyed && version === taskVersion;

  const reportTaskError = (error, version) => {
    if (!isCurrentTaskVersion(version)) return;
    if (reportedTaskErrors.has(error)) return;
    reportedTaskErrors.add(error);
    setLoading(false);
    onTaskError(error);
  };

  const queryTaskStatus = async (version = taskVersion) => {
    clearTimer();
    if (!taskId || !isCurrentTaskVersion(version)) return null;

    let taskProcess;
    try {
      taskProcess = await getTaskProcess({
        task_id: taskId,
      });
    } catch (error) {
      reportTaskError(error, version);
      throw error;
    }
    if (!isCurrentTaskVersion(version)) return null;

    const nextTaskProcess = emitTaskProcess(taskProcess);
    if (nextTaskProcess.task_status === 'executing') {
      timer = schedule(() => queryTaskStatus(version), pollInterval);
    } else {
      if (nextTaskProcess.task_status !== 'pause' || !keepLoadingOnPause) {
        setLoading(false);
      }
      clearTimer();
    }
    return nextTaskProcess;
  };

  const start = async (payload) => {
    const version = taskVersion + 1;
    taskVersion = version;
    clearTimer();
    setLatestTaskProcess(null);
    setLoading(true);
    try {
      const response = await createTask(payload);
      if (!isCurrentTaskVersion(version)) return response;
      setTaskId(response.task_id);
      onTaskCreated(response);
      await queryTaskStatus(version);
      return response;
    } catch (error) {
      if (isCurrentTaskVersion(version)) {
        setTaskId(null);
        reportTaskError(error, version);
      }
      throw error;
    }
  };

  const resume = async (payload) => {
    if (!taskId) return null;
    const version = taskVersion + 1;
    taskVersion = version;
    setLoading(true);
    try {
      await resumeTask({
        ...payload,
        task_id: taskId,
      });
      return queryTaskStatus(version);
    } catch (error) {
      reportTaskError(error, version);
      throw error;
    }
  };

  const stop = async () => {
    const version = taskVersion + 1;
    taskVersion = version;
    clearTimer();
    const currentTaskId = taskId;
    try {
      if (currentTaskId) {
        await stopTask({
          task_id: currentTaskId,
        });
      }
    } catch (error) {
      onStopError(error);
    } finally {
      if (isCurrentTaskVersion(version)) {
        setLoading(false);
      }
    }

    if (isCurrentTaskVersion(version) && latestTaskProcess) {
      emitTaskProcess(createStoppedTaskProcess(latestTaskProcess));
    }
  };

  const destroy = () => {
    destroyed = true;
    taskVersion += 1;
    clearTimer();
  };

  return {
    destroy,
    queryTaskStatus,
    resume,
    setLatestTaskProcess,
    setTaskId,
    setTimer: (nextTimer) => {
      timer = nextTimer;
    },
    start,
    stop,
  };
}

module.exports = {
  createStoppedTaskProcess,
  createWorkflowDebugTaskController,
};
