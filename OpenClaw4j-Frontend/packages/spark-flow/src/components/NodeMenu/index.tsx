import { Empty, Input, Popover } from '@spark-ai/design';
import { useStore } from '@spark-flow/flow/context';
import { useNodesReadOnly } from '@spark-flow/hooks';
import { useNodesInteraction } from '@spark-flow/hooks/useNodesInteraction';
import $i18n from '@spark-flow/i18n';
import { INodeSchema, IPointItem } from '@spark-flow/types/work-flow';
import { useNodes } from '@xyflow/react';
import { message } from 'antd';
import { TooltipPlacement } from 'antd/es/tooltip';
import classNames from 'classnames';
import { debounce, groupBy } from 'lodash-es';
import React, { memo, useCallback, useEffect, useMemo, useState } from 'react';
import CustomIcon from '../CustomIcon';
import FlowIcon from '../FlowIcon';
import './index.less';
import { getNodeMenuItemInteraction } from './nodeMenuInteraction';

interface INodeMenuItemProps {
  data: {
    title: string;
    type: string;
    iconType: string;
    desc: string;
  };
  onDragStart: (
    event: React.DragEvent<HTMLDivElement>,
    nodeType: string,
  ) => void;
  draggable?: boolean;
  disabled?: boolean;
  onClick: () => void;
}

export const NodeMenuItem = memo((props: INodeMenuItemProps) => {
  const { data, onDragStart } = props;
  const [open, setOpen] = useState(false);
  const _dragStart = useCallback(
    (event: React.DragEvent<HTMLDivElement>) => {
      if (!props.draggable) {
        event.preventDefault();
        return;
      }
      onDragStart(event, data.type);
      setOpen(false);
    },
    [props.draggable, onDragStart, data.type],
  );
  return (
    <Popover
      key={data.type}
      placement="left"
      arrow={false}
      onOpenChange={setOpen}
      open={open}
      destroyTooltipOnHide
      getPopupContainer={() =>
        document.querySelector('.spark-flow-node-menu') || document.body
      }
      content={
        <div className="spark-flow-node-menu-item-info">
          <div
            className={
              'cursor-pointer spark-flow-node-menu-item-info-title flex items-center gap-[8px]'
            }
          >
            <FlowIcon nodeType={data.type} />
            <span>{data.title}</span>
          </div>
          <div className="spark-flow-node-menu-item-info-desc">{data.desc}</div>
        </div>
      }
    >
      <div
        draggable={props.draggable}
        onDragStart={_dragStart}
        onClick={props.onClick}
        className={classNames(
          'spark-flow-node-menu-item flex items-center gap-[8px] h-9',
          {
            'cursor-pointer': !props.disabled,
            'spark-flow-node-menu-item-disabled': props.disabled,
          },
        )}
      >
        <FlowIcon nodeType={data.type} />
        <span>{data.title}</span>
      </div>
    </Popover>
  );
});

interface IProps {
  source?: IPointItem;
  target?: IPointItem;
  onSelect?: () => void;
  disableDrag?: boolean;
  parentId?: string;
}

const generateMenuList = ({
  nodeSchemaMap,
  searchValue,
  hasParent,
}: {
  nodeSchemaMap: Record<string, INodeSchema>;
  searchValue: string;
  hasParent: boolean;
}) => {
  const list: Array<{
    title: string;
    type: string;
    iconType: string;
    desc: string;
    group?: string;
  }> = [];

  Object.keys(nodeSchemaMap).forEach((nodeTypeKey) => {
    if (nodeSchemaMap[nodeTypeKey].hideInMenu) return;
    if (
      hasParent &&
      (nodeSchemaMap[nodeTypeKey].isGroup ||
        nodeSchemaMap[nodeTypeKey].disableInGroup)
    )
      return;
    const nodeSchemaItem = nodeSchemaMap[nodeTypeKey];

    if (
      searchValue &&
      !nodeSchemaItem.title.toLowerCase().includes(searchValue.toLowerCase())
    ) {
      return;
    }

    list.push({
      title: nodeSchemaItem.title,
      type: nodeSchemaItem.type,
      desc: nodeSchemaItem.desc,
      iconType: nodeSchemaItem.iconType,
      group: nodeSchemaItem.groupLabel,
    });
  });

  return list;
};

