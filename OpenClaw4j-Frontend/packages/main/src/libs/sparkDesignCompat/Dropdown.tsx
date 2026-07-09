import OriginalDropdown from '@spark-ai/design/dist/components/commonComponents/Dropdown';
import type { DropdownProps } from 'antd';
import { isValidElement } from 'react';

export type SparkDropdownProps = DropdownProps;

const SparkDropdown = ({ children, ...props }: SparkDropdownProps) => {
  // antd Dropdown 的 trigger 需要能直接拿到 DOM ref；@spark-ai/design 的
  // IconButton、antd Flex 等组件在 rc-trigger 下会回退到 findDOMNode。
  const triggerNode =
    isValidElement(children) && typeof children.type === 'string' ? (
      children
    ) : (
      <span style={{ display: 'inline-flex' }}>{children}</span>
    );

  return <OriginalDropdown {...props}>{triggerNode}</OriginalDropdown>;
};

export default SparkDropdown;
