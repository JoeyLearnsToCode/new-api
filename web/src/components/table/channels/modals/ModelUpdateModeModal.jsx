/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useState, useEffect } from 'react';
import {
  Modal,
  Button,
  Radio,
  Typography,
  Card,
  Space,
  Divider,
} from '@douyinfe/semi-ui';
import {
  IconDeleteStroked,
  IconPlusStroked,
  IconRefresh,
} from '@douyinfe/semi-icons';

const { Text, Title } = Typography;

// 更新模式枚举
export const UPDATE_MODE = {
  REMOVE_INVALID: 'remove_invalid',
  ADD_NEW: 'add_new',
  FULL_UPDATE: 'full_update',
};

const ModelUpdateModeModal = ({ visible, onCancel, onConfirm, t }) => {
  const [selectedMode, setSelectedMode] = useState(UPDATE_MODE.REMOVE_INVALID);

  // 重置选择模式
  useEffect(() => {
    if (visible) {
      setSelectedMode(UPDATE_MODE.REMOVE_INVALID);
    }
  }, [visible]);

  // 处理失去焦点事件
  const handleModalBlur = (e) => {
    // 检查焦点是否移到了弹窗外部
    if (!e.currentTarget.contains(e.relatedTarget)) {
      onCancel();
    }
  };

  const handleConfirm = () => {
    onConfirm(selectedMode);
  };

  const modeOptions = [
    {
      value: UPDATE_MODE.REMOVE_INVALID,
      icon: <IconDeleteStroked style={{ color: '#f5222d' }} />,
      title: t('删除无效模型'),
      description: t('仅删除渠道不再支持的无效模型，不添加新模型'),
      detail: t('计划阶段只寻找、列出需要删除的无效模型。适用于清理过期模型。'),
    },
    {
      value: UPDATE_MODE.ADD_NEW,
      icon: <IconPlusStroked style={{ color: '#52c41a' }} />,
      title: t('添加新模型'),
      description: t('仅添加渠道新支持的模型，不删除现有模型'),
      detail: t(
        '计划阶段只寻找、列出尚未添加的新模型。',
      ),
    },
    {
      value: UPDATE_MODE.FULL_UPDATE,
      icon: <IconRefresh style={{ color: '#1890ff' }} />,
      title: t('完整更新'),
      description: t('同时删除无效模型和添加新模型'),
      detail: t('现有的更新模式，结合了以上两种模式。全面同步渠道模型列表。'),
    },
  ];

  return (
    <Modal
      title={t('选择模型更新模式')}
      visible={visible}
      onCancel={onCancel}
      width={600}
      maskClosable={true}
      closable={true}
      onBlur={handleModalBlur}
      footer={
        <Space>
          <Button onClick={onCancel}>{t('取消')}</Button>
          <Button type='primary' onClick={handleConfirm}>
            {t('确认')}
          </Button>
        </Space>
      }
    >
      <div style={{ padding: '8px 0' }}>
        <Text type='secondary' style={{ marginBottom: 16, display: 'block' }}>
          {t('请选择模型更新模式，不同模式将执行不同的更新策略：')}
        </Text>

        <Radio.Group
          value={selectedMode}
          onChange={(e) => setSelectedMode(e.target.value)}
          style={{ width: '100%' }}
        >
          {modeOptions.map((option) => (
            <Card
              key={option.value}
              style={{
                marginBottom: 12,
                cursor: 'pointer',
                border:
                  selectedMode === option.value
                    ? '2px solid #1890ff'
                    : '1px solid #d9d9d9',
                backgroundColor:
                  selectedMode === option.value ? '#f6ffed' : 'transparent',
                width: '100%',
              }}
              bodyStyle={{ padding: 16 }}
              onClick={() => setSelectedMode(option.value)}
            >
              <Radio value={option.value} style={{ marginBottom: 8 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  {option.icon}
                  <Text strong>{option.title}</Text>
                </div>
              </Radio>
              <div style={{ marginLeft: 24, marginTop: 4 }}>
                <Text type='secondary'>{option.description}</Text>
                <Divider style={{ margin: '8px 0' }} />
                <Text size='small' type='tertiary'>
                  {option.detail}
                </Text>
              </div>
            </Card>
          ))}
        </Radio.Group>
      </div>
    </Modal>
  );
};

export default ModelUpdateModeModal;
