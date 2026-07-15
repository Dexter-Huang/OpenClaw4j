import $i18n from '@/i18n';
import Welcome from '@/pages/App/AssistantAppEdit/components/SparkChat/components/Welcome';
import { createWorkFlowTask } from '@/services/workflow';
import { Markdown } from '@spark-ai/chat';
import { Button } from '@spark-ai/design';
import {
  IWorkFlowTaskProcess,
  useFlowDebugInteraction,
  useFlowInteraction,
  useStore,
} from '@spark-ai/flow';
import { useSetState } from 'ahooks';
import { Flex, Segmented, Tabs } from 'antd';
import classNames from 'classnames';
import { memo, useMemo, useRef } from 'react';
import { useWorkflowAppStore } from '../../context/WorkflowAppProvider';
import { useWorkflowDebugTask } from '../../hooks/useWorkflowDebugTask';
import { NodeResultPanelList } from '../NodeResultPanel';
import ResultStatus from '../ResultStatus';
import UserInputForm, { IUserInputSubmitParams } from '../UserInputForm';
import styles from './index.module.less';
import InputParamsForm from './InputParamsForm';

export const TextCard = ({ data }: { data: { text: string } }) => {
  return (
    <div className={styles['result-panel-content']}>
      <Markdown baseFontSize={12} content={data.text || ''} />
    </div>
  );
};

export const getSparkFlowUsageList = (data?: IWorkFlowTaskProcess | null) => {
  if (!data) return [];
  let newList: {
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
  }[] = [];

  data.node_results.forEach((item) => {
    if (item.usages) {
      newList = [...newList, ...item.usages];
    }
  });
  return newList;
};

