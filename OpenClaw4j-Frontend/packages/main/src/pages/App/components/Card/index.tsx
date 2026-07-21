import ProCard from '@/components/Card/ProCard';
import $i18n from '@/i18n';
import { APP_ICON_IMAGE } from '@/pages/Component/AppComponent/components/AppSelector';
import { IAppType } from '@/services/appComponent';
import { IAppCard } from '@/types/appManage';
import { Button, Dropdown, IconButton, Tag } from '@spark-ai/design';
import dayjs from 'dayjs';
import React, { useMemo } from 'react';
import styles from './index.module.less';
import Status from './Status';

export interface AppCardProps extends IAppCard {
  onClickAction: (key: string) => void;
}

const typeLabelMap: Record<IAppType, string> = {
  [IAppType.AGENT]: $i18n.get({
    id: 'main.pages.App.components.Card.index.intelligentAgentApp',
    dm: '智能体应用',
  }),
  [IAppType.WORKFLOW]: $i18n.get({
    id: 'main.pages.App.components.Card.index.workflowApp',
    dm: '流程编排应用',
  }),
};

const AppCard: React.FC<AppCardProps> = ({
  name,
  gmt_modified,
  type,
  status,
  onClickAction,
}) => {
  const updateTime = useMemo(() => {
    return dayjs(gmt_modified).format('YYYY-MM-DD HH:mm:ss');
  }, [gmt_modified]);

  const language = $i18n.getCurrentLanguage();

  return (
    <ProCard
      title={name}
      logo={<img className={styles['logo']} src={APP_ICON_IMAGE[type]} />}
      statusNode={<Status status={status} />}
      stackStatus
      labelWidth={language === 'en' ? 70 : 60}
      info={[
        {
          label: $i18n.get({
            id: 'main.pages.App.components.Card.index.updateTime',
            dm: '更新时间',
          }),
          content: updateTime,
        },
      ]}
      onClick={() => onClickAction('click')}
      footerDescNode={<Tag color="mauve">{typeLabelMap[type]}</Tag>}
      footerOperateNode={
        <>
          <Button
            type="primary"
            className="flex-1"
            onClick={(event) => {
              event.stopPropagation();
              onClickAction('edit');
            }}
          >
            {$i18n.get({
              id: 'main.pages.App.components.Card.index.edit',
              dm: '编辑',
            })}
          </Button>
          <Dropdown
            trigger={['click']}
            menu={{
              onClick: (e) => {
                e.domEvent.stopPropagation();
                onClickAction(e.key);
              },
              items: [
                {
                  label: $i18n.get({
                    id: 'main.pages.App.components.Card.index.modifyAppName',
                    dm: '修改应用名',
                  }),
                  key: 'editName',
                },
                {
                  label: $i18n.get({
                    id: 'main.pages.App.components.Card.index.copyApp',
                    dm: '复制应用',
                  }),
                  key: 'copy',
                },
                {
                  label: $i18n.get({
                    id: 'main.pages.App.components.Card.index.delete',
                    dm: '删除',
                  }),
                  danger: true,
                  key: 'delete',
                },
              ],
            }}
          >
            <div onClick={(event) => event.stopPropagation()}>
              <IconButton shape="default" icon="spark-more-line" />
            </div>
          </Dropdown>
        </>
      }
    />
  );
};

export default AppCard;
