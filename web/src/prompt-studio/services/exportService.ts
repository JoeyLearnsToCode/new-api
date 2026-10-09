/**
 * 数据导入导出服务
 */

import JSZip, { type JSZipGeneratorOptions } from 'jszip';
import { saveAs } from 'file-saver';
import { promptStudioApi } from '@/prompt-studio/api/client';
import { storage, STORAGE_KEYS } from '@/prompt-studio/utils/storage';
import { importService } from './importService';
import type {
  ImportOptions,
  ImportProgressCallback,
} from '@/prompt-studio/types/import';

export const zipGenOptions: JSZipGeneratorOptions<'blob'> = {
  type: 'blob',
  compression: 'DEFLATE',
  compressionOptions: { level: 5 },
};

/** base64 -> Uint8Array，用于把附件内容写回 ZIP 的二进制目录 */
const base64ToUint8Array = (base64: string): Uint8Array => {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
};

export class ExportService {
  /**
   * 导出单个项目为 JSON
   */
  async exportProjectAsJSON(projectId: string): Promise<void> {
    const data = await promptStudioApi.exportAll();
    const project = data.projects.find((p) => p.id === projectId);
    const versions = data.versions.filter((v) => v.projectId === projectId);
    const versionIds = new Set(versions.map((v) => v.id));
    const attachments = data.attachments.filter((a) =>
      versionIds.has(a.versionId),
    );

    const payload = {
      project,
      versions,
      attachments,
      exportedAt: new Date().toISOString(),
      version: '1.0',
    };

    const blob = new Blob([JSON.stringify(payload, null, 2)], {
      type: 'application/json',
    });
    saveAs(blob, `${project?.name || 'project'}_${Date.now()}.json`);
  }

  /**
   * 构建备份 ZIP
   * 本地导出与 WebDAV 备份共用同一份打包实现，两者产出的备份包结构完全一致
   */
  async buildBackupZip(): Promise<Blob> {
    const zip = new JSZip();

    // 一次性从服务端取回全部数据，前端只负责打包
    const { projects, folders, versions, snippets, attachments } =
      await promptStudioApi.exportAll();

    zip.file('projects.json', JSON.stringify(projects, null, 2));
    zip.file('folders.json', JSON.stringify(folders, null, 2));
    zip.file('versions.json', JSON.stringify(versions, null, 2));
    zip.file('snippets.json', JSON.stringify(snippets, null, 2));
    zip.file('attachments.json', JSON.stringify(attachments, null, 2));

    // 附件二进制同步写入 attachments 子目录，保持与原 prompt-studio 备份包一致的结构
    const attachmentsFolder = zip.folder('attachments');
    if (attachmentsFolder) {
      for (const attachment of attachments) {
        const content = (attachment as any).content;
        if (!content) continue;
        const fileExtension = ExportService.getFileExtension(
          attachment.fileType,
          attachment.fileName,
        );
        attachmentsFolder.file(
          `${attachment.id}${fileExtension}`,
          base64ToUint8Array(content),
        );
      }
    }

    // 导出 localStorage 中的布局偏好
    const settings: Record<string, any> = {};
    const canvasRatio = storage.get(STORAGE_KEYS.CANVAS_RATIO, null);
    const editorHeightRatio = storage.get(
      STORAGE_KEYS.EDITOR_HEIGHT_RATIO,
      null,
    );
    const sidebarCollapsed = storage.get(STORAGE_KEYS.SIDEBAR_COLLAPSED, null);
    if (canvasRatio !== null) settings[STORAGE_KEYS.CANVAS_RATIO] = canvasRatio;
    if (editorHeightRatio !== null)
      settings[STORAGE_KEYS.EDITOR_HEIGHT_RATIO] = editorHeightRatio;
    if (sidebarCollapsed !== null)
      settings[STORAGE_KEYS.SIDEBAR_COLLAPSED] = sidebarCollapsed;

    zip.file('settings.json', JSON.stringify(settings, null, 2));

    zip.file(
      'metadata.json',
      JSON.stringify(
        {
          exportedAt: new Date().toISOString(),
          version: '1.0',
          counts: {
            projects: projects.length,
            folders: folders.length,
            versions: versions.length,
            snippets: snippets.length,
            attachments: attachments.length,
          },
        },
        null,
        2,
      ),
    );

    return zip.generateAsync(zipGenOptions);
  }

  /**
   * 导出所有数据为 ZIP（下载到本地）
   */
  async exportAllAsZip(): Promise<void> {
    const blob = await this.buildBackupZip();
    saveAs(blob, `prompt-studio-backup-${Date.now()}.zip`);
  }

  /**
   * 根据 MIME 类型或原始文件名获取文件扩展名
   */
  private static getFileExtension(
    mimeType: string,
    originalFileName: string = '',
  ): string {
    // 首先尝试从原始文件名中提取扩展名
    if (originalFileName && originalFileName.includes('.')) {
      const parts = originalFileName.split('.');
      const ext = parts[parts.length - 1].toLowerCase();
      if (ext.length >= 2 && ext.length <= 5) {
        return `.${ext}`;
      }
    }

    // 如果没有原始扩展名或扩展名不符合预期，则根据 MIME 类型推断
    const mimeToExt: { [key: string]: string } = {
      'image/jpeg': '.jpg',
      'image/jpg': '.jpg',
      'image/png': '.png',
      'image/gif': '.gif',
      'image/webp': '.webp',
      'image/svg+xml': '.svg',
      'video/mp4': '.mp4',
      'video/webm': '.webm',
      'video/ogg': '.ogg',
      'video/quicktime': '.mov',
      'application/pdf': '.pdf',
      'text/plain': '.txt',
      'text/html': '.html',
      'application/json': '.json',
    };

    return mimeToExt[mimeType] || '.bin';
  }

  /**
   * 导入 JSON 数据（使用新的导入服务）
   */
  async importFromJSON(
    file: File,
    options: ImportOptions,
    onProgress?: ImportProgressCallback,
  ): Promise<void> {
    const text = await file.text();
    const data = JSON.parse(text);

    // 使用新的导入服务，将 JSON 数据转换为 ZIP 格式
    const zip = new JSZip();

    if (data.project) {
      zip.file('projects.json', JSON.stringify([data.project], null, 2));
    }
    if (data.versions) {
      // 确保版本数据不包含 normalizedContent 字段
      const cleanVersions = data.versions.map(
        ({ normalizedContent, ...version }: any) => version,
      );
      zip.file('versions.json', JSON.stringify(cleanVersions, null, 2));
    }
    if (data.attachments) {
      zip.file('attachments.json', JSON.stringify(data.attachments, null, 2));
    }

    const blob = await zip.generateAsync({ type: 'blob' });
    const zipFile = new File([blob], 'import.zip', { type: 'application/zip' });

    const result = await importService.importFromZip(
      zipFile,
      options,
      onProgress,
    );
    if (!result.success) {
      throw new Error(result.message);
    }
  }
  /**
   * 导入 ZIP 备份（使用新的导入服务）
   */
  async importFromZip(
    file: File,
    options: ImportOptions,
    onProgress?: ImportProgressCallback,
  ): Promise<void> {
    const result = await importService.importFromZip(file, options, onProgress);
    if (!result.success) {
      throw new Error(result.message);
    }
  }
}

export const exportService = new ExportService();
