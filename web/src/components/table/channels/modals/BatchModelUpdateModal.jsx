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

import React, { useState, useRef, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Modal,
  Button,
  Progress,
  Typography,
  Card,
  List,
  Tag,
  Space,
  Spin,
  Divider,
  Collapse,
  Empty,
  Toast,
} from '@douyinfe/semi-ui';
import {
  IconPlay,
  IconRefresh,
  IconClose,
  IconCheckCircleStroked,
  IconAlertTriangle,
  IconChevronDown,
  IconChevronUp,
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess, showInfo } from '../../../../helpers';

const { Text, Title } = Typography;

// 任务状态枚举
const TASK_STATUS = {
  PLANNING: 'planning',
  PLAN_REVIEW: 'plan_review',
  EXECUTING: 'executing',
  COMPLETED: 'completed',
  STOPPED: 'stopped',
  ERROR: 'error',
};

// 渠道状态枚举
const CHANNEL_STATUS = {
  PENDING: 'pending',
  PLANNING: 'planning',
  PLAN_READY: 'plan_ready',
  PLAN_FAILED: 'plan_failed',
  EXECUTING: 'executing',
  SUCCESS: 'success',
  FAILED: 'failed',
};

// 模型操作类型
const MODEL_ACTION = {
  ADD: 'add',
  REMOVE: 'remove',
};

