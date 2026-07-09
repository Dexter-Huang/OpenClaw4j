import ProCard from '@/components/Card/ProCard';
import $i18n from '@/i18n';
import { ISkill, SkillStatus, SkillStatusMap } from '@/types/skill';
import { CodeOutlined } from '@ant-design/icons';
import { Button, Dropdown, IconFont } from '@spark-ai/design';
import type { MenuProps } from 'antd';
import classNames from 'classnames';
import dayjs from 'dayjs';
import React, { useMemo } from 'react';
import styles from './index.module.less';

interface SkillCardProps {
  data: ISkill;
  onClick?: (action?: string, data?: ISkill) => void;
  className?: string;
}

const SkillCard: React.FC<SkillCardProps> = ({ data, onClick, className }) => {
  const currentStatus = data.status as SkillStatus;
  const { color, text } =
    SkillStatusMap[currentStatus] || SkillStatusMap[SkillStatus.DELETED];

  const updateTime = useMemo(() => {
    return data.gmt_modified
      ? dayjs(data.gmt_modified).format('YYYY-MM-DD HH:mm:ss')
      : '-';
  }, [data.gmt_modified]);

  const handleDropdownClick: MenuProps['onClick'] = (info) => {
    info.domEvent.stopPropagation();
    onClick?.(info.key as string, data);
  };

  const handleButtonClick = (action: string, e: React.MouseEvent) => {
    e.stopPropagation();
    onClick?.(action, data);
  };

  return (
    <ProCard
      title={data.name}
      logo={<CodeOutlined />}
      statusNode={
        <div
          className={styles['status-tag']}
          style={{ color }}
          data-color={color}
        >
          <span className={styles.dot}></span>
          <span>{text}</span>
        </div>
      }
      info={[
        {
          label: $i18n.get({
            id: 'main.pages.Skill.components.SkillCard.description',
            dm: '描述',
          }),
          content: data.description || '-',
        },
        {
          label: $i18n.get({
            id: 'main.pages.Skill.components.SkillCard.version',
            dm: '版本',
          }),
          content: data.current_version || data.version || '-',
        },
        {
          label: $i18n.get({
            id: 'main.pages.Skill.components.SkillCard.id',
            dm: 'ID',
          }),
          content: data.skill_code,
        },
      ]}
      footerDescNode={
        <div className={styles['update-time']}>
          {$i18n.get({
            id: 'main.pages.Skill.components.SkillCard.updatedAt',
            dm: '更新于',
          })}
          {updateTime}
        </div>
      }
      footerOperateNode={
        <>
          <Button
            type="default"
            className="flex-1"
            onClick={(e) => handleButtonClick('edit', e)}
          >
            {$i18n.get({
              id: 'main.pages.Skill.components.SkillCard.edit',
              dm: '编辑',
            })}
          </Button>
          <Dropdown
            getPopupContainer={(ele) => ele}
            trigger={['click']}
            menu={{
              items: [
                {
                  key: 'delete',
                  label: $i18n.get({
                    id: 'main.pages.Skill.components.SkillCard.delete',
                    dm: '删除',
                  }),
                  danger: true,
                },
              ],
              onClick: handleDropdownClick,
            }}
          >
            <div onClick={(e) => e.stopPropagation()}>
              <Button icon={<IconFont type="spark-more-line" />} />
            </div>
          </Dropdown>
        </>
      }
      className={classNames(className)}
      onClick={() => onClick?.('detail', data)}
    />
  );
};

export default SkillCard;
