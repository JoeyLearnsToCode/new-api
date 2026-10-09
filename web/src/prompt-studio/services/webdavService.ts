/**
 * WebDAV 备份与还原服务
 *
 * 与原实现的差异：数据不再来自 IndexedDB，而是
 * - 备份：`GET /api/prompt-studio/export` 取回全量数据 → 复用 exportService 的打包实现 → 浏览器直传 WebDAV
 * - 还原：从 WebDAV 下载 → 走 importService 现有的 ZIP 解析 → `POST /api/prompt-studio/import`
 *
 * 上传/下载全程在浏览器与 WebDAV 服务器之间进行，new-api 后端不参与，
 * 因此要求 WebDAV 服务端允许跨域（CORS）并接受 Authorization 头。
 */

import { createClient, type WebDAVClient } from 'webdav';
import { exportService } from './exportService';
import { importService } from './importService';
import type {
  ImportOptions,
  ImportProgressCallback,
} from '@/prompt-studio/types/import';
import { useI18nStore } from '@/prompt-studio/store/i18nStore';
import { translations } from '@/prompt-studio/i18n/locales';

export interface WebDAVConfig {
  url: string;
  username: string;
  password: string;
}

const WEBDAV_DIR = 'prompt-studio-backups';

/** 简单的翻译辅助函数（复制自上游，供抛错文案使用） */
function t(key: string): string {
  const currentLocale = useI18nStore.getState().currentLocale;
  const keys = key.split('.');
  let value: any = translations[currentLocale];
  for (const k of keys) {
    value = value?.[k];
    if (value === undefined) {
      return key;
    }
  }
  return typeof value === 'string' ? value : key;
}

export interface WebDAVBackup {
  name: string;
  path: string;
  size: number;
  lastMod: string;
}

export class WebDAVService {
  private client: WebDAVClient | null = null;
  private config: WebDAVConfig | null = null;

  /** 配置 WebDAV 客户端 */
  configure(config: WebDAVConfig): void {
    this.config = config;
    this.client = createClient(config.url, {
      username: config.username,
      password: config.password,
    });
  }

  /** 测试连接 */
  async testConnection(): Promise<boolean> {
    if (!this.client) {
      throw new Error(t('pages.settings.webdav.configureFirst'));
    }
    try {
      await this.client.exists('/');
      return true;
    } catch (error) {
      console.error('WebDAV 连接测试失败:', error);
      return false;
    }
  }

  /** 备份所有数据到 WebDAV */
  async backupToWebDAV(): Promise<void> {
    if (!this.client) {
      throw new Error(t('pages.settings.webdav.configureFirst'));
    }

    try {
      // 复用本地导出的打包实现，备份包结构与本地导出完全一致
      const blob = await exportService.buildBackupZip();

      const filename = `prompt-studio-backup-${Date.now()}.zip`;
      const remotePath = `/${WEBDAV_DIR}/${filename}`;

      // 确保目录存在
      try {
        await this.client.createDirectory(`/${WEBDAV_DIR}`);
      } catch (error) {
        // 目录可能已存在，忽略错误
      }

      const arrayBuffer = await blob.arrayBuffer();
      await this.client.putFileContents(remotePath, arrayBuffer);

      console.log(`备份成功: ${remotePath}`);
    } catch (error) {
      console.error('WebDAV 备份失败:', error);
      const errorMessage =
        error instanceof Error
          ? error.message
          : t('pages.settings.errors.unknown');
      throw new Error(
        `${t('pages.settings.webdav.backupFailed')}: ${errorMessage}`,
      );
    }
  }

  /** 列出所有备份文件 */
  async listBackups(): Promise<WebDAVBackup[]> {
    if (!this.client) {
      throw new Error(t('pages.settings.webdav.configureFirst'));
    }

    try {
      const dirPath = `/${WEBDAV_DIR}/`;
      const exists = await this.client.exists(dirPath);
      if (!exists) {
        return [];
      }

      const contents = await this.client.getDirectoryContents(dirPath);

      return (contents as any[])
        .filter(
          (item) => item.type === 'file' && item.basename.endsWith('.zip'),
        )
        .map((item) => ({
          name: item.basename,
          path: item.filename,
          size: item.size,
          lastMod: item.lastmod,
        }))
        .sort(
          (a, b) =>
            new Date(b.lastMod).getTime() - new Date(a.lastMod).getTime(),
        );
    } catch (error) {
      console.error('获取备份列表失败:', error);
      const errorMessage =
        error instanceof Error
          ? error.message
          : t('pages.settings.errors.unknown');
      throw new Error(
        `${t('pages.settings.errors.loadBackupsFailed')}: ${errorMessage}`,
      );
    }
  }

  /** 从 WebDAV 还原数据：下载后复用 ZIP 导入流程 */
  async restoreFromWebDAV(
    remotePath: string,
    options: ImportOptions,
    onProgress?: ImportProgressCallback,
  ): Promise<void> {
    if (!this.client) {
      throw new Error(t('pages.settings.webdav.configureFirst'));
    }
    const arrayBuffer = (await this.client.getFileContents(remotePath, {
      format: 'binary',
    })) as ArrayBuffer;

    const file = new File([arrayBuffer], 'backup.zip', {
      type: 'application/zip',
    });
    const result = await importService.importFromZip(file, options, onProgress);
    if (!result.success) {
      throw new Error(result.message);
    }
  }

  /** 删除远程备份 */
  async deleteBackup(remotePath: string): Promise<void> {
    if (!this.client) {
      throw new Error(t('pages.settings.webdav.configureFirst'));
    }
    try {
      await this.client.deleteFile(remotePath);
      console.log(`删除备份成功: ${remotePath}`);
    } catch (error) {
      console.error('删除备份失败:', error);
      const errorMessage =
        error instanceof Error
          ? error.message
          : t('pages.settings.errors.unknown');
      throw new Error(
        `${t('pages.settings.webdav.deleteFailed')}: ${errorMessage}`,
      );
    }
  }

  /** 获取当前配置 */
  getConfig(): WebDAVConfig | null {
    return this.config;
  }

  /** 清除配置 */
  clearConfig(): void {
    this.client = null;
    this.config = null;
  }
}

export const webdavService = new WebDAVService();
