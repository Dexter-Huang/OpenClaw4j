import useIndexStyle from '@spark-ai/design/dist/components/commonComponents/Tooltip/index.style';
import { getCommonConfig } from '@spark-ai/design/dist/config';
import { findClosestBySelector } from '@spark-ai/design/dist/libs/dom';
import { Tooltip as AntTooltip, type TooltipProps } from 'antd';
import classNames from 'classnames';

export type SparkTooltipProps = TooltipProps & {
  mode?: 'dark' | 'light';
};

const SparkTooltip = (props: SparkTooltipProps) => {
  useIndexStyle();

  const {
    arrow,
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
    />
  );
};

export default SparkTooltip;