const BatchModelUpdateModal = ({
  visible,
  onCancel,
  selectedChannels = [],
  onRefresh,
  updateMode = 'full_update', // 新增更新模式参数
  t,
}) => {
  // 任务状态
  const [taskStatus, setTaskStatus] = useState(TASK_STATUS.PLANNING);
  const [taskProgress, setTaskProgress] = useState({ current: 0, total: 1 });

  // 渠道状态和计划
  const [channelStates, setChannelStates] = useState({});
  const [channelPlans, setChannelPlans] = useState({});

  // 任务控制
  const [isStoppable, setIsStoppable] = useState(false);
  const shouldStopRef = useRef(false);

  // UI状态
  const [expandedChannels, setExpandedChannels] = useState(new Set());
  const [taskSummary, setTaskSummary] = useState(null);
  const [planOverrides, setPlanOverrides] = useState({});

  // 计算安全的进度百分比
  const calculateProgressPercent = (current, total) => {
    if (isNaN(current) || isNaN(total) || total === 0) return 0;
    return Math.min(100, Math.max(0, Math.round((current / total) * 100)));
  };

  // 执行计划阶段的函数 - 使用useCallback避免依赖问题
  const executePlanningPhaseInternal = useCallback(async (channels) => {
    if (!channels || channels.length === 0) {
      setTaskStatus(TASK_STATUS.ERROR);
      return;
    }

    setTaskStatus(TASK_STATUS.PLANNING);
    setIsStoppable(true);

    let completedCount = 0;
    const totalCount = channels.length;

    for (const channel of channels) {
      if (shouldStopRef.current) break;

      // 更新渠道状态为规划中
      setChannelStates((prev) => ({
        ...prev,
        [channel.id]: {
          ...prev[channel.id],
          status: CHANNEL_STATUS.PLANNING,
        },
      }));

      try {
        // 获取可用模型
        const availableModels = await fetchChannelModels(channel);
        const currentModels = channel.models
          ? channel.models.split(',').filter((m) => m.trim())
          : [];
        const modelMapping = parseModelMapping(channel.model_mapping);

        // 生成计划
        const plan = generateUpdatePlan(
          currentModels,
          availableModels,
          modelMapping,
        );

        // 检查映射兼容性
        const incompatibleMappings = checkModelMappingCompatibility(
          currentModels,
          availableModels,
          modelMapping,
        );

        // 更新渠道状态
        setChannelStates((prev) => ({
          ...prev,
          [channel.id]: {
            ...prev[channel.id],
            status: CHANNEL_STATUS.PLAN_READY,
            availableModels,
            currentModels,
            plan,
            incompatibleMappings,
            model_mapping: channel.model_mapping, // 保存映射信息用于UI显示
          },
        }));

        setChannelPlans((prev) => ({
          ...prev,
          [channel.id]: plan,
        }));
        setPlanOverrides((prev) => ({
          ...prev,
          [channel.id]: {
            add: plan.add.reduce(
              (acc, model) => ({ ...acc, [model]: 'active' }),
              {},
            ),
            remove: plan.remove.reduce(
              (acc, model) => ({ ...acc, [model]: 'active' }),
              {},
            ),
          },
        }));
      } catch (error) {
        // 规划失败
        setChannelStates((prev) => ({
          ...prev,
          [channel.id]: {
            ...prev[channel.id],
            status: CHANNEL_STATUS.PLAN_FAILED,
            error: error.message,
          },
        }));
      }

      completedCount++;
      setTaskProgress({ current: completedCount, total: totalCount });
    }

    if (!shouldStopRef.current) {
      setTaskStatus(TASK_STATUS.PLAN_REVIEW);
    } else {
      setTaskStatus(TASK_STATUS.STOPPED);
    }
    setIsStoppable(false);
  }, []);

  // 初始化渠道状态
  useEffect(() => {
    if (visible && selectedChannels && selectedChannels.length > 0) {
      const initialStates = {};
      selectedChannels.forEach((channel) => {
        initialStates[channel.id] = {
          ...channel,
          status: CHANNEL_STATUS.PENDING,
          availableModels: [],
          currentModels: [],
          plan: { add: [], remove: [] },
          error: null,
        };
      });
      setChannelStates(initialStates);
      setChannelPlans({});
      setTaskStatus(TASK_STATUS.PLANNING);
      setTaskProgress({
        current: 0,
        total: Math.max(selectedChannels.length, 1),
      });
      setTaskSummary(null);
      shouldStopRef.current = false;

      // 启动计划阶段
      setTimeout(() => {
        executePlanningPhaseInternal(selectedChannels).catch((error) => {
          setTaskStatus(TASK_STATUS.ERROR);
        });
      }, 100);
    }
  }, [visible, selectedChannels, executePlanningPhaseInternal]);

  // 获取渠道可用模型
  const fetchChannelModels = async (channel) => {

    try {
      const response = await API.get(`/api/channel/fetch_models/${channel.id}`);

      if (response.data.success) {
        const models = response.data.data || [];

        return models;
      } else {
        throw new Error(response.data.message || t('获取模型失败'));
      }
    } catch (error) {

      throw new Error(error.response?.data?.message || error.message);
    }
  };

  // 解析模型映射，处理映射关系
  const parseModelMapping = (modelMappingStr) => {
    if (!modelMappingStr) return {};
    try {
      return JSON.parse(modelMappingStr);
    } catch {
      return {};
    }
  };

  // 检查模型映射兼容性
  const checkModelMappingCompatibility = (
    currentModels,
    availableModels,
    modelMapping,
  ) => {
    const availableModelSet = new Set(availableModels);
    const mappingEntries = Object.entries(modelMapping);
    const incompatibleMappings = [];

    // 检查每个映射关系
    mappingEntries.forEach(([mappedName, originalName]) => {
      if (
        currentModels.includes(mappedName) &&
        !availableModelSet.has(originalName)
      ) {
        incompatibleMappings.push({
          mappedModel: mappedName,
          targetModel: originalName,
          mappedAvailable: availableModelSet.has(mappedName),
        });
      }
    });

    return incompatibleMappings;
  };

  // 生成更新计划 - 兼容模型映射和不同更新模式
  const generateUpdatePlan = (currentModels, availableModels, modelMapping) => {
    const currentModelSet = new Set(currentModels);
    const availableModelSet = new Set(availableModels);
    const mappingEntries = Object.entries(modelMapping);



    const plan = { add: [], remove: [] };

    // 创建一个集合来跟踪被映射使用的底层模型
    const usedByMappingSet = new Set();
    mappingEntries.forEach(([mappedName, originalName]) => {
      if (currentModelSet.has(mappedName)) {
        usedByMappingSet.add(originalName);
      }
    });

    // 根据更新模式决定是否计算需要添加的模型
    if (updateMode === 'add_new' || updateMode === 'full_update') {
      availableModels.forEach((model) => {
        // 如果模型不在当前列表中，且不是被映射使用的底层模型，则添加
        if (!currentModelSet.has(model) && !usedByMappingSet.has(model)) {
          plan.add.push(model);
        }
      });
    }

    // 根据更新模式决定是否计算需要删除的模型
    if (updateMode === 'remove_invalid' || updateMode === 'full_update') {
      currentModels.forEach((model) => {
        let shouldRemove = false;

        // 检查是否是映射模型
        const mappingEntry = mappingEntries.find(
          ([mappedName]) => mappedName === model,
        );
        if (mappingEntry) {
          const [mappedName, originalName] = mappingEntry;
          // 如果映射的底层模型不在可用列表中，则映射模型应删除
          if (!availableModelSet.has(originalName)) {
            if (!availableModelSet.has(mappedName)) {
              shouldRemove = true;

            } else {

            }
          } else {

          }
        } else {
          // 非映射模型：检查是否直接在可用列表中
          if (!availableModelSet.has(model)) {
            shouldRemove = true;

          }
        }

        if (shouldRemove && !plan.remove.includes(model)) {
          plan.remove.push(model);
        }
      });
    }


    return plan;
  };

  // 执行计划阶段
  const executePlanningPhase = async () => {
    return executePlanningPhaseInternal(selectedChannels);
  };

  // 执行实施阶段
  const executeImplementationPhase = async () => {
    setTaskStatus(TASK_STATUS.EXECUTING);
    setIsStoppable(true);

    const channelsToUpdate = Object.values(channelStates).filter(
      (channel) => channel.status === CHANNEL_STATUS.PLAN_READY,
    );

    if (channelsToUpdate.length === 0) {
      setTaskStatus(TASK_STATUS.COMPLETED);
      setTaskSummary({ success: 0, failed: 0, details: [] });
      return;
    }

    let completedCount = 0;
    const totalCount = channelsToUpdate.length;
    setTaskProgress({ current: 0, total: totalCount });

    const summary = {
      success: 0,
      failed: 0,
      details: [],
    };

    for (const channel of channelsToUpdate) {
      if (shouldStopRef.current) break;

      // 更新渠道状态为执行中
      setChannelStates((prev) => ({
        ...prev,
        [channel.id]: {
          ...prev[channel.id],
          status: CHANNEL_STATUS.EXECUTING,
        },
      }));

      try {
        const plan = channelPlans[channel.id];

        // 根据更新模式判断是否需要更新
        const shouldExecuteAdd =
          updateMode === 'add_new' || updateMode === 'full_update';
        const shouldExecuteRemove =
          updateMode === 'remove_invalid' || updateMode === 'full_update';

        const hasAddOperations = shouldExecuteAdd && plan.add.length > 0;
        const hasRemoveOperations =
          shouldExecuteRemove && plan.remove.length > 0;

        if (!plan || (!hasAddOperations && !hasRemoveOperations)) {
          // 无需更新
          setChannelStates((prev) => ({
            ...prev,
            [channel.id]: {
              ...prev[channel.id],
              status: CHANNEL_STATUS.SUCCESS,
            },
          }));

          summary.success++;
          summary.details.push({
            channelId: channel.id,
            channelName: channel.name,
            success: true,
            addedCount: 0,
            removedCount: 0,
            addedModels: [],
            removedModels: [],
          });
        } else {
          // 根据更新模式执行相应的模型更新操作
          const shouldExecuteAdd =
            updateMode === 'add_new' || updateMode === 'full_update';
          const shouldExecuteRemove =
            updateMode === 'remove_invalid' || updateMode === 'full_update';

          // 根据模式过滤要执行的操作
          const activeAdds = shouldExecuteAdd
            ? plan.add.filter(
                (model) =>
                  planOverrides[channel.id]?.add?.[model] !== 'inactive',
              )
            : [];

          const activeRemoves = shouldExecuteRemove
            ? new Set(
                plan.remove.filter(
                  (model) =>
                    planOverrides[channel.id]?.remove?.[model] !== 'inactive',
                ),
              )
            : new Set();

          // 构建新的模型列表
          const newModels = [
            ...channel.currentModels.filter(
              (model) => !activeRemoves.has(model),
            ),
            ...activeAdds,
          ];

          const updateData = {
            id: channel.id,
            models: newModels.join(','),
          };

          const response = await API.put('/api/channel/', updateData);

          if (response.data.success) {
            setChannelStates((prev) => ({
              ...prev,
              [channel.id]: {
                ...prev[channel.id],
                status: CHANNEL_STATUS.SUCCESS,
              },
            }));

            summary.success++;
            summary.details.push({
              channelId: channel.id,
              channelName: channel.name,
              success: true,
              addedCount: activeAdds.length,
              removedCount: Array.from(activeRemoves).length,
              addedModels: activeAdds,
              removedModels: Array.from(activeRemoves),
            });
          } else {
            throw new Error(response.data.message);
          }
        }
      } catch (error) {
        setChannelStates((prev) => ({
          ...prev,
          [channel.id]: {
            ...prev[channel.id],
            status: CHANNEL_STATUS.FAILED,
            error: error.message,
          },
        }));

        summary.failed++;
        summary.details.push({
          channelId: channel.id,
          channelName: channel.name,
          success: false,
          error: error.message,
        });
      }

      completedCount++;
      setTaskProgress({ current: completedCount, total: totalCount });
    }

    setTaskSummary(summary);

    if (!shouldStopRef.current) {
      setTaskStatus(TASK_STATUS.COMPLETED);
      showSuccess(
        t(
          `批量模型更新完成！成功: ${summary.success}, 失败: ${summary.failed}`,
        ),
      );
    } else {
      setTaskStatus(TASK_STATUS.STOPPED);
    }
    setIsStoppable(false);
  };

  // 停止任务
  const stopTask = () => {
    shouldStopRef.current = true;
    setIsStoppable(false);
    showInfo(t('正在停止任务...'));
  };

  // 调整计划
  const toggleModelInPlan = (channelId, modelName, action) => {
    setPlanOverrides((prev) => {
      const channelOverrides = prev[channelId] || { add: {}, remove: {} };
      const actionOverrides = channelOverrides[action] || {};
      const currentState = actionOverrides[modelName] || 'active';
      const newState = currentState === 'active' ? 'inactive' : 'active';
      return {
        ...prev,
        [channelId]: {
          ...channelOverrides,
          [action]: {
            ...actionOverrides,
            [modelName]: newState,
          },
        },
      };
    });
  };

  // 切换渠道展开状态
  const toggleChannelExpanded = (channelId) => {
    setExpandedChannels((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(channelId)) {
        newSet.delete(channelId);
      } else {
        newSet.add(channelId);
      }
      return newSet;
    });
  };

  // 渲染渠道状态
  const renderChannelStatus = (channel) => {
    const statusMap = {
      [CHANNEL_STATUS.PENDING]: { color: 'grey', text: t('待处理') },
      [CHANNEL_STATUS.PLANNING]: { color: 'blue', text: t('规划中') },
      [CHANNEL_STATUS.PLAN_READY]: { color: 'green', text: t('计划就绪') },
      [CHANNEL_STATUS.PLAN_FAILED]: { color: 'red', text: t('规划失败') },
      [CHANNEL_STATUS.EXECUTING]: { color: 'orange', text: t('执行中') },
      [CHANNEL_STATUS.SUCCESS]: { color: 'green', text: t('成功') },
      [CHANNEL_STATUS.FAILED]: { color: 'red', text: t('失败') },
    };

    const status =
      statusMap[channel.status] || statusMap[CHANNEL_STATUS.PENDING];
    return <Tag color={status.color}>{status.text}</Tag>;
  };

  // 渲染模型标签，显示映射信息
  const renderModelTag = (model, channel, action, canEdit) => {
    const modelMapping = parseModelMapping(channel.model_mapping);
    const mappingEntry = Object.entries(modelMapping).find(
      ([mappedName]) => mappedName === model,
    );
    const isReverseMapped = Object.entries(modelMapping).some(
      ([, originalName]) => originalName === model,
    );
    const overrides = planOverrides[channel.id] || { add: {}, remove: {} };
    const currentOverride = overrides[action] || {};
    const state = currentOverride[model] || 'active';
    const isInactive = state === 'inactive';

    let tooltipContent = model;
    let tagColor = action === MODEL_ACTION.ADD ? 'green' : 'red';
    let displayText = model;

    if (mappingEntry) {
      const [mappedName, originalName] = mappingEntry;
      tooltipContent = `${mappedName} → ${originalName} (${t('映射模型')})`;
      tagColor = action === MODEL_ACTION.ADD ? 'blue' : 'orange'; // 映射模型使用不同颜色
      displayText = mappedName; // 确保显示映射模型名称，不是 "A→"
    } else if (isReverseMapped) {
      tooltipContent = `${model} (${t('被其他模型映射')})`;
      tagColor = action === MODEL_ACTION.ADD ? 'cyan' : 'purple'; // 底层模型使用不同颜色
    }

    const toggleState = () => {
      if (!canEdit) return;
      setPlanOverrides((prev) => {
        const channelOverrides = prev[channel.id] || { add: {}, remove: {} };
        const actionOverrides = channelOverrides[action] || {};
        const currentState = actionOverrides[model] || 'active';
        const newState = currentState === 'active' ? 'inactive' : 'active';
        return {
          ...prev,
          [channel.id]: {
            ...channelOverrides,
            [action]: {
              ...actionOverrides,
              [model]: newState,
            },
          },
        };
      });
    };

    return (
      <Tag
        key={`${action}_${model}`}
        color={tagColor}
        closable={false}
        onClick={toggleState}
        style={{
          margin: 2,
          cursor: canEdit ? 'pointer' : 'default',
          opacity: isInactive ? 0.4 : 1,
          textDecoration: isInactive ? 'line-through' : 'none',
        }}
      >
        {displayText}
      </Tag>
    );
  };

  // 渲染计划详情
  const renderPlanDetails = (channel) => {
    const plan = channelPlans[channel.id];
    if (!plan) return null;

    const isExpanded = expandedChannels.has(channel.id);
    const canEdit = taskStatus === TASK_STATUS.PLAN_REVIEW;
    const modelMapping = parseModelMapping(channel.model_mapping);
    const incompatibleMappings = checkModelMappingCompatibility(
      channel.currentModels || [],
      channel.availableModels || [],
      modelMapping,
    );

    // 根据更新模式决定显示哪些操作
    const shouldShowAdd =
      updateMode === 'add_new' || updateMode === 'full_update';
    const shouldShowRemove =
      updateMode === 'remove_invalid' || updateMode === 'full_update';

    // 生成动态的计划详情标题
    const getPlanDetailsTitle = () => {
      const parts = [];
      if (shouldShowAdd && plan.add.length > 0) {
        parts.push(t('新增: ${count}').replace('${count}', plan.add.length));
      }
      if (shouldShowRemove && plan.remove.length > 0) {
        parts.push(t('删除: ${count}').replace('${count}', plan.remove.length));
      }

      if (parts.length === 0) {
        return t('计划详情 (无操作)');
      }

      return t('计划详情 (${details})').replace('${details}', parts.join(', '));
    };

    // 检查是否需要更新
    const hasOperations =
      (shouldShowAdd && plan.add.length > 0) ||
      (shouldShowRemove && plan.remove.length > 0);

    return (
      <div style={{ marginTop: 8 }}>
        <div
          style={{ cursor: 'pointer', display: 'flex', alignItems: 'center' }}
          onClick={() => toggleChannelExpanded(channel.id)}
        >
          {isExpanded ? <IconChevronUp /> : <IconChevronDown />}
          <Text style={{ marginLeft: 4 }}>{getPlanDetailsTitle()}</Text>
          {incompatibleMappings.length > 0 && shouldShowRemove && (
            <IconAlertTriangle
              style={{ marginLeft: 8, color: 'orange', fontSize: '14px' }}
              title={t('发现 ${count} 个映射兼容性问题').replace(
                '${count}',
                incompatibleMappings.length,
              )}
            />
          )}
        </div>

        {isExpanded && (
          <div style={{ marginTop: 8, paddingLeft: 16 }}>
            {incompatibleMappings.length > 0 && shouldShowRemove && (
              <div
                style={{
                  marginBottom: 12,
                  padding: 8,
                  backgroundColor: '#fff7e6',
                  borderRadius: 4,
                }}
              >
                <Text strong style={{ color: '#fa8c16' }}>
                  {t('模型映射兼容性提醒:')}
                </Text>
                {incompatibleMappings.map((mapping, index) => (
                  <div
                    key={index}
                    style={{ marginTop: 4, fontSize: '12px', color: '#666' }}
                  >
                    • {mapping.mappedModel} → {mapping.targetModel}{' '}
                    {mapping.mappedAvailable
                      ? `(${t('原模型可用但映射模型不可用，请检查模型映射')})`
                      : `(${t('底层模型不可用，将被删除')})`}
                  </div>
                ))}
              </div>
            )}

            {shouldShowAdd && plan.add.length > 0 && (
              <div style={{ marginBottom: 8 }}>
                <Text strong>{t('新增模型:')}</Text>
                <div style={{ marginTop: 4 }}>
                  {plan.add.map((model) =>
                    renderModelTag(model, channel, MODEL_ACTION.ADD, canEdit),
                  )}
                </div>
              </div>
            )}

            {shouldShowRemove && plan.remove.length > 0 && (
              <div>
                <Text strong>{t('删除模型:')}</Text>
                <div style={{ marginTop: 4 }}>
                  {plan.remove.map((model) =>
                    renderModelTag(
                      model,
                      channel,
                      MODEL_ACTION.REMOVE,
                      canEdit,
                    ),
                  )}
                </div>
              </div>
            )}

            {!hasOperations && <Text type='quaternary'>{t('无需更新')}</Text>}

            {planOverrides[channel.id] && hasOperations && (
              <div style={{ marginTop: 8, fontSize: '12px', color: '#999' }}>
                {t('点击模型标签可切换激活状态')}
              </div>
            )}
          </div>
        )}
      </div>
    );
  };

  // 渲染任务摘要
  const renderTaskSummary = () => {
    if (!taskSummary) return null;

    return (
      <Card title={t('批量更新摘要')} style={{ marginTop: 16 }}>
        <div style={{ marginBottom: 16 }}>
          <Text>
            {t('成功: ${success} 个渠道').replace(
              '${success}',
              taskSummary.success,
            )}
          </Text>
          <br />
          <Text>
            {t('失败: ${failed} 个渠道').replace(
              '${failed}',
              taskSummary.failed,
            )}
          </Text>
        </div>

        <Collapse>
          <Collapse.Panel header={t('详细信息')} itemKey='details'>
            <List
              dataSource={taskSummary.details}
              renderItem={(detail) => (
                <List.Item>
                  <div style={{ width: '100%' }}>
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        marginBottom: 4,
                      }}
                    >
                      {detail.success ? (
                        <IconCheckCircleStroked
                          style={{ color: 'green', marginRight: 8 }}
                        />
                      ) : (
                        <IconAlertTriangle
                          style={{ color: 'red', marginRight: 8 }}
                        />
                      )}
                      <Text strong>{detail.channelName}</Text>
                    </div>

                    {detail.success ? (
                      <div>
                        {/* 根据更新模式显示不同的摘要信息 */}
                        {updateMode === 'add_new' && (
                          <Text>
                            {t('新增 ${added} 个模型').replace(
                              '${added}',
                              detail.addedCount,
                            )}
                          </Text>
                        )}
                        {updateMode === 'remove_invalid' && (
                          <Text>
                            {t('删除 ${removed} 个模型').replace(
                              '${removed}',
                              detail.removedCount,
                            )}
                          </Text>
                        )}
                        {updateMode === 'full_update' && (
                          <Text>
                            {t('新增 ${added} 个模型，删除 ${removed} 个模型')
                              .replace('${added}', detail.addedCount)
                              .replace('${removed}', detail.removedCount)}
                          </Text>
                        )}

                        {/* 根据更新模式显示相应的模型列表 */}
                        {(updateMode === 'add_new' ||
                          updateMode === 'full_update') &&
                          detail.addedModels.length > 0 && (
                            <div style={{ marginTop: 4 }}>
                              <Text type='secondary'>
                                {t('新增: ${models}').replace(
                                  '${models}',
                                  detail.addedModels.join(', '),
                                )}
                              </Text>
                            </div>
                          )}
                        {(updateMode === 'remove_invalid' ||
                          updateMode === 'full_update') &&
                          detail.removedModels.length > 0 && (
                            <div style={{ marginTop: 4 }}>
                              <Text type='secondary'>
                                {t('删除: ${models}').replace(
                                  '${models}',
                                  detail.removedModels.join(', '),
                                )}
                              </Text>
                            </div>
                          )}
                      </div>
                    ) : (
                      <Text type='danger'>
                        {t('错误: ${error}').replace('${error}', detail.error)}
                      </Text>
                    )}
                  </div>
                </List.Item>
              )}
            />
          </Collapse.Panel>
        </Collapse>
      </Card>
    );
  };

  // 渲染主要内容
  const renderMainContent = () => {
    if (taskStatus === TASK_STATUS.PLANNING) {
      return (
        <div>
          <div style={{ textAlign: 'center', marginBottom: 24 }}>
            <Spin size='large' />
            <div style={{ marginTop: 16 }}>
              <Text>
                {updateMode === 'remove_invalid' &&
                  t('正在获取渠道可用模型并生成删除计划...')}
                {updateMode === 'add_new' &&
                  t('正在获取渠道可用模型并生成添加计划...')}
                {updateMode === 'full_update' &&
                  t('正在获取渠道可用模型并生成更新计划...')}
              </Text>
            </div>
          </div>

          <Progress
            percent={calculateProgressPercent(
              taskProgress.current,
              taskProgress.total,
            )}
            showInfo
            format={() =>
              `${taskProgress.current || 0}/${taskProgress.total || 0}`
            }
          />

          <div style={{ marginTop: 16 }}>
            <List
              dataSource={Object.values(channelStates)}
              renderItem={(channel) => (
                <List.Item>
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      width: '100%',
                    }}
                  >
                    <Text>{channel.name}</Text>
                    {renderChannelStatus(channel)}
                  </div>
                  {channel.error && (
                    <div style={{ marginTop: 4 }}>
                      <Text type='danger' size='small'>
                        {channel.error}
                      </Text>
                    </div>
                  )}
                </List.Item>
              )}
            />
          </div>
        </div>
      );
    }

    if (taskStatus === TASK_STATUS.PLAN_REVIEW) {
      const readyChannels = Object.values(channelStates).filter(
        (channel) => channel.status === CHANNEL_STATUS.PLAN_READY,
      );
      const failedChannels = Object.values(channelStates).filter(
        (channel) => channel.status === CHANNEL_STATUS.PLAN_FAILED,
      );

      return (
        <div>
          <div style={{ marginBottom: 16 }}>
            <Text>
              {t(
                '计划生成完成！成功: ${success} 个渠道，失败: ${failed} 个渠道',
              )
                .replace('${success}', readyChannels.length)
                .replace('${failed}', failedChannels.length)}
            </Text>
          </div>

          {readyChannels.length > 0 && (
            <Card title={t('更新计划')} style={{ marginBottom: 16 }}>
              <Text type='secondary' size='small'>
                {updateMode === 'add_new' &&
                  t('点击模型标签可以取消添加操作，再次点击可以恢复')}
                {updateMode === 'remove_invalid' &&
                  t('点击模型标签可以取消删除操作，再次点击可以恢复')}
                {updateMode === 'full_update' &&
                  t('点击模型标签可以取消对应操作，再次点击可以恢复')}
              </Text>
              <List
                dataSource={readyChannels}
                renderItem={(channel) => (
                  <List.Item>
                    <div style={{ width: '100%' }}>
                      <div
                        style={{
                          display: 'flex',
                          justifyContent: 'space-between',
                          alignItems: 'center',
                        }}
                      >
                        <Text strong>{channel.name}</Text>
                        {renderChannelStatus(channel)}
                      </div>
                      {renderPlanDetails(channel)}
                    </div>
                  </List.Item>
                )}
              />
            </Card>
          )}

          {failedChannels.length > 0 && (
            <Card title={t('规划失败的渠道')} style={{ marginBottom: 16 }}>
              <List
                dataSource={failedChannels}
                renderItem={(channel) => (
                  <List.Item>
                    <div style={{ width: '100%' }}>
                      <div
                        style={{
                          display: 'flex',
                          justifyContent: 'space-between',
                          alignItems: 'center',
                        }}
                      >
                        <Text>{channel.name}</Text>
                        {renderChannelStatus(channel)}
                      </div>
                      {channel.error && (
                        <Text type='danger' size='small'>
                          {channel.error}
                        </Text>
                      )}
                    </div>
                  </List.Item>
                )}
              />
            </Card>
          )}
        </div>
      );
    }

    if (taskStatus === TASK_STATUS.EXECUTING) {
      return (
        <div>
          <div style={{ textAlign: 'center', marginBottom: 24 }}>
            <Spin size='large' />
            <div style={{ marginTop: 16 }}>
              <Text>
                {updateMode === 'remove_invalid' && t('正在删除无效模型...')}
                {updateMode === 'add_new' && t('正在添加新模型...')}
                {updateMode === 'full_update' && t('正在执行模型更新...')}
              </Text>
            </div>
          </div>

          <Progress
            percent={calculateProgressPercent(
              taskProgress.current,
              taskProgress.total,
            )}
            showInfo
            format={() =>
              `${taskProgress.current || 0}/${taskProgress.total || 0}`
            }
          />

          <div style={{ marginTop: 16 }}>
            <List
              dataSource={Object.values(channelStates).filter(
                (channel) => channel.status !== CHANNEL_STATUS.PLAN_FAILED,
              )}
              renderItem={(channel) => (
                <List.Item>
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      width: '100%',
                    }}
                  >
                    <Text>{channel.name}</Text>
                    {renderChannelStatus(channel)}
                  </div>
                  {channel.error && (
                    <div style={{ marginTop: 4 }}>
                      <Text type='danger' size='small'>
                        {channel.error}
                      </Text>
                    </div>
                  )}
                </List.Item>
              )}
            />
          </div>
        </div>
      );
    }

    if (
      taskStatus === TASK_STATUS.COMPLETED ||
      taskStatus === TASK_STATUS.STOPPED ||
      taskStatus === TASK_STATUS.ERROR
    ) {
      return (
        <div>
          <div style={{ textAlign: 'center', marginBottom: 24 }}>
            <IconCheckCircleStroked
              size='large'
              style={{
                color: taskStatus === TASK_STATUS.ERROR ? 'red' : 'green',
              }}
            />
            <div style={{ marginTop: 16 }}>
              <Text>
                {taskStatus === TASK_STATUS.COMPLETED
                  ? t('批量模型更新完成！')
                  : taskStatus === TASK_STATUS.ERROR
                    ? t('任务执行错误')
                    : t('任务已停止')}
              </Text>
            </div>
          </div>

          {renderTaskSummary()}
        </div>
      );
    }

    return <Empty description={t('无内容')} />;
  };

  // 渲染操作按钮
  const renderActionButtons = () => {
    if (
      taskStatus === TASK_STATUS.PLANNING ||
      taskStatus === TASK_STATUS.EXECUTING
    ) {
      return (
        <Space>
          <Button
            type='danger'
            icon={<IconClose />}
            disabled={!isStoppable}
            onClick={stopTask}
          >
            {t('停止任务')}
          </Button>
          <Button onClick={onCancel}>{t('关闭')}</Button>
        </Space>
      );
    }

    if (taskStatus === TASK_STATUS.PLAN_REVIEW) {
      const readyChannels = Object.values(channelStates).filter(
        (channel) => channel.status === CHANNEL_STATUS.PLAN_READY,
      );

      return (
        <Space>
          <Button
            type='primary'
            icon={<IconPlay />}
            disabled={readyChannels.length === 0}
            onClick={executeImplementationPhase}
          >
            {t('开始执行')}
          </Button>
          <Button onClick={onCancel}>{t('取消')}</Button>
        </Space>
      );
    }

    if (
      taskStatus === TASK_STATUS.COMPLETED ||
      taskStatus === TASK_STATUS.STOPPED
    ) {
      return (
        <Button
          type='primary'
          icon={<IconRefresh />}
          onClick={async () => {
            onRefresh && onRefresh(); // Refresh parent component
            onCancel();
          }}
        >
          {t('刷新页面')}
        </Button>
      );
    }

    return <Button onClick={onCancel}>{t('关闭')}</Button>;
  };

  // 获取标题
  const getTitle = () => {
    // 根据更新模式获取基础标题
    const getBaseTitleByMode = () => {
      switch (updateMode) {
        case 'remove_invalid':
          return t('删除无效模型');
        case 'add_new':
          return t('添加新模型');
        case 'full_update':
        default:
          return t('批量模型更新');
      }
    };

    const baseTitle = getBaseTitleByMode();

    const statusTitles = {
      [TASK_STATUS.PLANNING]: `${baseTitle} - ${t('计划阶段')}`,
      [TASK_STATUS.PLAN_REVIEW]: `${baseTitle} - ${t('计划确认')}`,
      [TASK_STATUS.EXECUTING]: `${baseTitle} - ${t('执行阶段')}`,
      [TASK_STATUS.COMPLETED]: `${baseTitle} - ${t('完成')}`,
      [TASK_STATUS.STOPPED]: `${baseTitle} - ${t('已停止')}`,
      [TASK_STATUS.ERROR]: `${baseTitle} - ${t('错误')}`,
    };

    return statusTitles[taskStatus] || baseTitle;
  };

  return (
    <Modal
      title={getTitle()}
      visible={visible}
      onCancel={onCancel}
      footer={renderActionButtons()}
      width={800}
      style={{ maxHeight: '80vh' }}
      bodyStyle={{ maxHeight: '60vh', overflowY: 'auto' }}
      maskClosable={false}
      closable={
        taskStatus !== TASK_STATUS.PLANNING &&
        taskStatus !== TASK_STATUS.EXECUTING
      }
      afterOpen={() => {
        // Modal opened
      }}
    >
      {renderMainContent()}
    </Modal>
  );
};

export default BatchModelUpdateModal;
