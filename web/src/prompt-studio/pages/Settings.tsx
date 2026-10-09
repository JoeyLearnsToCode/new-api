/**
 * 提示词工坊设置页
 *
 * 数据已在服务端持久化，原 WebDAV 备份/恢复卡片随之移除：
 * 备份能力由服务端数据库与这里的 ZIP 导入导出承担。
 */

import React, { useState } from 'react';
import { exportService } from '@/prompt-studio/services/exportService';
import { useProjectStore } from '@/prompt-studio/store/projectStore';
import { MinimalButton } from '@/prompt-studio/components/common/MinimalButton';
import { ImportModeDialog } from '@/prompt-studio/components/common/ImportModeDialog';
import { Icons } from '@/prompt-studio/components/icons/Icons';
import { useTranslation } from '@/prompt-studio/i18n/I18nContext';

const Settings: React.FC = () => {
  const t = useTranslation();
  const { loadFolders, loadProjects } = useProjectStore();
  const [showImportModeDialog, setShowImportModeDialog] = useState(false);
  const [pendingImportFile, setPendingImportFile] = useState<File | null>(null);

  const handleExportClick = async () => {
    try {
      await exportService.exportAllAsZip();
    } catch (error) {
      alert(
        `${t('pages.settings.local.exportFailed')}: ${
          error instanceof Error
            ? error.message
            : t('pages.settings.errors.unknown')
        }`,
      );
    }
  };

  const fileInputRef = React.useRef<HTMLInputElement>(null);

  const handleImportClick = () => {
    fileInputRef.current?.click();
  };

  const handleImportFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // 设置待处理的文件并显示模式选择对话框
    setPendingImportFile(file);
    setShowImportModeDialog(true);
  };

  const handleImportWithMode = async (mode: 'merge' | 'overwrite') => {
    if (!pendingImportFile) return;
    try {
      if (pendingImportFile.name.endsWith('.zip')) {
        await exportService.importFromZip(pendingImportFile, { mode });
      } else if (pendingImportFile.name.endsWith('.json')) {
        await exportService.importFromJSON(pendingImportFile, { mode });
      } else {
        alert(t('pages.settings.local.unsupportedFormat'));
        return;
      }

      // 刷新数据而不是重新加载页面
      await loadFolders();
      await loadProjects();

      alert(t('pages.settings.local.importSuccess'));
    } catch (error) {
      alert(
        `${t('pages.settings.local.importFailed')}: ${
          error instanceof Error
            ? error.message
            : t('pages.settings.errors.unknown')
        }`,
      );
    } finally {
      // 清理状态
      setShowImportModeDialog(false);
      setPendingImportFile(null);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const handleCancelImportMode = () => {
    setShowImportModeDialog(false);
    setPendingImportFile(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  return (
    <div className='flex flex-col h-full bg-background dark:bg-background-dark text-surface-onSurface overflow-hidden'>
      {/* 头部 */}
      <header className='h-12 bg-primary text-white flex items-center justify-between px-6 shrink-0 shadow-md z-20'>
        <div className='flex items-center gap-2'>
          <Icons.Settings className='text-2xl' />
          <h1 className='text-lg font-bold tracking-wide'>
            {t('pages.settings.title')}
          </h1>
        </div>
        <MinimalButton
          variant='ghost'
          onClick={() => (window.location.hash = '#/')}
          className='h-9 w-9 !text-white/90 !hover:text-white hover:bg-white/10'
          title={t('common.back')}
        >
          <Icons.ArrowLeft className='h-6 w-6' />
        </MinimalButton>
      </header>

      {/* 主内容区 */}
      <div className='flex-1 flex overflow-hidden p-2 gap-2 justify-center'>
        <main className='flex-1 flex flex-col overflow-y-auto max-w-4xl p-4 md:p-6 gap-6'>
          {/* 本地导入导出卡片 */}
          <section className='bg-surface dark:bg-surface-dark rounded-xl shadow-sm border border-border dark:border-border-dark p-6'>
            <div className='flex items-start gap-4 mb-6'>
              <div className='p-3 bg-primary/10 rounded-lg text-primary shrink-0'>
                <span className='material-symbols-outlined text-2xl'>
                  folder_zip
                </span>
              </div>
              <div>
                <h2 className='text-lg font-bold text-surface-onSurface dark:text-surface-onSurfaceDark'>
                  {t('pages.settings.local.title')}
                </h2>
                <p className='text-sm text-surface-onVariant dark:text-surface-onVariantDark mt-1'>
                  {t('pages.settings.local.description')}
                </p>
              </div>
            </div>
            <div className='flex gap-3 w-full sm:w-auto justify-end'>
              <MinimalButton
                variant='default'
                onClick={handleExportClick}
                className='px-4 py-2.5 text-sm gap-2'
              >
                <Icons.Download size={18} />
                {t('pages.settings.local.exportZip')}
              </MinimalButton>
              <MinimalButton
                variant='default'
                onClick={handleImportClick}
                className='px-4 py-2.5 text-sm gap-2'
              >
                <Icons.Upload size={18} />
                {t('pages.settings.local.importZip')}
              </MinimalButton>
              <input
                ref={fileInputRef}
                type='file'
                accept='.zip'
                className='hidden'
                onChange={handleImportFile}
              />
            </div>
          </section>
        </main>
      </div>

      {/* 导入模式选择对话框 */}
      <ImportModeDialog
        open={showImportModeDialog}
        onClose={handleCancelImportMode}
        onConfirm={handleImportWithMode}
      />
    </div>
  );
};

export default Settings;
