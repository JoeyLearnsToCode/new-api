/**
 * 附件管理服务
 *
 * 原实现把 Blob 直接存进 IndexedDB，这里改为上传到 new-api 后端。
 * 附件内容需要带 New-Api-User 头才能拉取，不能直接用 <img src> 指向接口地址，
 * 因此统一在这里换取 Object URL 并缓存，供预览与下载复用。
 */

import { promptStudioApi } from '@/prompt-studio/api/client';
import type { Attachment } from '@/prompt-studio/models/Attachment';

/** id -> Object URL */
const previewUrlCache = new Map<string, string>();

const isPreviewable = (attachment: Attachment): boolean =>
  !attachment.isMissing && !!attachment.fileType?.startsWith('image/');

export class AttachmentManager {
  /**
   * 批量上传附件（一次请求提交多个文件）
   */
  async uploadAttachments(versionId: string, files: File[]): Promise<string[]> {
    if (files.length === 0) return [];
    const attachments = await promptStudioApi.uploadAttachments(
      versionId,
      files,
    );
    return attachments.map((attachment) => attachment.id);
  }

  /**
   * 上传单个附件
   */
  async uploadAttachment(versionId: string, file: File): Promise<string> {
    const [id] = await this.uploadAttachments(versionId, [file]);
    return id;
  }

  /**
   * 获取版本的所有附件
   */
  async getAttachmentsByVersion(versionId: string): Promise<Attachment[]> {
    return await promptStudioApi.listAttachments(versionId);
  }

  /**
   * 删除附件
   */
  async deleteAttachment(id: string): Promise<void> {
    await promptStudioApi.deleteAttachment(id);
    const cached = previewUrlCache.get(id);
    if (cached) {
      URL.revokeObjectURL(cached);
      previewUrlCache.delete(id);
    }
  }

  /**
   * 下载附件
   * 传入附件对象时能拿到原始文件名，传 id 时用 id 兜底
   */
  async downloadAttachment(attachment: string | Attachment): Promise<void> {
    const target =
      typeof attachment === 'string' ? { id: attachment } : attachment;
    if ('isMissing' in target && target.isMissing) {
      throw new Error('附件文件已丢失或损坏');
    }

    const url = await this.ensurePreviewUrl(target.id);
    if (!url) {
      throw new Error('附件不存在或数据不完整');
    }

    const a = document.createElement('a');
    a.href = url;
    a.download = ('fileName' in target && target.fileName) || target.id;
    a.click();
  }

  /**
   * 获取附件预览 URL（同步）
   * 需要先调用 preloadPreviewUrls 预取，未就绪时返回 null
   */
  getPreviewUrl(attachment: Attachment): string | null {
    if (!isPreviewable(attachment)) {
      return null;
    }
    return previewUrlCache.get(attachment.id) ?? null;
  }

  /**
   * 批量预取附件的 Object URL，供同步渲染 <img src> 使用
   */
  async preloadPreviewUrls(attachments: Attachment[]): Promise<void> {
    await Promise.all(
      attachments
        .filter(isPreviewable)
        .map((attachment) => this.ensurePreviewUrl(attachment.id)),
    );
  }

  /**
   * 拉取并缓存附件的 Object URL
   */
  private async ensurePreviewUrl(id: string): Promise<string | null> {
    const cached = previewUrlCache.get(id);
    if (cached) return cached;
    try {
      const blob = await promptStudioApi.fetchAttachmentBlob(id);
      const url = URL.createObjectURL(blob);
      previewUrlCache.set(id, url);
      return url;
    } catch (error) {
      console.error('Failed to load attachment', id, error);
      return null;
    }
  }
}

export const attachmentManager = new AttachmentManager();
