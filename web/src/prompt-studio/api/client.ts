/**
 * Prompt Studio 后端接口客户端
 *
 * 原实现直接读写浏览器 IndexedDB（Dexie），这里改为访问 new-api 的
 * /api/prompt-studio 接口：数据落在服务端数据库，浏览器只做缓存。
 *
 * - 复用 new-api 的 API 实例，自动带上 New-Api-User 头与会话 cookie
 * - 后端用空字符串表示"根节点"，前端用 null，这里统一做转换
 */

import { API } from '@/helpers/api';
import type { Attachment } from '@/prompt-studio/models/Attachment';
import type { Folder } from '@/prompt-studio/models/Folder';
import type { Project } from '@/prompt-studio/models/Project';
import type { Snippet } from '@/prompt-studio/models/Snippet';
import type { Version } from '@/prompt-studio/models/Version';

const BASE = '/api/prompt-studio';

/** 后端返回 success:false 时抛出，message 原样透出（含 DUPLICATE_DETECTED: 前缀） */
export class PromptStudioRequestError extends Error {}

async function unwrap<T>(promise: Promise<any>): Promise<T> {
  const res = await promise;
  const body = res?.data;
  if (!body || body.success === false) {
    throw new PromptStudioRequestError(body?.message || '请求失败');
  }
  return body.data as T;
}

/** 空字符串（后端表示根）转换为 null（前端表示根） */
const toRef = (value?: string | null): string | null => (value ? value : null);

const toFolder = (raw: any): Folder => ({
  id: raw.id,
  name: raw.name,
  parentId: toRef(raw.parentId),
  createdAt: raw.createdAt,
});

const toProject = (raw: any): Project => ({
  id: raw.id,
  folderId: toRef(raw.folderId),
  name: raw.name,
  tags: raw.tags ?? undefined,
  createdAt: raw.createdAt,
  updatedAt: raw.updatedAt,
});

const toVersion = (raw: any): Version => ({
  id: raw.id,
  projectId: raw.projectId,
  parentId: toRef(raw.parentId),
  content: raw.content ?? '',
  contentHash: raw.contentHash ?? '',
  name: raw.name ?? undefined,
  score: raw.score ?? undefined,
  notes: raw.notes ?? undefined,
  createdAt: raw.createdAt,
  updatedAt: raw.updatedAt,
});

const toAttachment = (raw: any): Attachment => ({
  id: raw.id,
  versionId: raw.versionId,
  fileName: raw.fileName,
  fileType: raw.fileType,
  size: raw.size,
});

/** 导入导出用的整包结构，与后端 model.PromptStudioExportData / ImportRequest 对应 */
export interface PromptStudioBulkData {
  folders: Folder[];
  projects: Project[];
  versions: Version[];
  snippets: Snippet[];
  attachments: Attachment[];
}

