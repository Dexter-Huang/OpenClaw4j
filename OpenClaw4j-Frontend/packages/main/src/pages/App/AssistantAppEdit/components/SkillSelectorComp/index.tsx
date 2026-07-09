import defaultSettings from '@/defaultSettings';
import $i18n from '@/i18n';
import { HelpIcon } from '@/libs/sparkDesignCompat';
import { SkillSelectDrawer } from '@/pages/App/components/SkillSelector';
import { ISkill } from '@/types/skill';
import { Button, IconFont } from '@spark-ai/design';
import { useSetState } from 'ahooks';
import { Divider, Flex } from 'antd';
import cls from 'classnames';
import { useContext, useEffect } from 'react';
import { AssistantAppContext } from '../../AssistantAppContext';
import SelectedConfigItem from '../SelectedConfigItem';
import styles from './index.module.less';

export const SKILL_MAX_LIMIT = defaultSettings.agentSkillMaxLimit;

export default function SkillSelectorComp() {
  const { appState, onAppConfigChange } = useContext(AssistantAppContext);
  const { skills = [] as ISkill[] } = appState.appBasicConfig?.config || {};
  const [state, setState] = useSetState({
    expand: false,
    selectVisible: false,
  });

  const onSelectSkills = (val: ISkill[]) => {
    onAppConfigChange({ skills: val });
  };

  useEffect(() => {
    if (skills.length) {
      setState({ expand: true });
    }
  }, [skills]);

  const onRemoveSkill = (val: string) => {
    onSelectSkills(skills.filter((item) => item.skill_code !== val));
  };

  return (
    <Flex vertical className="assistantSkillSelectorSection">
      <Flex justify="space-between" align="center">
        <Flex
          gap={8}
          className="text-[13px] font-medium leading-[20px]"
          style={{ color: 'var(--ag-ant-color-text)' }}
          align="center"
        >
          <Flex align="center">
            <span>
              {$i18n.get({
                id: 'main.pages.App.AssistantAppEdit.components.SkillSelectorComp.index.skill',
                dm: 'Skill',
              })}
            </span>
            <HelpIcon
              content={$i18n.get({
                id: 'main.pages.App.AssistantAppEdit.components.SkillSelectorComp.index.skillHelp',
                dm: 'Skill用于给智能体挂载可复用的说明文件、脚本文件和特定任务执行规范。',
              })}
            />
          </Flex>
          <span
            className="text-[12px] leading-[20px]"
            style={{ color: 'var(--ag-ant-color-text-tertiary)' }}
          >
            {skills.length}/{SKILL_MAX_LIMIT}
          </span>
        </Flex>
        <span>
          <Button
            style={{ padding: 0 }}
            onClick={() => setState({ selectVisible: true })}
            iconType="spark-plus-line"
            type="text"
            size="small"
          >
            Skill
          </Button>
          <Divider type="vertical" className="ml-[16px] mr-[16px]" />
          <IconFont
            onClick={() => setState({ expand: !state.expand })}
            className={cls(styles.expandBtn, !state.expand && styles.hidden)}
            type="spark-up-line"
            isCursorPointer
          />
        </span>
      </Flex>
      {state.expand && (
        <Flex vertical gap={8}>
          {skills.map(
            (item) =>
              item && (
                <SelectedConfigItem
                  iconType="spark-fileCode-line"
                  name={item.name}
                  description={item.description}
                  rightArea={
                    <Flex gap={12}>
                      <div style={{ color: 'var(--ag-ant-color-text-tertiary)' }}>
                        {item.current_version || item.version}
                      </div>
                      <IconFont
                        type="spark-delete-line"
                        isCursorPointer
                        onClick={() => {
                          onRemoveSkill(item.skill_code);
                        }}
                      />
                    </Flex>
                  }
                  key={item.skill_code}
                />
              ),
          )}
        </Flex>
      )}
      {state.selectVisible && (
        <SkillSelectDrawer
          selectedSkills={skills}
          onOk={onSelectSkills}
          onClose={() => {
            setState({ selectVisible: false });
          }}
        />
      )}
    </Flex>
  );
}
