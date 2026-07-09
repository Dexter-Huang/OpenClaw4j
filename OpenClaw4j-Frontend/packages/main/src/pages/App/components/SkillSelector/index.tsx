import defaultSettings from '@/defaultSettings';
import $i18n from '@/i18n';
import { CreateSkillBtn } from '@/pages/Skill/Manage';
import { listSkills } from '@/services/skill';
import { IListSkillsParams, ISkill, SkillStatus } from '@/types/skill';
import {
  Button,
  Drawer,
  Empty,
  IconFont,
  Input,
  message,
  Pagination,
} from '@spark-ai/design';
import { renderTooltip } from '@/libs/sparkDesignCompat';
import { useSetState } from 'ahooks';
import { Checkbox, Flex, Spin, Typography } from 'antd';
import classNames from 'classnames';
import { debounce } from 'lodash-es';
import { useEffect, useState } from 'react';
import styles from './index.module.less';

const SKILL_MAX_LIMIT = defaultSettings.agentSkillMaxLimit;

interface ISkillListItemProps {
  item: ISkill;
  selectedSkills?: ISkill[];
  onSelectSkill?: (val: ISkill) => void;
  onRemoveSkill?: (val: string) => void;
}

function SkillListItem(props: ISkillListItemProps) {
  const { item, selectedSkills = [] } = props;
  const selected = !!selectedSkills.find(
    (skill) => skill.skill_code === item.skill_code,
  );

  return (
    <div
      className={classNames(styles.skillWrapper, {
        [styles.active]: selected,
      })}
    >
      <Flex gap={8}>
        <Checkbox
          checked={selected}
          onChange={(e) => {
            if (e.target.checked) {
              props.onSelectSkill?.(item);
            } else {
              props.onRemoveSkill?.(item.skill_code);
            }
          }}
          disabled={item.status !== SkillStatus.PUBLISHED}
        />
        <Flex gap={8} className="w-full h-[52px] flex-1" align="center">
          <Flex align="center" className="h-[40px] w-[40px]">
            <Flex
              align="center"
              justify="center"
              className={styles.iconBox}
            >
              <IconFont
                className="h-full w-full rounded-[6px]"
                type="spark-fileCode-line"
              />
            </Flex>
          </Flex>
          <div style={{ width: 'calc(100% - 48px)' }}>
            <Flex
              justify="space-between"
              className="header leading-[22px] h-[22px]"
            >
              <Typography.Text
                ellipsis={{ tooltip: renderTooltip(item.name) }}
                className="text-[16px] font-semibold mr-[4px]"
                style={{
                  color: 'var(--ag-ant-color-text-base)',
                  width: 0,
                  flex: 1,
                }}
              >
                {item.name}
              </Typography.Text>
              <Typography.Text
                style={{
                  color: 'var(--ag-ant-color-text-tertiary)',
                  fontSize: 12,
                  flexShrink: 0,
                }}
              >
                {item.current_version || item.version}
              </Typography.Text>
            </Flex>
            <Typography.Paragraph
              className={styles.desc}
              style={{ marginBottom: 0 }}
              ellipsis={{ rows: 1, tooltip: renderTooltip(item.description) }}
            >
              {item.description}
            </Typography.Paragraph>
          </div>
        </Flex>
      </Flex>
    </div>
  );
}

interface ISkillSelectorProps {
  onSkillsChange?: (val: ISkill[]) => void;
  selectedSkills?: ISkill[];
}

