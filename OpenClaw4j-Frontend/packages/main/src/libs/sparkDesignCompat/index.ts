import type { CSSProperties, ReactNode } from 'react';

export * from '@spark-ai/design/dist/index';
export { copy, isElement } from '@spark-ai/design/dist/libs/utils';
export { default as Button } from './Button';
export type { SparkButtonProps as ButtonProps } from './Button';
export { default as Modal } from './Modal';
export type { SparkModalProps as ModalProps } from './Modal';
export { default as Tooltip } from './Tooltip';
export type { SparkTooltipProps as TooltipProps } from './Tooltip';

type TooltipExtraProps = {
  maxHeight?: CSSProperties['maxHeight'];
  maxWidth?: CSSProperties['maxWidth'];
  overlayInnerStyle?: CSSProperties;
  styles?: {
    body?: CSSProperties;
  };
} & Record<string, unknown>;

export const renderTooltip = (
  title: ReactNode,
  extraProps: TooltipExtraProps = {},
) => {
  const { maxHeight, maxWidth, overlayInnerStyle, styles, ...restProps } =
    extraProps;

  return {
    arrow: false,
    title,
    ...restProps,
    styles: {
      ...styles,
      body: {
        maxWidth: maxWidth || 326,
        maxHeight: maxHeight || 150,
        overflowY: 'auto',
        padding: '6px 12px',
        ...(styles?.body ?? {}),
        ...(overlayInnerStyle ?? {}),
      },
    },
  };
};
