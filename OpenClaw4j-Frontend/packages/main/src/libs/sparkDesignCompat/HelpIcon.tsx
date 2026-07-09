import IconButton from '@spark-ai/design/dist/components/commonComponents/IconButton';
import useIconFontStyle from '@spark-ai/design/dist/components/commonComponents/IconFont/index.style';
import useIndexStyle from '@spark-ai/design/dist/components/commonComponents/HelpIcon/index.style';
import type { CSSProperties, ReactNode } from 'react';
import Tooltip, { type SparkTooltipProps } from './Tooltip';

export type SparkHelpIconProps = {
  className?: string;
  content: ReactNode;
  popoverProps?: SparkTooltipProps;
  style?: CSSProperties;
};

const SparkHelpIcon = (props: SparkHelpIconProps) => {
  const { className, content, popoverProps, style } = props;

  useIconFontStyle();
  useIndexStyle();

  return (
    <Tooltip
      title={content}
      overlayInnerStyle={{ maxWidth: 376 }}
      trigger="hover"
      style={style}
      {...popoverProps}
    >
      <IconButton
        className={className}
        icon="spark-info-line"
        shape="circle"
        bordered={false}
        size="small"
      />
    </Tooltip>
  );
};

export default SparkHelpIcon;
