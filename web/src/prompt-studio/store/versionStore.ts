import { create } from 'zustand';
import { promptStudioApi } from '@/prompt-studio/api/client';
import type { Version } from '@/prompt-studio/models/Version';
import { normalize } from '@/prompt-studio/utils/normalize';

interface VersionState {
  // State
  versions: Version[];
  currentVersionId: string | null;
  compareMode: boolean; // 对比选择模式状态
  compareState: {
    isOpen: boolean;
    sourceVersionId: string | null;
    targetVersionId: string | null;
  };

  // Actions
  loadVersions: (projectId: string) => Promise<void>;
  loadAllVersions: () => Promise<Version[]>;
  createVersion: (
    projectId: string,
    content: string,
    parentId: string | null,
    skipDuplicateCheck?: boolean,
    name?: string,
  ) => Promise<string>;
  updateVersionInPlace: (
    id: string,
    content: string,
    name?: string,
  ) => Promise<void>;
  updateVersionName: (id: string, name: string) => Promise<void>;
  deleteVersion: (id: string) => Promise<void>;
  updateVersionScore: (id: string, score: number) => Promise<void>;
  updateVersionNotes: (id: string, notes: string) => Promise<void>;
  setCurrentVersion: (id: string | null) => void;
  openCompare: (sourceVersionId: string) => void;
  setCompareTarget: (targetVersionId: string) => void;
  closeCompare: () => void;
  toggleCompareMode: (sourceVersionId?: string | null) => void;
}

/** 补齐运行时字段 normalizedContent（不落库，仅前端搜索/比对使用） */
const withNormalized = (version: Version): Version => ({
  ...version,
  normalizedContent: normalize(version.content),
});

const upsert = (versions: Version[], next: Version): Version[] => {
  const index = versions.findIndex((v) => v.id === next.id);
  if (index === -1) return [...versions, next];
  const copy = [...versions];
  copy[index] = next;
  return copy;
};

export const useVersionStore = create<VersionState>((set, get) => ({
  versions: [],
  currentVersionId: null,
  compareMode: false,
  compareState: {
    isOpen: false,
    sourceVersionId: null,
    targetVersionId: null,
  },

  loadVersions: async (projectId) => {
    const versions = await promptStudioApi.listVersions(projectId);
    set({ versions: versions.map(withNormalized) });
  },

  /** 全局搜索用：拉取当前用户的全部版本，不写入 versions 状态 */
  loadAllVersions: async () => {
    const versions = await promptStudioApi.listVersions();
    return versions.map(withNormalized);
  },

  createVersion: async (
    projectId,
    content,
    parentId,
    skipDuplicateCheck = false,
    name,
  ) => {
    // 根版本唯一性与重复检测由服务端校验
    // 重复时服务端返回 message 为 DUPLICATE_DETECTED:<id>，这里原样向上抛
    const version = await promptStudioApi.createVersion({
      projectId,
      content,
      parentId,
      name,
      skipDuplicateCheck,
    });

    set((state) => ({
      versions: [...state.versions, withNormalized(version)],
      currentVersionId: version.id,
    }));

    return version.id;
  },

  updateVersionInPlace: async (id, content, name) => {
    const version = await promptStudioApi.updateVersion(id, {
      content,
      ...(name !== undefined ? { name } : {}),
    });
    set((state) => ({
      versions: upsert(state.versions, withNormalized(version)),
    }));
  },

  updateVersionName: async (id, name) => {
    const version = await promptStudioApi.updateVersion(id, { name });
    set((state) => ({
      versions: upsert(state.versions, withNormalized(version)),
    }));
  },

  deleteVersion: async (id) => {
    const target = get().versions.find((v) => v.id === id);
    // 子版本"接骨"、附件清理与项目 updatedAt 刷新由服务端在事务内完成
    await promptStudioApi.deleteVersion(id);
    if (target) {
      await get().loadVersions(target.projectId);
    }
    set((state) => ({
      currentVersionId:
        state.currentVersionId === id ? null : state.currentVersionId,
    }));
  },

  updateVersionScore: async (id, score) => {
    // 评分属于轻量修改，不同步刷新项目 updatedAt，与前端原实现一致
    const version = await promptStudioApi.updateVersion(id, {
      score,
      touchProject: false,
    });
    set((state) => ({
      versions: upsert(state.versions, withNormalized(version)),
    }));
  },

  updateVersionNotes: async (id, notes) => {
    const version = await promptStudioApi.updateVersion(id, {
      notes,
      touchProject: false,
    });
    set((state) => ({
      versions: upsert(state.versions, withNormalized(version)),
    }));
  },

  setCurrentVersion: (id) => {
    set({ currentVersionId: id });
  },

  openCompare: (sourceVersionId) => {
    set({
      compareState: {
        isOpen: true,
        sourceVersionId,
        targetVersionId: null,
      },
    });
  },

  setCompareTarget: (targetVersionId) => {
    set((state) => {
      const newState = {
        ...state.compareState,
        targetVersionId,
      };

      // 如果目标版本已设置，则打开对比模态框
      if (targetVersionId) {
        newState.isOpen = true;
      }

      return {
        compareState: newState,
        compareMode: false, // 设置目标后退出对比模式
        // 不改变currentVersionId，保持对比前的版本选中状态
      };
    });
  },

  closeCompare: () => {
    set((_) => ({
      compareState: {
        isOpen: false,
        sourceVersionId: null,
        targetVersionId: null,
      },
      compareMode: false,
      // 不改变currentVersionId，保持对比前的版本选中状态
    }));
  },

  toggleCompareMode: (sourceVersionId = null) => {
    const currentCompareMode = get().compareMode;

    if (currentCompareMode) {
      // 退出对比模式
      set({ compareMode: false });
    } else {
      // 进入对比模式
      if (!sourceVersionId) {
        sourceVersionId = get().currentVersionId;
      }

      if (sourceVersionId) {
        set({
          compareMode: true,
          compareState: {
            isOpen: false,
            sourceVersionId,
            targetVersionId: null,
          },
        });
      }
    }
  },
}));
