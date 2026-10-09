import React from 'react';
import { Modal } from './Modal';
import { useTranslation } from '@/prompt-studio/i18n/I18nContext';

export interface ImportModeDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (mode: 'merge' | 'overwrite') => void;
}

export const ImportModeDialog: React.FC<ImportModeDialogProps> = ({
  open,
  onClose,
  onConfirm,
}) => {
  const t = useTranslation();

  const handleMerge = () => {
    onConfirm('merge');
  };

  const handleOverwrite = () => {
    onConfirm('overwrite');
  };

  if (!open) return null;

  return (
    <Modal
      isOpen={open}
      onClose={onClose}
      title={t('pages.settings.local.importMode.selectMode')}
    >
      <div className='space-y-4'>
        <div className='space-y-3'>
          {/* 这里保留大按钮样式，因为 MinimalButton 不适合承载复杂的内部布局 */}
          <button
            onClick={handleMerge}
            className='w-full text-left p-4 border border-border dark:border-border-dark rounded-lg hover:bg-surface-variant dark:hover:bg-surface-variantDark focus:outline-none focus:ring-2 focus:ring-primary transition-colors text-surface-onSurface dark:text-surface-onSurfaceDark'
            autoFocus
          >
            <div className='flex items-start space-x-3'>
              <div className='flex-1'>
                <h3 className='font-semibold'>
                  {t('pages.settings.local.importMode.mergeMode')}
                </h3>
                <p className='text-sm text-surface-onVariant dark:text-surface-onVariantDark mt-1'>
                  {t('pages.settings.local.importMode.mergeModeDescription')}
                </p>
              </div>
            </div>
          </button>

          <button
            onClick={handleOverwrite}
            className='w-full text-left p-4 border border-border dark:border-border-dark rounded-lg hover:bg-surface-variant dark:hover:bg-surface-variantDark focus:outline-none focus:ring-2 focus:ring-error transition-colors text-surface-onSurface dark:text-surface-onSurfaceDark'
          >
            <div className='flex items-start space-x-3'>
              <div className='flex-1'>
                <h3 className='font-semibold'>
                  {t('pages.settings.local.importMode.overwriteMode')}
                </h3>
                <p className='text-sm text-surface-onVariant dark:text-surface-onVariantDark mt-1'>
                  {t(
                    'pages.settings.local.importMode.overwriteModeDescription',
                  )}
                </p>
              </div>
            </div>
          </button>
        </div>
      </div>
    </Modal>
  );
};
