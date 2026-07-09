import InnerLayout from '@/components/InnerLayout';
import $i18n from '@/i18n';
import {
  createSkill,
  getSkill,
  updateSkill,
  uploadSkillPackage,
} from '@/services/skill';
import {
  ICreateSkillParams,
  ISkill,
  IUpdateSkillParams,
  SkillStatus,
} from '@/types/skill';
import {
  AlertDialog,
  Button,
  Form,
  IconFont,
  Input,
  message,
} from '@spark-ai/design';
import { useMount } from 'ahooks';
import { Flex, Upload } from 'antd';
import type { UploadFile } from 'antd';
import { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import styles from './Create.module.less';

const DEFAULT_MAIN_FILE = 'SKILL.md';

export default function SkillCreate() {
  const navigate = useNavigate();
  const { id: skillCode } = useParams<{ id: string }>();
  const [form] = Form.useForm();

  const [loading, setLoading] = useState(!!skillCode);
  const [saveLoading, setSaveLoading] = useState(false);
  const [packageUploading, setPackageUploading] = useState(false);
  const [packageFileList, setPackageFileList] = useState<UploadFile[]>([]);
  const [initialData, setInitialData] = useState<ISkill | null>(null);

  useMount(() => {
    if (!skillCode) {
      form.setFieldsValue({
        mainFilePath: DEFAULT_MAIN_FILE,
        manifest: '{}',
      });
      return;
    }

    getSkill({
      skill_code: skillCode,
    })
      .then((res) => {
        const data = res.data;
        const mainFilePath = data.main_file_path || DEFAULT_MAIN_FILE;
        setInitialData(data);
        form.setFieldsValue({
          name: data.name,
          description: data.description,
          tags: data.tags,
          mainFilePath,
          storageType: data.storage_type,
          storageBucket: data.storage_bucket,
          storagePrefix: data.storage_prefix,
          packageObjectKey: data.package_object_key,
          contentHash: data.content_hash,
          fileCount: data.file_count,
          totalSizeBytes: data.total_size_bytes,
          manifest: data.manifest || '{}',
        });
        if (data.package_object_key) {
          setPackageFileList([
            {
              uid: data.package_object_key,
              name: data.package_object_key.split('/').pop() || 'package.zip',
              status: 'done',
            },
          ]);
        }
      })
      .finally(() => {
        setLoading(false);
      });
  });

  const isFormChanged = () => {
    const currentValues = form.getFieldsValue();
    if (!skillCode) {
      return Object.values(currentValues).some(Boolean);
    }
    if (!initialData) return false;

    const mainFilePath = initialData.main_file_path || DEFAULT_MAIN_FILE;
    return (
      currentValues.name !== initialData.name ||
      currentValues.description !== initialData.description ||
      currentValues.tags !== initialData.tags ||
      currentValues.mainFilePath !== mainFilePath ||
      currentValues.storageType !== initialData.storage_type ||
      currentValues.storageBucket !== initialData.storage_bucket ||
      currentValues.storagePrefix !== initialData.storage_prefix ||
      currentValues.packageObjectKey !== initialData.package_object_key ||
      currentValues.contentHash !== initialData.content_hash ||
      currentValues.fileCount !== initialData.file_count ||
      currentValues.totalSizeBytes !== initialData.total_size_bytes ||
      currentValues.manifest !== (initialData.manifest || '{}')
    );
  };

  const onBack = () => {
    if (isFormChanged()) {
      AlertDialog.warning({
        title: $i18n.get({
          id: 'main.pages.Skill.Create.confirmReturn',
          dm: '确认返回吗',
        }),
        children: $i18n.get({
          id: 'main.pages.Skill.Create.returnWarning',
          dm: '返回将不会保存当前编辑的内容，确认返回吗？',
        }),
        onOk: () => {
          navigate('/skill');
        },
      });
      return;
    }
    navigate('/skill');
  };

  const handleOk = async () => {
    if (saveLoading) return;

    try {
      const values = await form.validateFields();
      if (!values.packageObjectKey) {
        message.error(
          $i18n.get({
            id: 'main.pages.Skill.Create.uploadPackageFirst',
            dm: '请先上传 Skill 压缩包',
          }),
        );
        return;
      }
      const manifest = values.manifest?.trim() || '{}';
      try {
        JSON.parse(manifest);
      } catch (error) {
        message.error(
          $i18n.get({
            id: 'main.pages.Skill.Create.invalidManifest',
            dm: 'Manifest 不是有效的 JSON 格式',
          }),
        );
        return;
      }

      setSaveLoading(true);
      const mainFilePath = values.mainFilePath || DEFAULT_MAIN_FILE;
      const apiParams: ICreateSkillParams = {
        name: values.name,
        description: values.description || '',
        tags: values.tags || '',
        main_file_path: mainFilePath,
        manifest,
        storage_type: values.storageType,
        storage_bucket: values.storageBucket,
        storage_prefix: values.storagePrefix,
        package_object_key: values.packageObjectKey || '',
        content_hash: values.contentHash,
        file_count: values.fileCount,
        total_size_bytes: values.totalSizeBytes,
        status: initialData?.status ?? SkillStatus.DRAFT,
      };

      if (skillCode) {
        const updateParams: IUpdateSkillParams = {
          ...apiParams,
          skill_code: skillCode,
        };
        await updateSkill(updateParams);
        message.success(
          $i18n.get({
            id: 'main.pages.Skill.Create.updateSuccess',
            dm: 'Skill更新成功',
          }),
        );
      } else {
        await createSkill(apiParams);
        message.success(
          $i18n.get({
            id: 'main.pages.Skill.Create.createSuccess',
            dm: 'Skill创建成功',
          }),
        );
      }

      navigate('/skill');
    } finally {
      setSaveLoading(false);
    }
  };

  const handlePackageUpload = (options: any) => {
    const file = options.file as File;
    setPackageUploading(true);
    uploadSkillPackage(file)
      .then((res) => {
        const data = res.data;
        form.setFieldsValue({
          mainFilePath: data.main_file_path || DEFAULT_MAIN_FILE,
          storageType: data.storage_type,
          storageBucket: data.storage_bucket,
          storagePrefix: data.storage_prefix,
          packageObjectKey: data.package_object_key,
          contentHash: data.content_hash,
          fileCount: data.file_count,
          totalSizeBytes: data.total_size_bytes,
          manifest: data.manifest || '{}',
        });
        setPackageFileList([
          {
            uid: data.package_object_key,
            name: file.name,
            status: 'done',
          },
        ]);
        options.onSuccess?.(data);
        message.success(
          $i18n.get({
            id: 'main.pages.Skill.Create.packageUploadSuccess',
            dm: 'Skill压缩包上传并校验成功',
          }),
        );
      })
      .catch((error) => {
        options.onError?.(error);
        setPackageFileList([]);
      })
      .finally(() => {
        setPackageUploading(false);
      });
  };

  const handleBeforeUpload = (file: File) => {
    const isZip = /\.zip$/i.test(file.name);
    if (!isZip) {
      message.error(
        $i18n.get({
          id: 'main.pages.Skill.Create.onlyZipPackage',
          dm: '请上传 zip 格式的 Skill 压缩包',
        }),
      );
      return Upload.LIST_IGNORE;
    }
    return true;
  };

  const clearPackageMeta = () => {
    form.setFieldsValue({
      storageType: undefined,
      storageBucket: undefined,
      storagePrefix: undefined,
      packageObjectKey: undefined,
      contentHash: undefined,
      fileCount: undefined,
      totalSizeBytes: undefined,
    });
    setPackageFileList([]);
  };

  const renderOkText = useMemo(() => {
    if (saveLoading) {
      return skillCode
        ? $i18n.get({ id: 'main.pages.Skill.Create.saving', dm: '保存中...' })
        : $i18n.get({
            id: 'main.pages.Skill.Create.creating',
            dm: '创建中...',
          });
    }
    return skillCode
      ? $i18n.get({ id: 'main.pages.Skill.Create.save', dm: '保存' })
      : $i18n.get({ id: 'main.pages.Skill.Create.create', dm: '创建' });
  }, [saveLoading, skillCode]);

  return (
    <InnerLayout
      loading={loading}
      breadcrumbLinks={[
        {
          title: $i18n.get({
            id: 'main.pages.Skill.Create.skillManagement',
            dm: 'Skill管理',
          }),
          onClick: onBack,
        },
        {
          title: skillCode
            ? $i18n.get({
                id: 'main.pages.Skill.Create.editSkill',
                dm: '编辑Skill',
              })
            : $i18n.get({
                id: 'main.pages.Skill.Create.createSkill',
                dm: '创建Skill',
              }),
        },
      ]}
      bottom={
        <div className={styles['bottom-container']}>
          <Button loading={saveLoading} onClick={handleOk} type="primary">
            {renderOkText}
          </Button>
          <Button onClick={onBack}>
            {$i18n.get({ id: 'main.pages.Skill.Create.cancel', dm: '取消' })}
          </Button>
        </div>
      }
    >
      <div className={styles.page}>
        <Flex className={styles.container} vertical>
          <div className={styles['content-wrap']}>
            <Form className={styles.content} form={form} layout="vertical">
              <Form.Item
                required
                label={$i18n.get({
                  id: 'main.pages.Skill.Create.name',
                  dm: 'Skill名称',
                })}
                name="name"
                rules={[
                  {
                    required: true,
                    message: $i18n.get({
                      id: 'main.pages.Skill.Create.enterName',
                      dm: '请输入Skill名称',
                    }),
                  },
                ]}
              >
                <Input
                  className={styles['fixed-width']}
                  showCount
                  maxLength={30}
                  placeholder={$i18n.get({
                    id: 'main.pages.Skill.Create.namePlaceholder',
                    dm: '例如 SQL 审查',
                  })}
                />
              </Form.Item>

              <Form.Item
                name="description"
                label={$i18n.get({
                  id: 'main.pages.Skill.Create.description',
                  dm: '描述',
                })}
              >
                <Input.TextArea
                  className={styles['fixed-width']}
                  showCount
                  maxLength={160}
                  autoSize={{ minRows: 2, maxRows: 3 }}
                  placeholder={$i18n.get({
                    id: 'main.pages.Skill.Create.descriptionPlaceholder',
                    dm: '描述这个Skill适合处理的问题',
                  })}
                />
              </Form.Item>

              <Form.Item
                name="tags"
                label={$i18n.get({
                  id: 'main.pages.Skill.Create.tags',
                  dm: '标签',
                })}
              >
                <Input
                  className={styles['fixed-width']}
                  placeholder={$i18n.get({
                    id: 'main.pages.Skill.Create.tagsPlaceholder',
                    dm: '用英文逗号分隔，例如 sql,review',
                  })}
                />
              </Form.Item>

              <Form.Item
                name="mainFilePath"
                label={$i18n.get({
                  id: 'main.pages.Skill.Create.mainFilePath',
                  dm: '主文件路径',
                })}
                rules={[
                  {
                    required: true,
                    message: $i18n.get({
                      id: 'main.pages.Skill.Create.enterMainFilePath',
                      dm: '请输入主文件路径',
                    }),
                  },
                ]}
              >
                <Input
                  className={styles['fixed-width']}
                  placeholder={DEFAULT_MAIN_FILE}
                />
              </Form.Item>

              <Form.Item
                required
                label={$i18n.get({
                  id: 'main.pages.Skill.Create.skillPackage',
                  dm: 'Skill压缩包',
                })}
              >
                <div className={styles['upload-width']}>
                  <Upload
                    accept=".zip"
                    beforeUpload={handleBeforeUpload}
                    customRequest={handlePackageUpload}
                    fileList={packageFileList}
                    maxCount={1}
                    onRemove={clearPackageMeta}
                  >
                    <Button
                      icon={<IconFont type="spark-upload-line" />}
                      loading={packageUploading}
                    >
                      {$i18n.get({
                        id: 'main.pages.Skill.Create.uploadZip',
                        dm: '上传 zip 包',
                      })}
                    </Button>
                  </Upload>
                  <Form.Item noStyle shouldUpdate>
                    {() => {
                      const fileCount = form.getFieldValue('fileCount');
                      const totalSizeBytes = form.getFieldValue('totalSizeBytes');
                      if (fileCount === undefined && totalSizeBytes === undefined) {
                        return null;
                      }
                      return (
                        <div className={styles['package-meta']}>
                          {$i18n.get(
                            {
                              id: 'main.pages.Skill.Create.packageMeta',
                              dm: '文件数 {fileCount}，总大小 {totalSizeBytes} bytes',
                            },
                            {
                              fileCount: fileCount ?? 0,
                              totalSizeBytes: totalSizeBytes ?? 0,
                            },
                          )}
                        </div>
                      );
                    }}
                  </Form.Item>
                </div>
              </Form.Item>

              <Form.Item hidden name="storageType">
                <Input />
              </Form.Item>
              <Form.Item hidden name="storageBucket">
                <Input />
              </Form.Item>
              <Form.Item hidden name="storagePrefix">
                <Input />
              </Form.Item>
              <Form.Item hidden name="packageObjectKey">
                <Input />
              </Form.Item>
              <Form.Item hidden name="contentHash">
                <Input />
              </Form.Item>
              <Form.Item hidden name="fileCount">
                <Input />
              </Form.Item>
              <Form.Item hidden name="totalSizeBytes">
                <Input />
              </Form.Item>
              <Form.Item hidden name="manifest">
                <Input />
              </Form.Item>

            </Form>
          </div>
        </Flex>
      </div>
    </InnerLayout>
  );
}
