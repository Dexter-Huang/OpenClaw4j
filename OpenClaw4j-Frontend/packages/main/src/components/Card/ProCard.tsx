import { Card } from '@spark-ai/design';
import React from 'react';
import styles from './index.module.less';

export interface ProCardInfo {
  label?: React.ReactNode;
  content: React.ReactNode;
}

export interface ProCardProps {
  title: React.ReactNode;
  logo?: React.ReactNode;
  info?: ProCardInfo[];
  labelWidth?: number;
  onClick?: () => void;
  className?: string;
  statusNode?: React.ReactNode;
  stackStatus?: boolean;
  footerDescNode?: React.ReactNode;
  footerOperateNode?: React.ReactNode;
}

const ProCard: React.FC<ProCardProps> = ({
  title,
  logo,
  info = [],
  labelWidth,
  onClick,
  className,
  statusNode,
  stackStatus = false,
  footerDescNode,
  footerOperateNode,
}) => {
  return (
    <Card
      className={`${styles.proCard} ${className || ''}`}
      onClick={onClick}
      hoverable={!!onClick}
    >
      <div
        className={
          stackStatus ? styles.cardHeaderStacked : styles.cardHeader
        }
      >
        <div
          className={
            stackStatus ? styles.headerLeftStacked : styles.headerLeft
          }
        >
          {logo && <div className={styles.logo}>{logo}</div>}
          <div
            className={
              stackStatus ? styles.titleWrapperStacked : styles.titleWrapper
            }
          >
            <h3 className={styles.title}>{title}</h3>
            {statusNode && <div className={styles.status}>{statusNode}</div>}
          </div>
        </div>
      </div>

      {info.length > 0 && (
        <div className={styles.cardBody}>
          {info.map((item, index) => (
            <div key={index} className={styles.infoItem}>
              {item.label && (
                <span
                  className={styles.label}
                  style={labelWidth ? { width: labelWidth } : undefined}
                >
                  {item.label}
                </span>
              )}
              <span className={styles.content}>{item.content}</span>
            </div>
          ))}
        </div>
      )}

      {(footerDescNode || footerOperateNode) && (
        <div className={styles.cardFooter}>
          {footerDescNode && (
            <div className={styles.footerDesc}>{footerDescNode}</div>
          )}
          {footerOperateNode && (
            <div className={styles.footerOperate}>{footerOperateNode}</div>
          )}
        </div>
      )}
    </Card>
  );
};

export default ProCard;