function SkillSelector(props: ISkillSelectorProps) {
  const [filterParams, setFilterParams] = useSetState<IListSkillsParams>({
    need_files: false,
    current: 1,
    size: 10,
    name: '',
  });
  const [total, setTotal] = useState(0);
  const [list, setList] = useState<ISkill[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchList = () => {
    setLoading(true);
    listSkills(filterParams)
      .then((res) => {
        setList(res.data.records);
        setTotal(res.data.total);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  useEffect(() => {
    fetchList();
  }, [filterParams]);

  const onInputChange = debounce((e) => {
    setFilterParams({
      current: 1,
      name: e.target.value,
    });
  }, 500);

  return (
    <>
      <Flex justify="space-between" className="mb-[16px]">
        <Input
          onChange={onInputChange}
          prefix={<IconFont type="spark-search-line" />}
          placeholder={$i18n.get({
            id: 'main.pages.App.components.SkillSelector.index.inputHere',
            dm: '在此输入',
          })}
          allowClear
          style={{ width: 220 }}
        />
        <CreateSkillBtn isOpenNew buttonProps={{ type: 'default' }} />
      </Flex>
      {loading ? (
        <Spin className="w-full h-full" />
      ) : (
        <Flex vertical gap={16}>
          {list.length ? (
            <>
              {list.map((item) => (
                <SkillListItem
                  item={item}
                  selectedSkills={props.selectedSkills}
                  key={item.skill_code}
                  onSelectSkill={(skill) => {
                    if (
                      props.selectedSkills &&
                      props.selectedSkills.length >= SKILL_MAX_LIMIT
                    ) {
                      message.warning(
                        $i18n.get({
                          id: 'main.pages.App.components.SkillSelector.index.reachedMaxLimit',
                          dm: '已达到最大数量限制',
                        }),
                      );
                      return;
                    }
                    props.onSkillsChange?.([
                      ...(props.selectedSkills || []),
                      skill,
                    ]);
                  }}
                  onRemoveSkill={(skillCode) => {
                    props.onSkillsChange?.(
                      (props.selectedSkills || []).filter(
                        (skill) => skill.skill_code !== skillCode,
                      ),
                    );
                  }}
                />
              ))}
              <Pagination
                pageSize={filterParams.size}
                current={filterParams.current}
                total={total}
                hideOnSinglePage
                hideTips
                onChange={(page, pageSize) => {
                  setFilterParams({ current: page, size: pageSize });
                }}
                pageSizeOptions={[10, 20]}
              />
            </>
          ) : (
            <Flex className="h-full" align="center" justify="center">
              <Empty
                title={
                  filterParams.name?.length
                    ? $i18n.get({
                        id: 'main.pages.App.components.SkillSelector.index.noSearchResult',
                        dm: '暂无搜索结果',
                      })
                    : $i18n.get({
                        id: 'main.pages.App.components.SkillSelector.index.noCustomSkill',
                        dm: '暂无自定义Skill',
                      })
                }
                description={
                  !filterParams.name?.length && (
                    <CreateSkillBtn
                      isOpenNew
                      text={$i18n.get({
                        id: 'main.pages.App.components.SkillSelector.index.goCreate',
                        dm: '去创建',
                      })}
                    />
                  )
                }
              />
            </Flex>
          )}
        </Flex>
      )}
    </>
  );
}

export interface ISkillSelectorDrawerProps {
  onOk: (val: ISkill[]) => void;
  onClose: () => void;
  selectedSkills?: ISkill[];
}

export const SkillSelectDrawer = (props: ISkillSelectorDrawerProps) => {
  const [cacheSelected, setCacheSelected] = useState<ISkill[]>([
    ...(props.selectedSkills || []),
  ]);

  return (
    <Drawer
      title={$i18n.get({
        id: 'main.pages.App.components.SkillSelector.index.selectSkill',
        dm: '选择Skill',
      })}
      open
      width={640}
      onClose={props.onClose}
      footer={
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
          className="w-full"
        >
          <div
            style={{
              color: 'var(--ag-ant-color-text-tertiary)',
              fontSize: '14px',
              fontWeight: 'normal',
              lineHeight: '24px',
            }}
          >
            {!!cacheSelected.length &&
              $i18n.get(
                {
                  id: 'main.pages.App.components.SkillSelector.index.addedSkill',
                  dm: '已添加Skill{var1}/{var2}',
                },
                { var1: cacheSelected.length, var2: SKILL_MAX_LIMIT },
              )}
          </div>
          <div style={{ display: 'flex', gap: 12 }}>
            <Button type="default" onClick={props.onClose}>
              {$i18n.get({
                id: 'main.pages.App.components.SkillSelector.index.cancel',
                dm: '取消',
              })}
            </Button>
            <Button
              type="primary"
              onClick={() => {
                props.onOk(cacheSelected);
                props.onClose();
                message.success(
                  $i18n.get({
                    id: 'main.pages.App.components.SkillSelector.index.addSuccess',
                    dm: '添加成功！',
                  }),
                );
              }}
            >
              {$i18n.get({
                id: 'main.pages.Setting.ModelService.components.ModelServiceProviderModal.index.confirm',
                dm: '确认',
              })}
            </Button>
          </div>
        </div>
      }
    >
      <SkillSelector
        selectedSkills={cacheSelected}
        onSkillsChange={setCacheSelected}
      />
    </Drawer>
  );
};