export default memo(function TaskTestPanel() {
  const inputParams = useWorkflowAppStore((state) => state.debugInputParams);
  const { updateTaskStore } = useFlowDebugInteraction();
  const setSelectedNode = useStore((state) => state.setSelectedNode);
  const setShowResults = useStore((state) => state.setShowResults);
  const appId = useWorkflowAppStore((state) => state.appId);
  const { focusElement } = useFlowInteraction();
  const [state, setState] = useSetState({
    activeTab: 'input',
  });
  const resultContainerRef = useRef<HTMLDivElement>(null);
  const cacheAnimateNodeId = useRef('');

  const scrollToBottom = (forceScroll: boolean = false) => {
    if (!resultContainerRef.current) return;

    const container = resultContainerRef.current;
    const isAtBottom =
      container.scrollHeight - container.scrollTop - container.offsetHeight <=
      100;

    if (forceScroll || isAtBottom) {
      const timer = setTimeout(() => {
        container.scrollTo({
          top: container.scrollHeight - container.offsetHeight,
          left: 0,
          behavior: 'smooth',
        });
        clearTimeout(timer);
      }, 100);
    }
  };

  const workflowTask = useWorkflowDebugTask({
    createTask: createWorkFlowTask,
    onTaskCreated: () => {
      setShowResults(true);
    },
    onTaskProcess: (res) => {
      const endNode = res.node_results[res.node_results.length - 1];
      if (endNode && endNode.node_id !== cacheAnimateNodeId.current) {
        focusElement({ nodeId: endNode.node_id });
        cacheAnimateNodeId.current = endNode.node_id;
      }
      updateTaskStore(res);
      scrollToBottom(
        res.task_results.some(
          (item) => item.node_type === 'Input' && item.node_status === 'pause',
        ),
      );
    },
  });

  const handleTest = () => {
    setSelectedNode(null);
    setState({
      activeTab: 'result',
    });
    workflowTask.start({
      app_id: appId,
      inputs: inputParams,
    });
  };

  const usageList = useMemo(() => {
    return getSparkFlowUsageList(workflowTask.taskInfo);
  }, [workflowTask.taskInfo]);

  const handleSubmitUserInput = (params: IUserInputSubmitParams) => {
    if (!workflowTask.taskId) return;
    setSelectedNode(null);
    workflowTask.resume({
      app_id: appId,
      ...params,
    });
  };

  return (
    <div className="h-full flex flex-col gap-[12px]">
      <Segmented
        value={state.activeTab}
        className={styles.segmented}
        onChange={(value) => setState({ activeTab: value })}
        options={[
          {
            label: $i18n.get({
              id: 'main.pages.App.Workflow.components.TaskTestPanel.index.input',
              dm: '输入',
            }),
            value: 'input',
          },
          {
            label: $i18n.get({
              id: 'main.pages.App.Workflow.components.TaskTestPanel.index.result',
              dm: '结果',
            }),
            value: 'result',
          },
        ]}
      />

      <Tabs
        className={styles.tabs}
        activeKey={state.activeTab}
        items={[
          {
            label: $i18n.get({
              id: 'main.pages.App.Workflow.components.TaskTestPanel.index.input',
              dm: '输入',
            }),
            key: 'input',
            children: (
              <div className="h-full flex flex-col gap-[12px] pb-[20px]">
                <InputParamsForm showPadding />
                <Button
                  className={classNames(
                    'mx-[20px] gap-[8px]',
                    styles['test-button'],
                  )}
                  loading={workflowTask.loading}
                  type="primaryLess"
                  onClick={handleTest}
                >
                  {workflowTask.loading
                    ? $i18n.get({
                        id: 'main.pages.App.Workflow.components.TaskTestPanel.index.testing',
                        dm: '测试中...',
                      })
                    : $i18n.get({
                        id: 'main.pages.App.Workflow.components.TaskTestPanel.index.startTesting',
                        dm: '开始测试',
                      })}
                </Button>
              </div>
            ),
          },
          {
            label: $i18n.get({
              id: 'main.pages.App.Workflow.components.TaskTestPanel.index.output',
              dm: '输出',
            }),
            key: 'result',
            children: (
              <div
                ref={resultContainerRef}
                className="h-full overflow-y-auto relative px-[20px] pb-[16px] flex flex-col gap-[12px]"
              >
                {!workflowTask.taskInfo ? (
                  <Welcome
                    data={{}}
                    title={$i18n.get({
                      id: 'main.pages.App.Workflow.components.TaskTestPanel.index.outputResultDisplayed',
                      dm: '输出结果在这里展示',
                    })}
                  />
                ) : (
                  <>
                    <div className="flex-justify-between">
                      <div className={styles['result-title']}>
                        {$i18n.get({
                          id: 'main.pages.App.Workflow.components.TaskTestPanel.index.runResult',
                          dm: '运行结果',
                        })}
                      </div>
                      <Flex align="center" gap={8}>
                        {workflowTask.loading && (
                          <Button onClick={workflowTask.stop}>
                            {$i18n.get({
                              id: 'main.pages.App.Workflow.components.TaskTestPanel.index.stop',
                              dm: '停止',
                            })}
                          </Button>
                        )}
                        {workflowTask.taskInfo && (
                          <ResultStatus
                            usages={usageList}
                            status={workflowTask.taskInfo.task_status}
                            execTime={workflowTask.taskInfo.task_exec_time}
                          />
                        )}
                      </Flex>
                    </div>
                    <Flex vertical gap={12}>
                      <NodeResultPanelList
                        data={workflowTask.taskInfo.node_results}
                      />
                      {workflowTask.taskInfo.task_results.map((item) => {
                        if (item.node_type === 'Input')
                          return (
                            <UserInputForm
                              key={`${item.node_id}_index`}
                              nodeData={item}
                              onSubmit={handleSubmitUserInput}
                            />
                          );

                        return (
                          <TextCard
                            key={`${item.node_id}_index`}
                            data={{ text: item.node_content as string }}
                          />
                        );
                      })}
                    </Flex>
                  </>
                )}
              </div>
            ),
          },
        ]}
      ></Tabs>
    </div>
  );
});
