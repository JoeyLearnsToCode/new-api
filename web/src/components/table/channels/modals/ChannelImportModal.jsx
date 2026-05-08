import React, { useState, useCallback } from 'react';
import {
  Modal,
  TextArea,
  Button,
  Typography,
  Banner,
  RadioGroup,
  Radio,
  Banner as SemiBanner,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess, showInfo } from '../../../../helpers';

const ChannelImportModal = ({ visible, handleClose, onRefresh, t }) => {
  const [jsonText, setJsonText] = useState('');
  const [jsonError, setJsonError] = useState('');
  const [mode, setMode] = useState('create_only');
  const [preview, setPreview] = useState(null);
  const [importing, setImporting] = useState(false);
  const [result, setResult] = useState(null);

  const resetState = () => {
    setJsonText('');
    setJsonError('');
    setPreview(null);
    setResult(null);
    setMode('create_only');
  };

  const handleCloseModal = () => {
    resetState();
    handleClose();
  };

  const handleJsonChange = (value) => {
    setJsonText(value);
    setResult(null);
    if (value && value.trim()) {
      try {
        const parsed = JSON.parse(value);
        if (!parsed.channels || !Array.isArray(parsed.channels)) {
          setJsonError(t('JSON 格式不正确，需要 {"channels": [...]} 结构'));
          setPreview(null);
          return;
        }
        setJsonError('');
        // Local preview
        const ids = parsed.channels.map((ch) => ch.id).filter((id) => id !== undefined);
        setPreview({
          total: parsed.channels.length,
          ids,
        });
      } catch (e) {
        setJsonError(e.message);
        setPreview(null);
      }
    } else {
      setJsonError('');
      setPreview(null);
    }
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
      setJsonText(text);
      handleJsonChange(text);
    };
    reader.readAsText(file);
  };

  const handlePreview = async () => {
    if (!preview || preview.ids.length === 0) return;
    try {
      const res = await API.post('/api/channel/import/preview', {
        ids: preview.ids,
      });
      const { success, data } = res.data;
      if (success) {
        setPreview((prev) => ({
          ...prev,
          existingIds: data.existing_ids || [],
        }));
      }
    } catch (error) {
      showError(error.message);
    }
  };

  const handleImport = async () => {
    if (jsonError) {
      showError(t('JSON 格式错误，请先修正'));
      return;
    }
    try {
      const parsed = JSON.parse(jsonText);
      setImporting(true);
      const res = await API.post('/api/channel/import', {
        channels: parsed.channels,
        mode,
      });
      const { success, message, data } = res.data;
      if (success) {
        setResult(data);
        const total = (data.created?.length || 0) + (data.updated?.length || 0) + (data.skipped?.length || 0) + (data.failed?.length || 0);
        showSuccess(
          t('导入完成：成功 ${success} 条，跳过 ${skipped} 条，失败 ${failed} 条')
            .replace('${success}', (data.created?.length || 0) + (data.updated?.length || 0))
            .replace('${skipped}', data.skipped?.length || 0)
            .replace('${failed}', data.failed?.length || 0),
        );
        onRefresh?.();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    } finally {
      setImporting(false);
    }
  };

  const existingCount = preview?.existingIds?.length ?? 0;
  const newCount = preview ? preview.total - existingCount : 0;

  return (
    <Modal
      title={t('导入渠道')}
      visible={visible}
      onCancel={handleCloseModal}
      maskClosable
      closable
      closeOnEsc
      footer={null}
      centered
      width={700}
      className='!rounded-lg'
    >
      {/* Mode selection */}
      <div className='mb-4'>
        <Typography.Text strong>{t('导入模式')}</Typography.Text>
        <RadioGroup
          type='button'
          value={mode}
          onChange={(e) => setMode(e.target.value)}
          className='ml-3'
        >
          <Radio value='create_only'>{t('仅新增')}</Radio>
          <Radio value='overwrite'>{t('覆盖')}</Radio>
          <Radio value='upsert'>{t('智能合并')}</Radio>
        </RadioGroup>
        <Typography.Text type='secondary' size='small' className='ml-3'>
          {mode === 'create_only'
            ? t('ID 已存在的渠道将被跳过')
            : mode === 'overwrite'
              ? t('ID 已存在的渠道将被完整替换')
              : t('ID和名称都相同则覆盖，否则追加')}
        </Typography.Text>
      </div>

      {/* File upload */}
      <div className='mb-3'>
        <input
          type='file'
          accept='.json'
          onChange={handleFileSelect}
          style={{ display: 'none' }}
          id='channel-import-file'
        />
        <Button
          type='tertiary'
          onClick={() => document.getElementById('channel-import-file')?.click()}
        >
          {t('选择文件')}
        </Button>
        <Typography.Text type='secondary' size='small' className='ml-2'>
          {t('或直接在下方粘贴 JSON 内容')}
        </Typography.Text>
      </div>

      {/* JSON input */}
      {jsonError && (
        <Banner
          type='danger'
          description={`${t('JSON 格式错误')}: ${jsonError}`}
          className='mb-3'
        />
      )}
      <TextArea
        value={jsonText}
        onChange={handleJsonChange}
        placeholder={'{"channels": [...]}'}
        rows={12}
        style={{ fontFamily: 'monospace', fontSize: 12 }}
      />

      {/* Preview */}
      {preview && !jsonError && (
        <div className='mt-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg'>
          <Typography.Text strong>{t('预览')}</Typography.Text>
          <div className='mt-1'>
            <Typography.Text>
              {t('识别到 ${count} 条渠道').replace('${count}', preview.total)}
            </Typography.Text>
          </div>
          {preview.existingIds ? (
            <>
              <div>
                <Typography.Text type='success'>
                  {t('新增: ${count} 条').replace('${count}', newCount)}
                </Typography.Text>
                <Typography.Text type='warning' className='ml-4'>
                  {t('已存在: ${count} 条').replace('${count}', existingCount)}
                </Typography.Text>
              </div>
            </>
          ) : (
            <div className='mt-1'>
              <Button
                type='tertiary'
                size='small'
                onClick={handlePreview}
              >
                {t('检查已存在的渠道')}
              </Button>
            </div>
          )}
        </div>
      )}

      {/* Import result */}
      {result && (
        <div className='mt-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg'>
          <Typography.Text strong>{t('导入结果')}</Typography.Text>
          <div className='mt-1 space-y-1'>
            {result.created?.length > 0 && (
              <Typography.Text type='success'>
                {t('新增: ${count} 条 (ID: ${ids})')
                  .replace('${count}', result.created.length)
                  .replace('${ids}', result.created.join(', '))}
              </Typography.Text>
            )}
            {result.updated?.length > 0 && (
              <Typography.Text type='warning'>
                {t('覆盖更新: ${count} 条 (ID: ${ids})')
                  .replace('${count}', result.updated.length)
                  .replace('${ids}', result.updated.join(', '))}
              </Typography.Text>
            )}
            {result.skipped?.length > 0 && (
              <Typography.Text type='secondary'>
                {t('跳过: ${count} 条 (ID: ${ids})')
                  .replace('${count}', result.skipped.length)
                  .replace('${ids}', result.skipped.join(', '))}
              </Typography.Text>
            )}
            {result.failed?.length > 0 && (
              <div>
                <Typography.Text type='danger'>
                  {t('失败: ${count} 条').replace('${count}', result.failed.length)}
                </Typography.Text>
                <div className='ml-4 mt-1'>
                  {result.failed.map((f, i) => (
                    <Typography.Text key={i} type='danger' size='small'>
                      ID {f.id} ({f.name}): {f.reason}
                    </Typography.Text>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Actions */}
      <div className='flex justify-end gap-2 mt-4'>
        <Button type='tertiary' onClick={handleCloseModal}>
          {t('取消')}
        </Button>
        <Button
          type='primary'
          theme='solid'
          onClick={handleImport}
          disabled={!!jsonError || !jsonText.trim() || importing}
          loading={importing}
        >
          {t('导入')}
        </Button>
      </div>
    </Modal>
  );
};

export default ChannelImportModal;
