/**
 * 数据导入导出服务
 */

import JSZip, { type JSZipGeneratorOptions } from 'jszip';
import { saveAs } from 'file-saver';
import { promptStudioApi } from '@/prompt-studio/api/client';
import { storage, STORAGE_KEYS } from '@/prompt-studio/utils/storage';
import { importService } from './importService';
import type { Attachment } from '@/prompt-studio/models/Attachment';
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

/**
 * 生成写入 attachments.json 的附件元数据
 *
 * - 二进制统一放在 ZIP 的 `attachments/<id><ext>` 里，元数据不再重复携带 base64。
 *   （`GET /export` 返回的附件自带 content，早期实现把它原样写出，
 *   导致同一张图在包里存了两份，且 base64 段几乎压不动，包体积接近翻倍）
 * - `hasBlob` 沿用上游备份格式的语义：该附件是否带有可用的二进制内容。
 *   上游在 IndexedDB 下判断的是 blob 是否存在，这里等价于 content 是否非空
 */
const toAttachmentMetadata = (attachment: Attachment): Attachment => {
  const { content, ...meta } = attachment;
  return { ...meta, hasBlob: !!content };
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
    // 单项目 JSON 是自包含的：没有 attachments/ 旁挂目录，二进制只能内联，
    // 因此这里保留 content（与上游不同，上游的 JSON 导出会丢附件）
    const attachments = data.attachments
      .filter((a) => versionIds.has(a.versionId))
      .map((attachment) => ({
        ...attachment,
        hasBlob: !!attachment.content,
      }));

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

    // 附件的 base64 只用于写下面的二进制目录，元数据里不再重复携带
    const attachmentMetadata = attachments.map(toAttachmentMetadata);

    zip.file('projects.json', JSON.stringify(projects, null, 2));
    zip.file('folders.json', JSON.stringify(folders, null, 2));
    zip.file('versions.json', JSON.stringify(versions, null, 2));
    zip.file('snippets.json', JSON.stringify(snippets, null, 2));
    zip.file('attachments.json', JSON.stringify(attachmentMetadata, null, 2));

    // 附件二进制同步写入 attachments 子目录，保持与原 prompt-studio 备份包一致的结构
    const attachmentsFolder = zip.folder('attachments');
    if (attachmentsFolder) {
      for (const attachment of attachments) {
        if (!attachment.content) continue;
        const fileExtension = ExportService.getFileExtension(
          attachment.fileType,
          attachment.fileName,
        );
        attachmentsFolder.file(
          `${attachment.id}${fileExtension}`,
          base64ToUint8Array(attachment.content),
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
