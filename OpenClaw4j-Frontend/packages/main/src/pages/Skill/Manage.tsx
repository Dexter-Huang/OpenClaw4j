import CardList from '@/components/Card/List';
import { useInnerLayout } from '@/components/InnerLayout/utils';
import $i18n from '@/i18n';
import { deleteSkill, listSkills } from '@/services/skill';
import { ISkill, ISkillPagingList } from '@/types/skill';
import {
  AlertDialog,
  Button,
  ButtonProps,
  IconFont,
  message,
} from '@spark-ai/design';
import { useMount, useSetState } from 'ahooks';
import { memo } from 'react';
import { useNavigate } from 'react-router-dom';
import { history } from 'umi';
import SkillCard from './components/SkillCard';
import styles from './Manage.module.less';

export const CreateSkillBtn = memo(
  (props: {
    buttonProps?: ButtonProps;
    text?: string;
    isOpenNew?: boolean;
  }) => {
    const handleCreateSkill = () => {
      if (props.isOpenNew) {
        window.open('/skill/create');
      } else {
        history.push('/skill/create');
      }
    };

    return (
      <Button
        onClick={handleCreateSkill}
        type="primary"
        icon={<IconFont type="spark-plus-line" />}
        {...props.buttonProps}
      >
        {props.text ||
          $i18n.get({
            id: 'main.pages.Skill.Manage.createSkill',
            dm: '创建Skill',
          })}
      </Button>
    );
  },
);

export default function SkillManage() {
  const { rightPortal } = useInnerLayout();
  const navigate = useNavigate();

  const [state, setState] = useSetState<{
    list: ISkill[];
    pageNo: number;
    pageSize: number;
    total: number;
    loading: boolean;
  }>({
    list: [],
    pageNo: 1,
    pageSize: 50,
    total: 0,
    loading: false,
  });

  const fetchList = async (
    extraParams = {} as Partial<{ pageNo: number; pageSize: number }>,
  ) => {
    setState({ loading: true });
    try {
      const queryParams = {
        current: extraParams.pageNo ?? state.pageNo,
        size: extraParams.pageSize ?? state.pageSize,
        need_files: false,
      };
      const response = await listSkills(queryParams);
      if (response?.data) {
        const pagingData = response.data as ISkillPagingList;
        setState({
          list: pagingData.records || [],
          total: pagingData.total || 0,
          pageNo: queryParams.current,
          pageSize: queryParams.size,
        });
      }
    } finally {
      setState({ loading: false });
    }
  };

  useMount(() => {
    fetchList();
  });

  const handleConfirmDelete = (item: ISkill) => {
    AlertDialog.warning({
      title: $i18n.get({
        id: 'main.pages.Skill.Manage.confirmDelete',
        dm: '确认删除此Skill吗',
      }),
      children: $i18n.get({
        id: 'main.pages.Skill.Manage.deleteWarning',
        dm: '删除后将不可恢复，已经添加该Skill的智能体可能会失效，请谨慎操作。',
      }),
      danger: true,
      onOk: async () => {
        await deleteSkill(item.skill_code);
        message.success(
          $i18n.get({
            id: 'main.pages.Skill.Manage.deleteSuccess',
            dm: '删除成功',
          }),
        );
        fetchList();
      },
    });
  };

  const handleAction = (action?: string, item?: ISkill) => {
    if (!action || !item) return;
    switch (action) {
      case 'delete':
        handleConfirmDelete(item);
        break;
      case 'edit':
        navigate(`/skill/edit/${item.skill_code}`);
        break;
      case 'detail':
      default:
        navigate(`/skill/detail/${item.skill_code}`);
    }
  };

  const handlePageChange = (page: number, pageSize: number) => {
    fetchList({ pageNo: page, pageSize });
  };

  return (
    <div className={styles.container}>
      {state.list.length > 0 && rightPortal(<CreateSkillBtn />)}
      <CardList
        loading={state.loading}
        pagination={{
          current: state.pageNo,
          total: state.total,
          pageSize: state.pageSize,
          onChange: handlePageChange,
        }}
        emptyAction={<CreateSkillBtn />}
      >
        {state.list.map((item) => (
          <SkillCard key={item.skill_code} data={item} onClick={handleAction} />
        ))}
      </CardList>
    </div>
  );
}
