import React from 'react';
import { useActualTheme, useSetTheme } from '@/context/Theme';
import { MinimalButton } from '@/prompt-studio/components/common/MinimalButton';

/**
 * 主题切换
 * 嵌入 new-api 后统一切换全局主题（同时影响控制台与提示词工坊）
 */
export const ThemeToggle: React.FC = () => {
  const actualTheme = useActualTheme();
  const setTheme = useSetTheme();

  const toggleTheme = () => {
    setTheme(actualTheme === 'dark' ? 'light' : 'dark');
  };

  return (
    <MinimalButton
      variant='ghost'
      onClick={toggleTheme}
      className='h-9 w-9 !text-white/90 !hover:text-white hover:bg-white/10'
      title='Toggle Theme'
      aria-label='Toggle Theme'
    >
      <span className='material-symbols-outlined' style={{ fontSize: '20px' }}>
        {'contrast'}
      </span>
    </MinimalButton>
  );
};
