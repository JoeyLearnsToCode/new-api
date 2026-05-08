import React, { useState, useRef } from 'react';
import { Button, Modal, TextArea, Typography, Banner } from '@douyinfe/semi-ui';
import { API, showError, showSuccess, showInfo } from '../../helpers';
import { useTranslation } from 'react-i18next';

const SettingsExportImport = ({ inputs, onRefresh }) => {
  const { t } = useTranslation();
  const [showExportModal, setShowExportModal] = useState(false);
  const [showImportModal, setShowImportModal] = useState(false);
  const [exportJson, setExportJson] = useState('');
  const [importJson, setImportJson] = useState('');
  const [importError, setImportError] = useState('');
  const [importing, setImporting] = useState(false);
  const [importResult, setImportResult] = useState(null);
  const fileInputRef = useRef(null);

  // Export: fetch all options from API and download as JSON
  const handleExport = async () => {
    try {
      const res = await API.get('/api/option/');
      const { success, message, data } = res.data;
      if (success) {
        const settings = {};
        data.forEach((item) => {
          settings[item.key] = item.value;
        });
        const jsonText = JSON.stringify({ settings }, null, 2);
        setExportJson(jsonText);
        setShowExportModal(true);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
  };

  const handleCopyExport = async () => {
    try {
      await navigator.clipboard.writeText(exportJson);
      showSuccess(t('已复制到剪贴板'));
    } catch {
      showError(t('复制失败'));
    }
  };

  const handleDownloadExport = () => {
    const now = new Date();
    const pad = (n) => String(n).padStart(2, '0');
    const timestamp = `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}_${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`;
    const filename = `newapi_settings_export_${timestamp}.json`;
    const blob = new Blob([exportJson], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  // Import: read JSON file, then PUT each setting via API
  const handleImportClick = () => {
    setImportJson('');
    setImportError('');
    setImportResult(null);
    setShowImportModal(true);
  };

  const handleFileSelect = (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (!file.name.endsWith('.json')) {
      showError(t('请选择 .json 文件'));
      return;
    }
    const reader = new FileReader();
    reader.onload = (event) => {
      const text = event.target.result;
      setImportJson(text);
      validateImportJson(text);
    };
    reader.readAsText(file);
    // Reset file input so the same file can be re-selected
    e.target.value = '';
  };

  const validateImportJson = (text) => {
    setImportResult(null);
    if (!text || !text.trim()) {
      setImportError('');
      return;
    }
    try {
      const parsed = JSON.parse(text);
      if (!parsed.settings || typeof parsed.settings !== 'object') {
        setImportError(t('JSON 格式不正确，需要 {"settings": {...}} 结构'));
        return;
      }
      setImportError('');
    } catch (e) {
      setImportError(e.message);
    }
  };

  const handleImportJsonChange = (value) => {
    setImportJson(value);
    validateImportJson(value);
  };

  const handleDoImport = async () => {
    if (importError) {
      showError(t('JSON 格式错误，请先修正'));
      return;
    }
    try {
      const parsed = JSON.parse(importJson);
      const settings = parsed.settings;
      if (!settings || typeof settings !== 'object') {
        showError(t('JSON 格式不正确'));
        return;
      }

      setImporting(true);
      const keys = Object.keys(settings);
      let successCount = 0;
      let failCount = 0;
      const failedKeys = [];

      for (const key of keys) {
        try {
          const res = await API.put('/api/option/', {
            key,
            value: settings[key],
          });
          if (res.data?.success) {
            successCount++;
          } else {
            failCount++;
            failedKeys.push({ key, reason: res.data?.message || 'unknown' });
          }
        } catch (error) {
          failCount++;
          failedKeys.push({ key, reason: error.message });
        }
      }

      setImportResult({ successCount, failCount, failedKeys });
      if (failCount === 0) {
        showSuccess(t('导入完成：成功 ${count} 项').replace('${count}', successCount));
      } else {
        showInfo(
          t('导入完成：成功 ${success} 项，失败 ${failed} 项')
            .replace('${success}', successCount)
            .replace('${failed}', failCount),
        );
      }
      onRefresh?.();
    } catch (error) {
      showError(error.message);
    } finally {
      setImporting(false);
    }
  };

  return (
    <>
      <div className='flex gap-2 mt-4'>
        <Button type='primary' theme='light' onClick={handleExport}>
          {t('导出设置')}
        </Button>
        <Button type='primary' theme='solid' onClick={handleImportClick}>
          {t('导入设置')}
        </Button>
      </div>

      {/* Export Modal */}
      <Modal
        title={t('导出设置')}
        visible={showExportModal}
        onCancel={() => setShowExportModal(false)}
        maskClosable
        closable
        closeOnEsc
        footer={null}
        centered
        width={700}
      >
        <TextArea
          value={exportJson}
          rows={20}
          style={{ fontFamily: 'monospace', fontSize: 12 }}
          readOnly
        />
        <div className='flex justify-end gap-2 mt-4'>
          <Button type='tertiary' onClick={() => setShowExportModal(false)}>
            {t('取消')}
          </Button>
          <Button type='primary' theme='light' onClick={handleCopyExport}>
            {t('复制')}
          </Button>
          <Button type='primary' theme='solid' onClick={handleDownloadExport}>
            {t('下载')}
          </Button>
        </div>
      </Modal>

      {/* Import Modal */}
      <Modal
        title={t('导入设置')}
        visible={showImportModal}
        onCancel={() => setShowImportModal(false)}
        maskClosable
        closable
        closeOnEsc
        footer={null}
        centered
        width={700}
      >
        <div className='mb-3'>
          <input
            type='file'
            accept='.json'
            onChange={handleFileSelect}
            style={{ display: 'none' }}
            ref={fileInputRef}
          />
          <Button
            type='tertiary'
            onClick={() => fileInputRef.current?.click()}
          >
            {t('选择文件')}
          </Button>
          <Typography.Text type='secondary' size='small' className='ml-2'>
            {t('或直接在下方粘贴 JSON 内容')}
          </Typography.Text>
        </div>

        {importError && (
          <Banner
            type='danger'
            description={`${t('JSON 格式错误')}: ${importError}`}
            className='mb-3'
          />
        )}

        <TextArea
          value={importJson}
          onChange={handleImportJsonChange}
          placeholder={'{"settings": {...}}'}
          rows={15}
          style={{ fontFamily: 'monospace', fontSize: 12 }}
        />

        {importResult && (
          <div className='mt-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg'>
            <Typography.Text strong>{t('导入结果')}</Typography.Text>
            <div className='mt-1'>
              <Typography.Text type='success'>
                {t('成功: ${count} 项').replace('${count}', importResult.successCount)}
              </Typography.Text>
              {importResult.failCount > 0 && (
                <Typography.Text type='danger' className='ml-4'>
                  {t('失败: ${count} 项').replace('${count}', importResult.failCount)}
                </Typography.Text>
              )}
            </div>
            {importResult.failedKeys?.length > 0 && (
              <div className='mt-2'>
                {importResult.failedKeys.map((f, i) => (
                  <Typography.Text key={i} type='danger' size='small'>
                    {f.key}: {f.reason}
                  </Typography.Text>
                ))}
              </div>
            )}
          </div>
        )}

        <div className='flex justify-end gap-2 mt-4'>
          <Button type='tertiary' onClick={() => setShowImportModal(false)}>
            {t('取消')}
          </Button>
          <Button
            type='primary'
            theme='solid'
            onClick={handleDoImport}
            disabled={!!importError || !importJson.trim() || importing}
            loading={importing}
          >
            {t('导入')}
          </Button>
        </div>
      </Modal>
    </>
  );
};

export default SettingsExportImport;
