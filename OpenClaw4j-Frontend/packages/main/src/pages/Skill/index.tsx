import InnerLayout from '@/components/InnerLayout';
import $i18n from '@/i18n';
import SkillManage from './Manage';

export default function SkillIndex() {
  return (
    <InnerLayout
      breadcrumbLinks={[
        {
          title: $i18n.get({
            id: 'main.pages.App.index.home',
            dm: '首页',
          }),
          path: '/',
        },
        {
          title: $i18n.get({
            id: 'main.pages.Skill.index.skillManagement',
            dm: 'Skill管理',
          }),
        },
      ]}
    >
      <SkillManage />
    </InnerLayout>
  );
}
