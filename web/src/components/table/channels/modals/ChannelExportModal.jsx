import React, { useState, useEffect, useCallback } from 'react';
import { Modal, TextArea, Button, Typography, Banner } from '@douyinfe/semi-ui';
import { API, copy, showError, showSuccess } from '../../../../helpers';

const ChannelExportModal = ({ visible, handleClose, selectedChannels, enableBatchDelete, t }) => {
  const [jsonText, setJsonText] = useState('');
  const [jsonError, setJsonError] = useState('');
  const [loading, setLoading] = useState(false);

  const fetchExportData = useCallback(async () => {
    setLoading(true);
    try {
      let url = '/api/channel/export';
      if (enableBatchDelete && selectedChannels && selectedChannels.length > 0) {
        const ids = selectedChannels.map((ch) => ch.id).join(',');
        url += `?ids=${ids}`;
      }
      const res = await API.get(url);
      const { success, message, data } = res.data;
      if (success) {
        const formatted = JSON.stringify({ channels: data.channels || [] }, null, 2);
        setJsonText(formatted);
        setJsonError('');
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    } finally {
      setLoading(false);
    }
  }, [enableBatchDelete, selectedChannels]);

  useEffect(() => {
    if (visible) {
      fetchExportData();
    } else {
      setJsonText('');
      setJsonError('');
    }
  }, [visible, fetchExportData]);

  const handleJsonChange = (value) => {
    setJsonText(value);
    if (value && value.trim()) {
      try {
        JSON.parse(value);
        setJsonError('');
      } catch (e) {
        setJsonError(e.message);
      }
    } else {
      setJsonError('');
    }
  };

  const handleCopy = async () => {
    if (jsonError) {
      showError(t('JSON 格式错误，无法复制'));
      return;
    }
    const copied = await copy(jsonText);
    if (copied) {
      showSuccess(t('已复制到剪贴板'));
    } else {
      showError(t('复制失败'));
    }
  };

  const handleDownload = () => {
    if (jsonError) {
      showError(t('JSON 格式错误，无法下载'));
      return;
    }
    const now = new Date();
    const pad = (n) => String(n).padStart(2, '0');
    const timestamp = `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}_${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`;
    const filename = `newapi_channels_export_${timestamp}.json`;
    const blob = new Blob([jsonText], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  return (
    <Modal
      title={t('导出渠道')}
      visible={visible}
      onCancel={handleClose}
      maskClosable
      closable
      closeOnEsc
      footer={null}
      centered
      width={700}
      className='!rounded-lg'
    >
      {loading ? (
        <div className='flex justify-center py-8'>
          <Typography.Text>{t('正在加载...')}</Typography.Text>
        </div>
      ) : (
        <>
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
            rows={20}
            style={{ fontFamily: 'monospace', fontSize: 12 }}
          />
          <div className='flex justify-end gap-2 mt-4'>
            <Button type='tertiary' onClick={handleClose}>
              {t('取消')}
            </Button>
            <Button type='primary' theme='light' onClick={handleCopy}>
              {t('复制')}
            </Button>
            <Button type='primary' theme='solid' onClick={handleDownload}>
              {t('下载')}
            </Button>
          </div>
        </>
      )}
    </Modal>
  );
};

export default ChannelExportModal;