export const NodeMenu = memo((props: IProps) => {
  const nodeSchemaMap = useStore((store) => store.nodeSchemaMap);
  const nodes = useNodes();
  const { onAddNewNodeWithSource } = useNodesInteraction();
  const [searchValue, setSearchValue] = useState('');
  const setIsDragging = useStore((state) => state.setIsDragging);
  const { nodesReadOnly } = useNodesReadOnly();
  const [menuList, setMenuList] = useState<
    Array<{
      title: string;
      type: string;
      iconType: string;
      desc: string;
      group?: string;
    }>
  >(
    generateMenuList({
      nodeSchemaMap,
      searchValue,
      hasParent: !!props.parentId,
    }),
  );

  const debouncedSetMenuList = useCallback(
    debounce(
      (params: {
        nodeSchemaMap: Record<string, INodeSchema>;
        searchValue: string;
        hasParent: boolean;
      }) => {
        setMenuList(generateMenuList(params));
      },
      300,
    ),
    [],
  );

  useEffect(() => {
    debouncedSetMenuList({
      nodeSchemaMap,
      searchValue,
      hasParent: !!props.parentId,
    });
    return () => {
      debouncedSetMenuList.cancel();
    };
  }, [nodeSchemaMap, searchValue, props.parentId, debouncedSetMenuList]);

  const groupList = useMemo(() => {
    if (!menuList) return [];

    const groups = groupBy(
      menuList,
      (item) =>
        item.group ||
        $i18n.get({
          id: 'spark-flow.components.NodeMenu.index.ungrouped',
          dm: '未分组',
        }),
    );
    return Object.entries(groups).map(([group, items]) => ({ group, items }));
  }, [menuList]);

  const menuItemInteraction = useMemo(
    () =>
      getNodeMenuItemInteraction({
        disableDrag: props.disableDrag,
        nodesReadOnly,
      }),
    [props.disableDrag, nodesReadOnly],
  );

  const onDragStart = useCallback(
    (event: React.DragEvent<HTMLDivElement>, nodeType: string) => {
      event.dataTransfer.setData('application/reactflow', nodeType);
      event.dataTransfer.effectAllowed = 'move';
      setIsDragging(true);
    },
    [setIsDragging],
  );

  const onMenuItemClick = useCallback(
    (data: INodeMenuItemProps['data']) => {
      if (menuItemInteraction.shouldWarnReadonly) {
        message.warning(
          $i18n.get({
            id: 'spark-flow.components.NodeMenu.index.readonlyReturnToDraft',
            dm: '当前为历史版本，请回到当前版本后编辑',
          }),
        );
        return;
      }
      if (!props.source) return;

      const parentId =
        props.parentId ||
        nodes.find((item) => item.id === props.source?.id)?.parentId;

      onAddNewNodeWithSource(
        { type: data.type, parentId },
        props.source,
        props.target,
      );
      props.onSelect?.();
    },
    [
      menuItemInteraction.shouldWarnReadonly,
      props.parentId,
      props.source,
      props.target,
      props.onSelect,
      nodes,
      onAddNewNodeWithSource,
    ],
  );

  return (
    <div className="spark-flow-node-menu h-full flex flex-col">
      <div className="spark-flow-node-menu-search">
        <Input
          prefix={<CustomIcon type="spark-search-line" />}
          placeholder={$i18n.get({
            id: 'spark-flow.components.NodeMenu.index.searchNodeName',
            dm: '搜索节点名称',
          })}
          value={searchValue}
          onChange={(e) => setSearchValue(e.target.value)}
        />
      </div>
      <div className="flex-1 min-h-0 overflow-y-auto nowheel flex flex-col gap-[16px] px-[16px] spark-flow-node-menu-list pb-[16px]">
        {!groupList.length ? (
          <div className="full-center">
            <Empty
              size={204}
              description={$i18n.get({
                id: 'main.pages.App.components.MCPSelector.index.noSearchResult',
                dm: '暂无搜索结果',
              })}
            />
          </div>
        ) : (
          groupList.map((groupItem, index) => (
            <div key={index}>
              <div className="spark-flow-node-menu-group-title">
                {groupItem.group}
              </div>
              {groupItem.items.map((item) => (
                <NodeMenuItem
                  onClick={() => onMenuItemClick(item)}
                  draggable={menuItemInteraction.draggable}
                  disabled={menuItemInteraction.disabled}
                  onDragStart={onDragStart}
                  data={item}
                  key={item.type}
                />
              ))}
            </div>
          ))
        )}
      </div>
    </div>
  );
});

interface IPopoverNodeMenuProps extends IProps {
  children: React.ReactNode;
  onOpenChange?: (open: boolean) => void;
  placement?: TooltipPlacement;
  disableDrag?: boolean;
}

export const PopoverNodeMenu = memo((props: IPopoverNodeMenuProps) => {
  const [open, setOpen] = useState(false);
  const { nodesReadOnly } = useNodesReadOnly();
  const closeOpen = useCallback(() => {
    setOpen(false);
  }, []);

  useEffect(() => {
    props.onOpenChange?.(open);
  }, [open]);

  return (
    <Popover
      placement={props.placement || 'right'}
      trigger="click"
      onOpenChange={setOpen}
      open={open && !nodesReadOnly}
      destroyTooltipOnHide
      getPopupContainer={(ele) => ele}
      content={
        <div
          onClick={(e) => e.stopPropagation()}
          className={'spark-flow-popover-node-menu'}
        >
          <NodeMenu
            disableDrag={props.disableDrag}
            onSelect={closeOpen}
            source={props.source}
            target={props.target}
            parentId={props.parentId}
          />
        </div>
      }
    >
      {props.children}
    </Popover>
  );
});

export default NodeMenu;