export const promptStudioApi = {
  // ---------- 文件夹 ----------
  listFolders: async (): Promise<Folder[]> => {
    const raw = await unwrap<any[]>(API.get(`${BASE}/folders`));
    return (raw ?? []).map(toFolder);
  },
  createFolder: async (
    name: string,
    parentId: string | null,
  ): Promise<Folder> => {
    const raw = await unwrap<any>(
      API.post(`${BASE}/folders`, { name, parentId: parentId ?? '' }),
    );
    return toFolder(raw);
  },
  renameFolder: async (id: string, name: string): Promise<void> => {
    await unwrap(API.put(`${BASE}/folders/${id}`, { name }));
  },
  deleteFolder: async (id: string): Promise<void> => {
    await unwrap(API.delete(`${BASE}/folders/${id}`));
  },

  // ---------- 项目 ----------
  listProjects: async (folderId?: string): Promise<Project[]> => {
    const url = folderId
      ? `${BASE}/projects?folderId=${folderId}`
      : `${BASE}/projects`;
    const raw = await unwrap<any[]>(API.get(url));
    return (raw ?? []).map(toProject);
  },
  createProject: async (
    name: string,
    folderId: string | null,
  ): Promise<{ project: Project; version: Version }> => {
    const raw = await unwrap<any>(
      API.post(`${BASE}/projects`, { name, folderId: folderId ?? '' }),
    );
    return { project: toProject(raw.project), version: toVersion(raw.version) };
  },
  updateProject: async (
    id: string,
    patch: { name?: string; tags?: Project['tags']; folderId?: string | null },
  ): Promise<Project> => {
    const raw = await unwrap<any>(
      API.put(`${BASE}/projects/${id}`, {
        ...patch,
        // 后端用空串表示根目录，null 会被当作"不更新"
        ...(patch.folderId === null ? { folderId: '' } : {}),
      }),
    );
    return toProject(raw);
  },
  deleteProject: async (id: string): Promise<void> => {
    await unwrap(API.delete(`${BASE}/projects/${id}`));
  },

  // ---------- 版本 ----------
  /** projectId 为空时返回当前用户的全部版本（全局搜索用） */
  listVersions: async (projectId?: string): Promise<Version[]> => {
    const url = projectId
      ? `${BASE}/versions?projectId=${projectId}`
      : `${BASE}/versions`;
    const raw = await unwrap<any[]>(API.get(url));
    return (raw ?? []).map(toVersion);
  },
  createVersion: async (payload: {
    projectId: string;
    content: string;
    parentId: string | null;
    name?: string;
    skipDuplicateCheck?: boolean;
  }): Promise<Version> => {
    const raw = await unwrap<any>(API.post(`${BASE}/versions`, payload));
    return toVersion(raw);
  },
  updateVersion: async (
    id: string,
    patch: {
      content?: string;
      name?: string | null;
      score?: number | null;
      notes?: string | null;
      touchProject?: boolean;
    },
  ): Promise<Version> => {
    const raw = await unwrap<any>(API.put(`${BASE}/versions/${id}`, patch));
    return toVersion(raw);
  },
  deleteVersion: async (id: string): Promise<void> => {
    await unwrap(API.delete(`${BASE}/versions/${id}`));
  },

  // ---------- 附件 ----------
  listAttachments: async (versionId: string): Promise<Attachment[]> => {
    const raw = await unwrap<any[]>(
      API.get(`${BASE}/attachments?versionId=${versionId}`),
    );
    return (raw ?? []).map(toAttachment);
  },
  /** 一次请求提交多个文件，避免前端逐个循环上传 */
  uploadAttachments: async (
    versionId: string,
    files: File[],
  ): Promise<Attachment[]> => {
    const form = new FormData();
    for (const file of files) {
      form.append('file', file);
    }
    const raw = await unwrap<any[]>(
      API.post(`${BASE}/versions/${versionId}/attachments`, form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      }),
    );
    return (raw ?? []).map(toAttachment);
  },
  /** 拉取附件二进制内容，用于预览与下载 */
  fetchAttachmentBlob: async (id: string): Promise<Blob> => {
    const res = await API.get(`${BASE}/attachments/${id}/content`, {
      responseType: 'blob',
    });
    return res.data as Blob;
  },
  deleteAttachment: async (id: string): Promise<void> => {
    await unwrap(API.delete(`${BASE}/attachments/${id}`));
  },

  // ---------- 批量：导入导出与示例数据 ----------
  exportAll: async (): Promise<PromptStudioBulkData> =>
    unwrap<PromptStudioBulkData>(API.get(`${BASE}/export`)),
  importAll: async (
    data: PromptStudioBulkData,
    mode: 'merge' | 'overwrite',
  ): Promise<Record<string, number>> =>
    unwrap(API.post(`${BASE}/import`, { mode, ...data })),
  createSample: async (sample: {
    projectName: string;
    rootName: string;
    rootContent: string;
    branch1Name: string;
    branch1Content: string;
    branch2Name: string;
    branch2Content: string;
  }): Promise<{ project: Project; version: Version }> => {
    const raw = await unwrap<any>(API.post(`${BASE}/sample`, sample));
    return { project: toProject(raw.project), version: toVersion(raw.version) };
  },
};
