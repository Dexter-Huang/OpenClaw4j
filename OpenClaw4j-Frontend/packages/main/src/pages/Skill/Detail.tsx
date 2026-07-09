import InnerLayout from '@/components/InnerLayout';
import $i18n from '@/i18n';
import { deleteSkill, getSkill, publishSkill } from '@/services/skill';
import { ISkill, SkillStatus } from '@/types/skill';
import {
  AlertDialog,
  Button,
  Dropdown,
  IconButton,
  message,
} from '@spark-ai/design';
import { Empty } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import styles from './Detail.module.less';

export default function SkillDetail() {
  const navigate = useNavigate();
  const { id: skillCode } = useParams<{ id: string }>();
  const [detail, setDetail] = useState<ISkill | null>(null);
  const [loading, setLoading] = useState(false);
  const [publishLoading, setPublishLoading] = useState(false);

  const getDetail = async () => {
    if (!skillCode) return;
    setLoading(true);
    try {
      const res = await getSkill({
        skill_code: skillCode,
      });
      setDetail(res.data);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    getDetail();
  }, [skillCode]);

  const handleDelete = () => {
    if (!detail) return;
    AlertDialog.warning({
      title: $i18n.get({
        id: 'main.pages.Skill.Detail.confirmDelete',
        dm: '确认删除此Skill吗',
      }),
      children: $i18n.get({
        id: 'main.pages.Skill.Detail.deleteWarning',
        dm: '删除后将不可恢复，已经添加该Skill的智能体可能会失效，请谨慎操作。',
      }),
      danger: true,
      onOk: () => {
        deleteSkill(detail.skill_code).then(() => {
          message.success(
            $i18n.get({
              id: 'main.pages.Skill.Detail.deleteSuccess',
              dm: '删除成功',
            }),
          );
          navigate('/skill');
        });
      },
    });
  };

  const handlePublish = () => {
    if (!detail || publishLoading) return;
    AlertDialog.warning({
      title: $i18n.get({
        id: 'main.pages.Skill.Detail.confirmPublish',
        dm: '确认发布此Skill吗',
      }),
      children: $i18n.get({
        id: 'main.pages.Skill.Detail.publishWarning',
        dm: '发布后该Skill可被添加到智能体中使用。',
      }),
      onOk: async () => {
        setPublishLoading(true);
        try {
          await publishSkill(detail.skill_code);
          message.success(
            $i18n.get({
              id: 'main.pages.Skill.Detail.publishSuccess',
              dm: '发布成功',
            }),
          );
          await getDetail();
        } finally {
          setPublishLoading(false);
        }
      },
    });
  };

  const renderOverviewItem = (label: string, value?: string) => (
    <>
      <div className={styles.label}>{label}</div>
      <div className={styles.value}>{value || '-'}</div>
    </>
  );

  const renderOverview = () => {
    if (!detail) return <Empty />;
    return (
      <div className={styles.content}>
        <div className={`${styles.panel} ${styles.overview}`}>
          {renderOverviewItem(
            $i18n.get({ id: 'main.pages.Skill.Detail.name', dm: '名称' }),
            detail.name,
          )}
          {renderOverviewItem(
            $i18n.get({
              id: 'main.pages.Skill.Detail.description',
              dm: '描述',
            }),
            detail.description,
          )}
          {renderOverviewItem(
            $i18n.get({ id: 'main.pages.Skill.Detail.version', dm: '版本' }),
            detail.current_version || detail.version,
          )}
          {renderOverviewItem(
            $i18n.get({
              id: 'main.pages.Skill.Detail.mainFilePath',
              dm: '主文件',
            }),
            detail.main_file_path,
          )}
          {renderOverviewItem(
            $i18n.get({
              id: 'main.pages.Skill.Detail.storageType',
              dm: '存储类型',
            }),
            detail.storage_type,
          )}
          {renderOverviewItem(
            $i18n.get({
              id: 'main.pages.Skill.Detail.storageBucket',
              dm: '存储Bucket',
            }),
            detail.storage_bucket,
          )}
          {renderOverviewItem(
            $i18n.get({
              id: 'main.pages.Skill.Detail.storagePrefix',
              dm: '文件包目录',
            }),
            detail.storage_prefix,
          )}
          {renderOverviewItem(
            $i18n.get({
              id: 'main.pages.Skill.Detail.packageObjectKey',
              dm: '压缩包路径',
            }),
            detail.package_object_key,
          )}
          {renderOverviewItem(
            $i18n.get({
              id: 'main.pages.Skill.Detail.fileCount',
              dm: '文件数量',
            }),
            detail.file_count === undefined ? undefined : String(detail.file_count),
          )}
          {renderOverviewItem(
            $i18n.get({
              id: 'main.pages.Skill.Detail.totalSizeBytes',
              dm: '总大小(bytes)',
            }),
            detail.total_size_bytes === undefined
              ? undefined
              : String(detail.total_size_bytes),
          )}
          {renderOverviewItem(
            $i18n.get({ id: 'main.pages.Skill.Detail.tags', dm: '标签' }),
            detail.tags,
          )}
          {renderOverviewItem(
            $i18n.get({
              id: 'main.pages.Skill.Detail.updatedAt',
              dm: '更新时间',
            }),
            detail.gmt_modified
              ? dayjs(detail.gmt_modified).format('YYYY-MM-DD HH:mm:ss')
              : '-',
          )}
          {renderOverviewItem(
            $i18n.get({
              id: 'main.pages.Skill.Detail.contentHash',
              dm: '内容Hash',
            }),
            detail.content_hash,
          )}
        </div>
      </div>
    );
  };

  const operations = () => (
    <>
      <Dropdown
        getPopupContainer={(ele) => ele}
        menu={{
          items: [
            {
              onClick: handleDelete,
              danger: true,
              label: $i18n.get({
                id: 'main.pages.Skill.Detail.delete',
                dm: '删除',
              }),
              key: 'delete',
            },
          ],
        }}
      >
        <IconButton icon="spark-more-line" bordered={false} />
      </Dropdown>
      <Button type="primary" onClick={() => navigate(`/skill/edit/${skillCode}`)}>
        {$i18n.get({ id: 'main.pages.Skill.Detail.edit', dm: '编辑' })}
      </Button>
      {detail && detail.status !== SkillStatus.PUBLISHED && (
        <Button loading={publishLoading} type="primary" onClick={handlePublish}>
          {$i18n.get({ id: 'main.pages.Skill.Detail.publish', dm: '发布' })}
        </Button>
      )}
    </>
  );

  return (
    <InnerLayout
      breadcrumbLinks={
        detail
          ? [
              {
                title: $i18n.get({
                  id: 'main.pages.Skill.Detail.skillManagement',
                  dm: 'Skill管理',
                }),
                path: '/skill',
              },
              {
                title:
                  detail.name ||
                  $i18n.get({
                    id: 'main.pages.Skill.Detail.skillDetail',
                    dm: 'Skill详情',
                  }),
              },
            ]
          : []
      }
      loading={loading}
      right={operations()}
      activeTab="overview"
      tabs={[
        {
          label: $i18n.get({ id: 'main.pages.Skill.Detail.overview', dm: '概览' }),
          key: 'overview',
          children: renderOverview(),
        },
      ]}
    />
  );
}
