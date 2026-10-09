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

/**
 * 提示词工坊（由外部仓库 prompt-studio 移植）
 *
 * - 宿主路由：/prompt-studio，已由 new-api 的 PrivateRoute 保证登录态
 * - 页面内部仍沿用原应用的 hash 路由：#/ 、#/project/:id 、#/settings
 * - 数据全部读写 new-api 后端接口，浏览器 localStorage 只用于偏好与首屏快照缓存
 */

import React, { useEffect, useState } from 'react';
import { AppInitializer } from '@/prompt-studio/components/AppInitializer';
import { I18nProvider } from '@/prompt-studio/i18n/I18nContext';
import MainView from '@/prompt-studio/pages/MainView';
import Settings from '@/prompt-studio/pages/Settings';
import '@/prompt-studio/styles/globals.css';

/** 页面内部的 hash 路由，与原 prompt-studio 保持一致 */
const useHashRoute = (): string => {
  const [hash, setHash] = useState(() => window.location.hash || '#/');

  useEffect(() => {
    const onChange = () => setHash(window.location.hash || '#/');
    window.addEventListener('hashchange', onChange);
    return () => window.removeEventListener('hashchange', onChange);
  }, []);

  return hash;
};

const PromptStudio: React.FC = () => {
  const hash = useHashRoute();

  return (
    <div className='prompt-studio-root w-full overflow-hidden'>
      <I18nProvider>
        <AppInitializer>
          {hash.startsWith('#/settings') ? <Settings /> : <MainView />}
        </AppInitializer>
      </I18nProvider>
    </div>
  );
};

export default PromptStudio;
