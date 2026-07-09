import useIndexStyle from '@spark-ai/design/dist/components/commonComponents/Tooltip/index.style';
import { getCommonConfig } from '@spark-ai/design/dist/config';
import { findClosestBySelector } from '@spark-ai/design/dist/libs/dom';
import { Tooltip as AntTooltip, type TooltipProps } from 'antd';
import classNames from 'classnames';
import { isValidElement } from 'react';

export type SparkTooltipProps = TooltipProps & {
  mode?: 'dark' | 'light';
};

const SparkTooltip = (props: SparkTooltipProps) => {
  useIndexStyle();

  const {
    arrow,
    children,
    classNames: tooltipClassNames,
    getPopupContainer,
    mode = 'dark',
    overlayClassName,
    overlayInnerStyle,
    overlayStyle,
    styles,
    ...restProps
  } = props;
  const { antPrefix = 'ant', sparkPrefix = 'spark' } = getCommonConfig();
  // 兼容 @spark-ai/design 的 Button/IconButton 等非 DOM trigger，避免 antd/rc-trigger 回退到 findDOMNode。
  const triggerNode =
    isValidElement(children) && typeof children.type === 'string' ? (
      children
    ) : (
      <span style={{ display: 'inline-flex' }}>{children}</span>
    );

  return (
    <AntTooltip
      {...restProps}
      arrow={arrow ?? false}
      classNames={{
        ...tooltipClassNames,
        root: classNames(
          tooltipClassNames?.root,
          overlayClassName,
          mode === 'light' && `${sparkPrefix}-tooltip-light`,
        ),
      }}
      getPopupContainer={
        getPopupContainer ||
        ((triggerNode) =>
          findClosestBySelector(triggerNode, `.${antPrefix}-app`))
      }
      styles={{
        ...styles,
        body: {
          ...(styles?.body ?? {}),
          ...(overlayInnerStyle ?? {}),
        },
        root: {
          ...(styles?.root ?? {}),
          ...(overlayStyle ?? {}),
        },
      }}
    >
      {triggerNode}
    </AntTooltip>
  );
};

export default SparkTooltip;
