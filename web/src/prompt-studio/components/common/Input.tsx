import React from 'react';

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
}

export const Input: React.FC<InputProps> = ({
  label,
  error,
  className = '',
  ...props
}) => {
  return (
    <div className='flex flex-col gap-1'>
      {label && (
        <label className='text-sm font-medium text-surface-onSurface dark:text-surface-onSurfaceDark'>
          {label}
        </label>
      )}
      {/* 
        修复：暗色模式下使用 background.dark (Zinc 950) 作为输入框背景，形成凹陷感，
        避免使用 bright white 或与 modal 相同的颜色 
      */}
      <input
        className={`px-4 py-2 border-2 rounded-m3-small 
          bg-surface-variant dark:bg-background-dark 
          text-surface-onVariant dark:text-surface-onSurfaceDark
          border-surface-variant dark:border-border-dark
          focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/20
          disabled:opacity-50 disabled:cursor-not-allowed
          ${error ? 'border-error' : ''}
          ${className}`}
        {...props}
      />
      {error && <span className='text-sm text-error'>{error}</span>}
    </div>
  );
};
