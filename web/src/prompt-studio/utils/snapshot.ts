/**
 * 浏览器本地快照
 *
 * 数据以服务端为准，这里只缓存"文件夹 / 项目"两份列表用于加速首屏渲染：
 * 挂载时先用快照出画面，接口返回后立即覆盖。所有写操作都发往服务端，
 * 不写入快照之外的持久化。
 */

import { storage, STORAGE_KEYS } from '@/prompt-studio/utils/storage';
import type { Folder } from '@/prompt-studio/models/Folder';
import type { Project } from '@/prompt-studio/models/Project';

const FOLDERS_KEY = STORAGE_KEYS.SNAPSHOT_FOLDERS;
const PROJECTS_KEY = STORAGE_KEYS.SNAPSHOT_PROJECTS;

export const promptStudioSnapshot = {
  readFolders: (): Folder[] | null =>
    storage.get<Folder[] | null>(FOLDERS_KEY, null),
  writeFolders: (folders: Folder[]): void => storage.set(FOLDERS_KEY, folders),
  readProjects: (): Project[] | null =>
    storage.get<Project[] | null>(PROJECTS_KEY, null),
  writeProjects: (projects: Project[]): void =>
    storage.set(PROJECTS_KEY, projects),
  clear: (): void => {
    storage.remove(FOLDERS_KEY);
    storage.remove(PROJECTS_KEY);
  },
};
