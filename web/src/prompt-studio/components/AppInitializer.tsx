/**
 * 应用初始化组件
 * 负责初始化示例数据并自动加载到UI
 *
 * 嵌入 new-api 后的调整：
 * - 数据来自服务端，初始化改为先加载再做示例数据判断
 * - 主题跟随 new-api 全局主题，不再自行操作 documentElement 的 dark class
 */

import { useEffect, useState, useRef, useCallback } from 'react';
import { useActualTheme } from '@/context/Theme';
import { initializeSampleData } from '@/prompt-studio/services/initializeSampleData';
import { useProjectStore } from '@/prompt-studio/store/projectStore';
import { useVersionStore } from '@/prompt-studio/store/versionStore';
import { useSettingsStore } from '@/prompt-studio/store/settingsStore';
import { useTranslation } from '@/prompt-studio/i18n/I18nContext';
import { storage, STORAGE_KEYS } from '@/prompt-studio/utils/storage';

interface AppInitializerProps {
  children: React.ReactNode;
}

/**
 * 从 URL hash 中解析项目 ID
 * 格式: #/project/{projectId}
 */
const getProjectIdFromUrl = (): string | null => {
  const hash = window.location.hash;
  if (!hash) return null;
  const match = hash.match(/^#\/project\/(.+)$/);
  return match ? match[1] : null;
};

export const AppInitializer: React.FC<AppInitializerProps> = ({ children }) => {
  const [isInitialized, setIsInitialized] = useState(false);
  const {
    loadProjects,
    setCurrentProject,
    loadFolders,
    expandFolderPathToProject,
  } = useProjectStore();
  const { loadVersions } = useVersionStore();
  const { setTheme } = useSettingsStore();
  const actualTheme = useActualTheme();
  const t = useTranslation();

  // 使用 ref 确保初始化只执行一次（防止 React 18 严格模式下的双重调用）
  const hasInitialized = useRef(false);

  // 主题统一跟随 new-api 的全局主题，同步给依赖 settingsStore 的子组件（Monaco、画布等）
  useEffect(() => {
    setTheme(actualTheme === 'dark' ? 'dark' : 'light');
  }, [actualTheme, setTheme]);

  // 处理从 URL 打开项目的函数
  const handleOpenProjectFromUrl = useCallback(
    async (projectId: string) => {
      setCurrentProject(projectId);
      await expandFolderPathToProject(projectId);
      await loadVersions(projectId);
    },
    [setCurrentProject, expandFolderPathToProject, loadVersions],
  );

  // 监听 hashchange 事件以处理浏览器前进/后退
  useEffect(() => {
    const handleHashChange = async () => {
      const urlProjectId = getProjectIdFromUrl();
      if (!urlProjectId) return;
      const project = await useProjectStore.getState().getProject(urlProjectId);
      if (project) {
        await handleOpenProjectFromUrl(urlProjectId);
      } else {
        alert(t('errors.projectNotFound'));
      }
    };

    window.addEventListener('hashchange', handleHashChange);
    return () => window.removeEventListener('hashchange', handleHashChange);
  }, [handleOpenProjectFromUrl, t]);

  useEffect(() => {
    // 如果已经初始化过，直接返回
    if (hasInitialized.current) {
      return;
    }
    hasInitialized.current = true;

    const initialize = async () => {
      // 先加载文件夹和项目，示例数据是否创建依赖项目数量
      await loadFolders();
      await loadProjects();

      // 检查 URL 中是否有项目 ID 参数
      const urlProjectId = getProjectIdFromUrl();
      const urlProject = urlProjectId
        ? await useProjectStore.getState().getProject(urlProjectId)
        : undefined;

      if (urlProjectId && urlProject) {
        // 项目存在，打开它
        await handleOpenProjectFromUrl(urlProjectId);
      } else if (urlProjectId) {
        // 项目不存在，提示用户并回到首页
        alert(t('errors.projectNotFound'));
        window.location.hash = '';
      } else {
        // 没有指定项目：全新用户则创建示例项目并自动选中
        const sampleProjectId = await initializeSampleData();
        if (sampleProjectId) {
          setCurrentProject(sampleProjectId);
          await loadVersions(sampleProjectId);
        }
      }

      if (storage.get(STORAGE_KEYS.FIRST_OPEN_TIME, null) === null) {
        // 记录首次打开时间
        storage.set(STORAGE_KEYS.FIRST_OPEN_TIME, Date.now());
      }

      setIsInitialized(true);
    };

    initialize();
  }, [
    loadProjects,
    loadFolders,
    setCurrentProject,
    loadVersions,
    handleOpenProjectFromUrl,
    t,
  ]);

  // 在初始化完成前，可以显示一个简单的加载提示
  if (!isInitialized) {
    return null; // 或者可以显示一个加载动画
  }

  return <>{children}</>;
};
