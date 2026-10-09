/**
 * 统一的导入服务
 *
 * 原实现逐表写入 IndexedDB；数据搬到服务端后改为：
 * 在浏览器里解析 ZIP / JSON，整理成整包后一次性 POST /api/prompt-studio/import，
 * 由后端在同一事务内完成 merge / overwrite，前端不做逐条循环。
 */

import JSZip from 'jszip';
import {
  promptStudioApi,
  type PromptStudioBulkData,
} from '@/prompt-studio/api/client';
import { storage, STORAGE_KEYS } from '@/prompt-studio/utils/storage';
import type {
  ImportOptions,
  ImportResult,
  ImportProgress,
  ImportProgressCallback,
} from '@/prompt-studio/types/import';

/** 附件二进制在 ZIP 里可能以独立文件存在（兼容原 prompt-studio 备份格式） */
const ATTACHMENT_EXTENSIONS = [
  '.jpg',
  '.jpeg',
  '.png',
  '.gif',
  '.webp',
  '.svg',
  '.mp4',
  '.webm',
  '.mov',
  '.pdf',
  '.txt',
  '.html',
  '.json',
  '.bin',
];

/**
 * 附件在 ZIP 里的候选文件名
 * 1. 附件原始文件名的扩展名（最可靠，导出时就是这么写的）
 * 2. 常见扩展名逐个试探（兼容其它来源的备份包）
 * 3. 不带扩展名（原 prompt-studio 的 WebDAV 备份就是以裸 id 存放的）
 */
const attachmentCandidates = (attachment: any): string[] => {
  const id = attachment?.id ?? '';
  const candidates: string[] = [];
  const fileName: string = attachment?.fileName ?? '';
  if (fileName.includes('.')) {
    candidates.push(`${id}.${fileName.split('.').pop()?.toLowerCase()}`);
  }
  for (const ext of ATTACHMENT_EXTENSIONS) {
    candidates.push(`${id}${ext}`);
  }
  candidates.push(id);
  return candidates;
};

const arrayBufferToBase64 = (buffer: ArrayBuffer): string => {
  const bytes = new Uint8Array(buffer);
  const chunkSize = 0x8000;
  let binary = '';
  for (let i = 0; i < bytes.length; i += chunkSize) {
    binary += String.fromCharCode(
      ...Array.from(bytes.subarray(i, i + chunkSize)),
    );
  }
  return btoa(binary);
};

const readJson = async (file: JSZip.JSZipObject | null): Promise<any[]> => {
  if (!file) return [];
  const text = await file.async('text');
  const parsed = JSON.parse(text);
  return Array.isArray(parsed) ? parsed : [];
};

export class ImportService {
  /**
   * 从 ZIP 文件导入数据
   */
  async importFromZip(
    file: File,
    options: ImportOptions,
    onProgress?: ImportProgressCallback,
  ): Promise<ImportResult> {
    const zip = await JSZip.loadAsync(file);

    const totalStages = 6; // projects, folders, versions, snippets, attachments, settings
    let currentStage = 0;
    const updateProgress = (
      stage: ImportProgress['stage'],
      message: string,
    ) => {
      currentStage++;
      onProgress?.({
        current: currentStage,
        total: totalStages,
        stage,
        message: `${message} (${currentStage}/${totalStages})`,
      });
    };

    const imported = {
      projects: 0,
      folders: 0,
      versions: 0,
      snippets: 0,
      attachments: 0,
    };

    try {
      const projects = await readJson(zip.file('projects.json'));
      updateProgress('projects', `正在解析项目 (${projects.length} 个)`);

      const folders = await readJson(zip.file('folders.json'));
      updateProgress('folders', `正在解析文件夹 (${folders.length} 个)`);

      const versions = await readJson(zip.file('versions.json'));
      updateProgress('versions', `正在解析版本 (${versions.length} 个)`);

      const snippets = await readJson(zip.file('snippets.json'));
      updateProgress('snippets', `正在解析代码片段 (${snippets.length} 个)`);

      const attachments = await this.resolveAttachments(
        await readJson(zip.file('attachments.json')),
        zip,
      );
      updateProgress('attachments', `正在解析附件 (${attachments.length} 个)`);

      const settingsFile = zip.file('settings.json');
      if (settingsFile) {
        await this.importSettings(settingsFile);
      }
      updateProgress('settings', '正在导入设置');

      // 整包一次性提交，后端在同一事务内按 mode 处理
      const result = await promptStudioApi.importAll(
        {
          folders,
          projects,
          versions,
          snippets,
          attachments,
        } as PromptStudioBulkData,
        options.mode,
      );

      imported.projects = result?.projects ?? 0;
      imported.folders = result?.folders ?? 0;
      imported.versions = result?.versions ?? 0;
      imported.snippets = result?.snippets ?? 0;
      imported.attachments = result?.attachments ?? 0;

      return {
        success: true,
        message: `导入成功！项目：${imported.projects}，文件夹：${imported.folders}，版本：${imported.versions}，代码片段：${imported.snippets}，附件：${imported.attachments}`,
        imported,
      };
    } catch (error) {
      console.error('导入失败:', error);
      return {
        success: false,
        message: `导入失败: ${error instanceof Error ? error.message : '未知错误'}`,
        imported,
      };
    }
  }

  /**
   * 整理附件：优先使用随包携带的 base64 内容，
   * 否则从 ZIP 的 attachments/<id><ext> 读二进制并转成 base64
   */
  private async resolveAttachments(
    attachments: any[],
    zip: JSZip,
  ): Promise<any[]> {
    const attachmentsFolder = zip.folder('attachments');

    return Promise.all(
      attachments.map(async (attachment: any) => {
        if (attachment?.content) {
          return { ...attachment, isMissing: false };
        }
        if (!attachmentsFolder) {
          return { ...attachment, isMissing: true };
        }
        for (const name of attachmentCandidates(attachment)) {
          const file = attachmentsFolder.file(name);
          if (!file) continue;
          try {
            const arrayBuffer = await file.async('arraybuffer');
            return {
              ...attachment,
              content: arrayBufferToBase64(arrayBuffer),
              isMissing: false,
            };
          } catch (error) {
            console.warn(`Failed to load attachment file: ${name}`, error);
          }
        }
        return { ...attachment, isMissing: true };
      }),
    );
  }

  /**
   * 导入设置数据（只恢复布局相关的本地偏好）
   */
  private async importSettings(settingsFile: JSZip.JSZipObject): Promise<void> {
    const settings = JSON.parse(await settingsFile.async('text'));

    if (settings[STORAGE_KEYS.CANVAS_RATIO] !== undefined) {
      storage.set(
        STORAGE_KEYS.CANVAS_RATIO,
        settings[STORAGE_KEYS.CANVAS_RATIO],
      );
    }
    if (settings[STORAGE_KEYS.EDITOR_HEIGHT_RATIO] !== undefined) {
      storage.set(
        STORAGE_KEYS.EDITOR_HEIGHT_RATIO,
        settings[STORAGE_KEYS.EDITOR_HEIGHT_RATIO],
      );
    }
    if (settings[STORAGE_KEYS.SIDEBAR_COLLAPSED] !== undefined) {
      storage.set(
        STORAGE_KEYS.SIDEBAR_COLLAPSED,
        settings[STORAGE_KEYS.SIDEBAR_COLLAPSED],
      );
    }
  }
}

export const importService = new ImportService();
