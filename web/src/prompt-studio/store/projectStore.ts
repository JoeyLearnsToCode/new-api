import { create } from 'zustand';
import { promptStudioApi } from '@/prompt-studio/api/client';
import type { Folder } from '@/prompt-studio/models/Folder';
import type { Project } from '@/prompt-studio/models/Project';
import { useUiStore } from '@/prompt-studio/store/uiStore';
import { promptStudioSnapshot } from '@/prompt-studio/utils/snapshot';

interface ProjectState {
  // State
  folders: Folder[];
  projects: Project[];
  currentProjectId: string | null;

  // Actions
  loadFolders: () => Promise<void>;
  loadProjects: (folderId?: string) => Promise<void>;
  createFolder: (name: string, parentId: string | null) => Promise<string>;
  renameFolder: (id: string, newName: string) => Promise<void>;
  deleteFolder: (id: string) => Promise<void>;
  createProject: (name: string, folderId: string) => Promise<string>;
  renameProject: (id: string, newName: string) => Promise<void>;
  updateProjectTags: (id: string, tags: Project['tags']) => Promise<void>;
  deleteProject: (id: string) => Promise<void>;
  selectProject: (id: string, options?: { updateUrl?: boolean }) => void;
  setCurrentProject: (id: string | null) => void;
  moveProject: (projectId: string, folderId: string) => Promise<void>;
  expandFolderPathToProject: (projectId: string) => Promise<void>;
  getProject: (id: string) => Promise<Project | undefined>;
  getProjectsByFolder: (folderId: string) => Promise<Project[]>;
}

export const useProjectStore = create<ProjectState>((set, get) => ({
  folders: [],
  projects: [],
  currentProjectId: null,

  loadFolders: async () => {
    // 先用本地快照渲染，再用服务端结果覆盖（快照只用于提速，不参与写操作）
    const cached = promptStudioSnapshot.readFolders();
    if (cached) set({ folders: cached });

    const folders = await promptStudioApi.listFolders();
    set({ folders });
    promptStudioSnapshot.writeFolders(folders);
  },

  loadProjects: async (folderId?: string) => {
    const cached = promptStudioSnapshot.readProjects();
    if (cached && !folderId) set({ projects: cached });

    const projects = await promptStudioApi.listProjects(folderId);
    set({ projects });
    if (!folderId) promptStudioSnapshot.writeProjects(projects);
  },

  createFolder: async (name, parentId) => {
    const folder = await promptStudioApi.createFolder(name, parentId);
    set((state) => ({ folders: [...state.folders, folder] }));
    promptStudioSnapshot.writeFolders(get().folders);
    return folder.id;
  },

  renameFolder: async (id, newName) => {
    await promptStudioApi.renameFolder(id, newName);
    set((state) => ({
      folders: state.folders.map((f) =>
        f.id === id ? { ...f, name: newName } : f,
      ),
    }));
    promptStudioSnapshot.writeFolders(get().folders);
  },

  deleteFolder: async (id) => {
    // 级联上提由服务端在事务内完成
    await promptStudioApi.deleteFolder(id);
    await get().loadFolders();
    await get().loadProjects();
  },

  createProject: async (name, folderId) => {
    // 服务端在同一事务内创建项目及其根版本
    const { project } = await promptStudioApi.createProject(
      name,
      folderId === 'root' ? null : folderId,
    );
    set((state) => ({ projects: [...state.projects, project] }));
    promptStudioSnapshot.writeProjects(get().projects);
    return project.id;
  },

  renameProject: async (id, newName) => {
    const project = await promptStudioApi.updateProject(id, { name: newName });
    set((state) => ({
      projects: state.projects.map((p) =>
        p.id === id
          ? { ...p, name: project.name, updatedAt: project.updatedAt }
          : p,
      ),
    }));
    promptStudioSnapshot.writeProjects(get().projects);
  },

  updateProjectTags: async (id, tags) => {
    const project = await promptStudioApi.updateProject(id, { tags });
    set((state) => ({
      projects: state.projects.map((p) =>
        p.id === id
          ? { ...p, tags: project.tags, updatedAt: project.updatedAt }
          : p,
      ),
    }));
    promptStudioSnapshot.writeProjects(get().projects);
  },

  deleteProject: async (id) => {
    // 版本与附件的级联删除由服务端在事务内完成
    await promptStudioApi.deleteProject(id);
    set((state) => ({
      projects: state.projects.filter((p) => p.id !== id),
      currentProjectId:
        state.currentProjectId === id ? null : state.currentProjectId,
    }));
    promptStudioSnapshot.writeProjects(get().projects);
  },

  selectProject: (id: string, options?: { updateUrl?: boolean }) => {
    set({ currentProjectId: id });
    // 可选：更新 URL
    if (options?.updateUrl !== false) {
      const hashRouter = window.location.hash.replace(/^#\/?|\/?$/g, '');
      if (hashRouter === `project/${id}` || hashRouter === `/#project/${id}`) {
        // 已经在正确的 URL，无需更新
      } else {
        window.location.hash = `#/project/${id}`;
      }
    }
  },

  setCurrentProject: (id) => {
    set({ currentProjectId: id });
  },

  moveProject: async (projectId, folderId) => {
    const target = folderId === 'root' ? null : folderId;
    const project = await promptStudioApi.updateProject(projectId, {
      folderId: target,
    });
    set((state) => ({
      projects: state.projects.map((p) =>
        p.id === projectId
          ? { ...p, folderId: project.folderId, updatedAt: project.updatedAt }
          : p,
      ),
    }));
    promptStudioSnapshot.writeProjects(get().projects);
  },

  getProject: async (id) => {
    const cached = get().projects.find((p) => p.id === id);
    if (cached) return cached;
    const list = await promptStudioApi.listProjects();
    return list.find((p) => p.id === id);
  },

  getProjectsByFolder: async (folderId) => {
    return await promptStudioApi.listProjects(folderId);
  },

  expandFolderPathToProject: async (projectId) => {
    const project = await get().getProject(projectId);
    if (!project || !project.folderId) return;

    // 文件夹树已在内存里，直接向上回溯出父链
    const folderMap = new Map(get().folders.map((f) => [f.id, f]));
    const parentFolderIds: string[] = [];
    let currentFolderId: string | null = project.folderId;

    while (currentFolderId) {
      const folder = folderMap.get(currentFolderId);
      if (!folder) break;
      parentFolderIds.unshift(folder.id);
      currentFolderId = folder.parentId;
    }

    const { expandFolder } = useUiStore.getState();
    for (const folderId of parentFolderIds) {
      expandFolder(folderId);
    }
  },
}));
