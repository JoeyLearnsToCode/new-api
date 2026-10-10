import React from 'react';
import { useActualTheme, useSetTheme } from '@/context/Theme';
import { MinimalButton } from '@/prompt-studio/components/common/MinimalButton';
import { duration, ease } from '@/prompt-studio/styles/motion';

/**
 * 主题切换
 * 嵌入 new-api 后统一切换全局主题（同时影响控制台与提示词工坊）
 */
export const ThemeToggle: React.FC = () => {
  const actualTheme = useActualTheme();
  const setTheme = useSetTheme();
  const isDark = actualTheme === 'dark';

  const toggleTheme = () => {
    setTheme(isDark ? 'light' : 'dark');
  };

  // 两个图标叠放做交叉淡入 + 旋转，而不是瞬间换一个字形
  const layerTransition = {
    transitionDuration: `${duration.fast * 1000}ms`,
    transitionTimingFunction: `cubic-bezier(${ease.outExpo.join(',')})`,
  };
  const iconState = (visible: boolean) => ({
    opacity: visible ? 1 : 0,
    transform: `rotate(${visible ? 0 : -90}deg) scale(${visible ? 1 : 0.6})`,
  });

  return (
    <MinimalButton
      variant='ghost'
      onClick={toggleTheme}
      className='relative h-9 w-9 overflow-hidden !text-white/90 !hover:text-white hover:bg-white/10'
      title='Toggle Theme'
      aria-label='Toggle Theme'
    >
      <span
        className='absolute inset-0 flex items-center justify-center transition-[opacity,transform]'
        style={{ ...iconState(!isDark), ...layerTransition }}
      >
        <span
          className='material-symbols-outlined'
          style={{ fontSize: '20px' }}
        >
          light_mode
        </span>
      </span>
      <span
        className='absolute inset-0 flex items-center justify-center transition-[opacity,transform]'
        style={{ ...iconState(isDark), ...layerTransition }}
      >
        <span
          className='material-symbols-outlined'
          style={{ fontSize: '20px' }}
        >
          dark_mode
        </span>
      </span>
    </MinimalButton>
  );
};
