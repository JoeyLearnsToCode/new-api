/**
 * 初始化示例数据服务
 * 为全新用户创建示例项目和版本
 *
 * 数据已搬到服务端，示例项目由后端在单个事务内创建，
 * 文案按当前语言由前端传入（缺省时后端回落为中文）
 */

import { promptStudioApi } from '@/prompt-studio/api/client';
import { useProjectStore } from '@/prompt-studio/store/projectStore';
import { storage, STORAGE_KEYS } from '@/prompt-studio/utils/storage';
import { translations } from '@/prompt-studio/i18n/locales';
import { initializeLanguage } from '@/prompt-studio/i18n/detectLanguage';
import type { Locale } from '@/prompt-studio/i18n/types';

/**
 * 检查是否为全新用户（没有任何本地痕迹，且服务端也没有项目）
 */
async function isNewUser(): Promise<boolean> {
  const hasOpenedBefore =
    storage.get(STORAGE_KEYS.FIRST_OPEN_TIME, null) !== null;
  if (hasOpenedBefore) {
    return false;
  }
  return useProjectStore.getState().projects.length === 0;
}

/**
 * 创建示例项目和版本
 * @returns 创建的项目ID
 */
async function createSampleProject(): Promise<string> {
  const locale: Locale = initializeLanguage();
  const t = translations[locale];

  const { project } = await promptStudioApi.createSample({
    projectName: t.sampleData.projectName,
    rootName: t.sampleData.versions.root.name,
    rootContent: t.sampleData.versions.root.content,
    branch1Name: t.sampleData.versions.branch1.name,
    branch1Content: t.sampleData.versions.branch1.content,
    branch2Name: t.sampleData.versions.branch2.name,
    branch2Content: t.sampleData.versions.branch2.content,
  });

  return project.id;
}

/**
 * 初始化示例数据（如果是全新用户）
 * @returns 如果创建了示例项目，返回项目ID；否则返回null
 */
export async function initializeSampleData(): Promise<string | null> {
  try {
    if (await isNewUser()) {
      console.log('检测到全新用户，正在创建示例项目...');
      const projectId = await createSampleProject();
      console.log('示例项目创建完成');
      return projectId;
    }
    return null;
  } catch (error) {
    console.error('创建示例项目失败:', error);
    return null;
  }
}
