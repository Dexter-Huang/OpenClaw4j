import IconFont from '@spark-ai/design/dist/components/commonComponents/IconFont';
import Popover from '@spark-ai/design/dist/components/commonComponents/Popover';
import { Button as AntButton, ConfigProvider, type ButtonProps } from 'antd';
import React, { type ReactElement, type ReactNode } from 'react';

export interface SparkButtonProps extends Omit<ButtonProps, 'size' | 'type'> {
  size?: 'small' | 'middle';
  type?:
    | 'primary'
    | 'dashed'
    | 'link'
    | 'text'
    | 'default'
    | 'primaryLess'
    | 'textCompact';
  tooltipContent?: string | ReactNode;
  iconType?: string;
}

const SparkButton = (props: SparkButtonProps) => {
  const { icon, iconType, size, style, tooltipContent, type, ...buttonProps } =
    props;

  const buttonType = React.useMemo<ButtonProps['type']>(() => {
    if (type === 'primaryLess') return 'primary';
    if (type === 'textCompact') return 'link';
    return type;
  }, [type]);
  const mergedIcon = React.useMemo(() => {
    if (iconType) return <IconFont type={iconType} size={size} />;
    if (React.isValidElement(icon)) {
      return React.cloneElement(icon as ReactElement<{ size?: typeof size }>, {
        size,
      });
    }
    return icon ?? null;
  }, [icon, iconType, size]);

  const button = (
    <AntButton
      {...buttonProps}
      icon={mergedIcon}
      size={size}
      style={{
        lineHeight: 1,
        ...style,
      }}
      type={buttonType}
    />
  );

  if (type === 'primaryLess') {
    return (
      <ConfigProvider
        theme={{
          token: {
            colorPrimary: 'rgba(38, 36, 76, 0.88)',
            colorPrimaryHover: 'rgba(38, 36, 76, 0.65)',
          },
        }}
      >
        <Popover content={tooltipContent}>{button}</Popover>
      </ConfigProvider>
    );
  }

  if (type === 'textCompact') {
    return (
      <Popover content={tooltipContent}>
        <AntButton
          {...buttonProps}
          color="default"
          icon={mergedIcon}
          size={size}
          style={{
            paddingLeft: 0,
            paddingRight: 0,
            lineHeight: 1,
            ...style,
          }}
          variant="link"
        />
      </Popover>
    );
  }

  return <Popover content={tooltipContent}>{button}</Popover>;
};

export default SparkButton;
