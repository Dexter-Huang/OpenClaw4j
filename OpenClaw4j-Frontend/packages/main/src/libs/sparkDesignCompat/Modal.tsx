import useIndexStyle from '@spark-ai/design/dist/components/commonComponents/Modal/index.style';
import { getCommonConfig } from '@spark-ai/design/dist/config';
import { Modal as AntModal, type ModalProps } from 'antd';
import classNames from 'classnames';
import type { CSSProperties, ReactNode } from 'react';

export interface SparkModalProps extends ModalProps {
  info?: string;
  showDivider?: boolean;
  wrapClassName?: string;
  wrapStyle?: CSSProperties;
}

const SparkModal = (props: SparkModalProps) => {
  useIndexStyle();

  const {
    classNames: modalClassNames,
    footer,
    info,
    showDivider = true,
    styles,
    wrapClassName,
    wrapStyle,
    ...restProps
  } = props;
  const { sparkPrefix = 'spark', variables } = getCommonConfig();

  const renderFooter = (originNode: ReactNode) =>
    info ? (
      <div className={`${sparkPrefix}-modal-footer-wrapper`}>
        <span className={`${sparkPrefix}-modal-footer-info`}>{info}</span>
        <div className={`${sparkPrefix}-modal-footer-origin-node`}>
          {originNode}
        </div>
      </div>
    ) : (
      originNode
    );
  const mergedFooter: ModalProps['footer'] =
    footer === undefined ? renderFooter : footer;

  return (
    <AntModal
      {...restProps}
      classNames={{
        ...modalClassNames,
        wrapper: classNames(
          `${sparkPrefix}-modal`,
          { [`${sparkPrefix}-show-divider`]: showDivider },
          modalClassNames?.wrapper,
          wrapClassName,
        ),
      }}
      footer={mergedFooter}
      styles={{
        ...styles,
        wrapper: {
          ...(variables ?? {}),
          ...(styles?.wrapper ?? {}),
          ...(wrapStyle ?? {}),
        },
      }}
      transitionName=""
    />
  );
};

SparkModal.useModal = AntModal.useModal;
SparkModal.success = AntModal.success;
SparkModal.error = AntModal.error;
SparkModal.warning = AntModal.warning;
SparkModal.info = AntModal.info;
SparkModal.confirm = AntModal.confirm;

export default SparkModal;
